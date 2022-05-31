package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	_ "embed"

	"github.com/chainbound/apollo/chainservice"
	"github.com/chainbound/apollo/db"
	"github.com/chainbound/apollo/dsl"
	"github.com/chainbound/apollo/log"
	"github.com/chainbound/apollo/output"
	"github.com/chainbound/apollo/types"
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
		Flags: BuildFlags(&opts),
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
	}

	defaultTimeout := time.Second * 60

	// Long timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	service := chainservice.NewChainService(defaultTimeout, opts.RateLimit, cfg.Rpc)

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
	chainResults := make(chan types.CallResult)

	go func() error {
		err := service.Start(ctx, schema, opts, chainResults)
		if err != nil {
			return err
		}

		return nil
	}()

	for res := range chainResults {
		if res.Err != nil {

			logger.Warn().Str("chain", string(res.Chain)).Msg(res.Err.Error())
			continue
		}

		save, err := schema.EvalSave(service, res)
		if err != nil {
			return fmt.Errorf("evaluating save block: %w", err)
		}

		// Result got filtered out
		if save == nil {
			continue
		}

		err = out.HandleResult(res.QueryName, save)
		if err != nil {
			return fmt.Errorf("handling result: %w", err)
		}
	}

	service.DumpMetrics()

	return nil
}
