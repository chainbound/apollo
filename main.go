package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path"
	"time"

	_ "embed"

	"github.com/XMonetae-DeFi/apollo/chainservice"
	"github.com/XMonetae-DeFi/apollo/db"
	"github.com/XMonetae-DeFi/apollo/dsl"
	"github.com/XMonetae-DeFi/apollo/log"
	"github.com/XMonetae-DeFi/apollo/output"
	"github.com/XMonetae-DeFi/apollo/types"
	"github.com/rs/zerolog"

	"github.com/urfave/cli/v2"
)

//go:embed config.yml
var cfg []byte

//go:embed schema.example.hcl
var schema []byte

var logger zerolog.Logger

func main() {
	var opts types.ApolloOpts
	logger = log.NewLogger("main")

	app := &cli.App{
		Name:  "apollo",
		Usage: "Run the chain analyzer",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "realtime",
				Aliases:     []string{"R"},
				Usage:       "Run apollo in realtime",
				Destination: &opts.Realtime,
			},
			&cli.BoolFlag{
				Name:        "db",
				Usage:       "Save results in database",
				Destination: &opts.Db,
			},
			&cli.BoolFlag{
				Name:        "csv",
				Usage:       "Save results in csv file",
				Destination: &opts.Csv,
			},
			&cli.BoolFlag{
				Name:        "stdout",
				Usage:       "Print to stdout",
				Destination: &opts.Stdout,
			},
			&cli.Int64Flag{
				Name:        "interval",
				Aliases:     []string{"i"},
				Usage:       "Interval in `BLOCKS` or SECONDS (realtime: seconds, historic: blocks)",
				Destination: &opts.Interval,
			},
			&cli.Int64Flag{
				Name:        "time-interval",
				Aliases:     []string{"I"},
				Usage:       "Interval in seconds",
				Destination: &opts.TimeInterval,
			},
			&cli.Int64Flag{
				Name:        "start-block",
				Usage:       "Starting block number for historical analysis",
				Destination: &opts.StartBlock,
			},
			&cli.Int64Flag{
				Name:        "end-block",
				Usage:       "End block number for historical analysis",
				Destination: &opts.EndBlock,
			},
			&cli.Int64Flag{
				Name:        "start-time",
				Usage:       "Start timestamp (UNIX) for historical analysis",
				Destination: &opts.StartTime,
			},
			&cli.Int64Flag{
				Name:        "end-time",
				Usage:       "End timestamp (UNIX) for historical analysis",
				Destination: &opts.EndTime,
			},
			&cli.IntFlag{
				Name:        "rate-limit",
				Usage:       "Rate limit `LEVEL`, from 1 - 5",
				Destination: &opts.RateLimit,
			},
			&cli.IntFlag{
				Name:        "log-level",
				Usage:       "log level from -1 to 5",
				Destination: &opts.LogLevel,
				Value:       1,
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Initialize apollo by creating the configs",
				Action: func(c *cli.Context) error {
					if err := Init(); err != nil {
						return err
					}

					return nil
				},
			},
		},
		Action: func(c *cli.Context) error {
			err := Run(opts)
			return err
		},
	}

	err := app.Run(os.Args)

	if err != nil {
		logger.Fatal().Err(err).Msg("running app")
	}
}

func Init() error {
	p, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	dirPath := path.Join(p, "apollo")
	_, err = os.Stat(dirPath)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dirPath, 0744)
		if err != nil {
			return err
		}
	}

	configPath := path.Join(dirPath, "config.yml")
	if err := os.WriteFile(configPath, cfg, 0644); err != nil {
		return err
	}
	fmt.Println("config written", configPath)

	schemaPath := path.Join(dirPath, "schema.hcl")
	if err := os.WriteFile(schemaPath, schema, 0644); err != nil {
		return err
	}
	fmt.Println("schema written", schemaPath)

	return nil
}

func Run(opts types.ApolloOpts) error {
	lvl := zerolog.Level(int8(opts.LogLevel))
	logger.Info().Int("log_level", int(lvl)).Msg("logger")
	zerolog.SetGlobalLevel(lvl)

	var pdb *db.DB

	confDir, err := ConfigDir()
	if err != nil {
		return err
	}

	confPath, err := ConfigPath()
	if err != nil {
		return err
	}

	cfg, err := NewConfig(confPath)
	if err != nil {
		return err
	}

	schema, err := dsl.NewSchema(confDir)
	if err != nil {
		return err
	}

	// Validate the schema
	if err := schema.Validate(opts); err != nil {
		return err
	}

	cfg.DbSettings.DefaultTimeout = time.Second * 20

	if opts.Db {
		pdb, err = db.NewDB(cfg.DbSettings).Connect()
		if err != nil {
			return err
		}

		logger.Debug().Str("db", pdb.Settings.Name).Msg("connected to db")
	}

	rpc, ok := cfg.Rpc[schema.Chain]
	if !ok {
		return fmt.Errorf("no rpc defined for chain %s", opts.Chain)
	}

	defaultTimeout := time.Second * 30

	// Long timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	service, err := chainservice.NewChainService(defaultTimeout, opts.RateLimit).Connect(ctx, rpc)
	if err != nil {
		return err
	}

	if opts.StartBlock == 0 && opts.StartTime != 0 {
		opts.StartBlock, err = service.BlockByTimestamp(ctx, opts.StartTime)
		if err != nil {
			return err
		}
	}

	if opts.EndBlock == 0 && opts.EndTime != 0 {
		opts.EndBlock, err = service.BlockByTimestamp(ctx, opts.EndTime)
		if err != nil {
			return err
		}
	}

	if opts.Interval == 0 && opts.TimeInterval != 0 {
		opts.Interval, err = service.SecondsToBlockInterval(ctx, opts.TimeInterval)
		if err != nil {
			return err
		}
	}

	out := output.NewOutputHandler()

	if opts.Db {
		out = out.WithDB(pdb)
	}

	if opts.Csv {
		out = out.WithCsv(output.NewCsvHandler())
	}

	if opts.Stdout {
		out = out.WithStdOut()
	}

	// First check if there are any methods to be called, it might just be events
	maxWorkers := 32
	blocks := make(chan *big.Int)
	chainResults := make(chan types.CallResult)

	service.RunMethodCaller(schema, opts.Realtime, blocks, chainResults, maxWorkers)

	// Start main program loop
	if opts.Realtime {
		go func() {
			for {
				blocks <- nil
				time.Sleep(time.Duration(opts.Interval) * time.Second)
			}
		}()
	} else {
		go func() {
			for i := opts.StartBlock; i < opts.EndBlock; i += opts.Interval {
				blocks <- big.NewInt(i)
			}

			close(blocks)
		}()
	}

	if opts.Realtime {
		service.ListenForEvents(schema, chainResults, maxWorkers)
	} else {
		service.FilterEvents(schema, big.NewInt(opts.StartBlock), big.NewInt(opts.EndBlock), chainResults, maxWorkers)
	}

	for res := range chainResults {
		if res.Err != nil {
			logger.Warn().Msg(res.Err.Error())
			continue
		}

		save, err := schema.EvaluateSaveBlock(res.ContractName, dsl.GenerateVarMap(res))
		if err != nil {
			return fmt.Errorf("evaluating save block: %w", err)
		}

		err = out.HandleResult(res.ContractName, save)
		if err != nil {
			return fmt.Errorf("handling result: %w", err)
		}
	}

	return nil
}
