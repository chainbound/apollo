package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"path"
	"time"

	_ "embed"

	"github.com/XMonetae-DeFi/apollo/chainservice"
	"github.com/XMonetae-DeFi/apollo/db"
	"github.com/XMonetae-DeFi/apollo/dsl"
	"github.com/XMonetae-DeFi/apollo/output"
	"github.com/urfave/cli/v2"
)

// Main program options, provided as cli arguments
type ApolloOpts struct {
	realtime   bool
	db         bool
	csv        bool
	stdout     bool
	interval   int64
	startBlock int64
	endBlock   int64
	rateLimit  int
	chain      string
}

//go:embed config.yml
var cfg []byte

//go:embed schema.hcl
var schema []byte

func main() {
	var opts ApolloOpts

	app := &cli.App{
		Name:  "apollo",
		Usage: "Run the chain analyzer",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "realtime",
				Aliases:     []string{"R"},
				Usage:       "Run apollo in realtime",
				Destination: &opts.realtime,
			},
			&cli.BoolFlag{
				Name:        "db",
				Usage:       "Save results in database",
				Destination: &opts.db,
			},
			&cli.BoolFlag{
				Name:        "csv",
				Usage:       "Save results in csv file",
				Destination: &opts.csv,
			},
			&cli.BoolFlag{
				Name:        "stdout",
				Usage:       "Print to stdout",
				Destination: &opts.stdout,
			},
			&cli.Int64Flag{
				Name:        "interval",
				Aliases:     []string{"i"},
				Usage:       "Interval in `BLOCKS` or SECONDS (realtime: seconds, historic: blocks)",
				Destination: &opts.interval,
			},
			&cli.Int64Flag{
				Name:        "start-block",
				Aliases:     []string{"s"},
				Usage:       "Starting block number for historical analysis",
				Destination: &opts.startBlock,
			},
			&cli.Int64Flag{
				Name:        "end-block",
				Aliases:     []string{"e"},
				Usage:       "End block number for historical analysis",
				Destination: &opts.endBlock,
			},
			&cli.IntFlag{
				Name:        "rate-limit",
				Usage:       "Rate limit `LEVEL`, from 1 - 5",
				Destination: &opts.rateLimit,
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
		log.Fatal(err)
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

func Run(opts ApolloOpts) error {
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
	// TODO
	// if err := schema.Validate(); err != nil {
	// 	log.Fatal(err)
	// }

	cfg.DbSettings.DefaultTimeout = time.Second * 20

	if opts.db {
		pdb, err = db.NewDB(cfg.DbSettings).Connect()

		if err != nil {
			return err
		}
	}

	rpc, ok := cfg.Rpc[schema.Chain]
	if !ok {
		return fmt.Errorf("no rpc defined for chain %s", opts.chain)
	}

	defaultTimeout := time.Second * 30

	// Long timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	service, err := chainservice.NewChainService(defaultTimeout, opts.rateLimit).Connect(ctx, rpc)
	if err != nil {
		return err
	}

	out := output.NewOutputHandler()

	if opts.db {
		out = out.WithDB(pdb)
	}

	if opts.csv {
		out = out.WithCsv(output.NewCsvHandler())
	}

	if opts.stdout {
		out = out.WithStdOut()
	}

	// First check if there are any methods to be called, it might just be events
	maxWorkers := 32
	blocks := make(chan *big.Int)
	chainResults := make(chan chainservice.EvaluationResult)

	service.RunMethodCaller(schema, opts.realtime, blocks, chainResults, maxWorkers)

	// Start main program loop
	if opts.realtime {
		go func() {
			for {
				blocks <- nil
				time.Sleep(time.Duration(opts.interval) * time.Second)
			}
		}()
	} else {
		go func() {
			for i := opts.startBlock; i < opts.endBlock; i += opts.interval {
				blocks <- big.NewInt(i)
			}

			close(blocks)
		}()
	}

	if opts.realtime {
		service.ListenForEvents(schema, chainResults, maxWorkers)
	} else {
		service.FilterEvents(schema, big.NewInt(opts.startBlock), big.NewInt(opts.endBlock), chainResults, maxWorkers)
	}

	for res := range chainResults {
		if res.Err != nil {
			fmt.Println(res.Err)
			continue
		}

		out.HandleResult(res.Name, res.Res)
	}

	return nil
}
