package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/XMonetae-DeFi/apollo/chainservice"
	"github.com/XMonetae-DeFi/apollo/db"
	"github.com/XMonetae-DeFi/apollo/generate"
	"github.com/XMonetae-DeFi/apollo/output"
	"github.com/urfave/cli/v2"
)

const (
	rpcUrl = "wss://arb-mainnet.g.alchemy.com/v2/5_JWUuiS1cewWFpLzRxdjgZM0yLA4Uqp"
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
				Required:    true,
				Destination: &opts.chain,
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

type OutputHandler interface {
	HandleResult(chainservice.CallResult) error
}

func Run(opts ApolloOpts) error {
	var (
		pdb *db.DB
	)

	cfg, err := NewConfig("config.yml")
	if err != nil {
		return err
	}

	schema, err := generate.ParseV2("schema.v2.yml")
	if err != nil {
		return err
	}

	if opts.db {
		pdb, err = db.NewDB(cfg.DbSettings).Connect()

		if err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
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

	blocks := make(chan *big.Int)
	res := service.RunMethodCaller(context.Background(), schema, opts.realtime, blocks)

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
			b := new(big.Int)
			for i := opts.startBlock; i < opts.endBlock; i += opts.interval {
				b = b.SetInt64(i)
				blocks <- b
				time.Sleep(100 * time.Millisecond)
			}

			close(blocks)
		}()
	}

	// We can differentiate between realtime results and historical results
	// by checking the BlockNumber field (realtime: BlockNumber = nil)
	for res := range res {
		if res.Err != nil {
			fmt.Println(res.Err)
			continue
		}

		fmt.Println(res)

		out.HandleResult(res)
	}

	return nil
}
