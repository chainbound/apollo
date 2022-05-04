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
	"github.com/XMonetae-DeFi/apollo/generate"
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
	chain      string
}

//go:embed config.yml
var cfg []byte

//go:embed schema.v2.yml
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
			&cli.StringFlag{
				Name:        "chain",
				Aliases:     []string{"c"},
				Usage:       "The chain name",
				Destination: &opts.chain,
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

	validateOpts(opts)

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func validateOpts(opts ApolloOpts) {
	switch {
	case opts.endBlock != 0:
		if opts.interval == 0 {
			log.Fatal("need interval for historical mode")
		}
	}
}

type OutputHandler interface {
	HandleResult(chainservice.CallResult) error
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

	schemaPath := path.Join(dirPath, "schema.yml")
	if err := os.WriteFile(schemaPath, schema, 0644); err != nil {
		return err
	}

	return nil
}

func Run(opts ApolloOpts) error {
	var pdb *db.DB

	confDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	confDir = path.Join(confDir, "apollo")
	confPath := path.Join(confDir, "config.yml")

	cfg, err := NewConfig(confPath)
	if err != nil {
		return err
	}

	schema, err := generate.ParseV2(confDir)
	if err != nil {
		return err
	}

	// Validate the schema
	if err := schema.Validate(); err != nil {
		log.Fatal(err)
	}

	if opts.db {
		pdb, err = db.NewDB(cfg.DbSettings).Connect()

		if err != nil {
			return err
		}
	}

	// Long timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	rpc, ok := cfg.Rpc[opts.chain]
	if !ok {
		return fmt.Errorf("no rpc defined for chain %s", opts.chain)
	}

	service, err := chainservice.NewChainService().Connect(ctx, rpc)
	if err != nil {
		return err
	}

	csv := output.NewCsvHandler()

	for _, s := range schema.Contracts {
		if opts.db {
			err = pdb.CreateTable(ctx, *s)
			if err != nil {
				log.Fatal(err)
			}
		}

		if opts.csv {
			err = csv.AddCsv(*s)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	out := output.NewOutputHandler()

	if opts.db {
		out = out.WithDB(pdb)
	}

	if opts.csv {
		out = out.WithCsv(csv)
	}

	if opts.stdout {
		out = out.WithStdOut()
	}

	// First check if there are any methods to be called, it might just be events
	maxWorkers := 16
	blocks := make(chan *big.Int)
	chainResults := make(chan chainservice.CallResult)

	service.RunMethodCaller(context.Background(), schema, opts.realtime, blocks, chainResults, maxWorkers)

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

	// Long timeout
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*50)
	defer cancel()

	if opts.realtime {
		fmt.Println("todo")
	} else {
		service.FilterEvents(ctx, schema, big.NewInt(opts.startBlock), big.NewInt(opts.endBlock), chainResults, maxWorkers)
	}

	for res := range chainResults {
		if res.Err != nil {
			fmt.Println(res.Err)
			continue
		}

		out.HandleResult(res)
	}

	return nil
}
