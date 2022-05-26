package main

import (
	"github.com/chainbound/apollo/types"
	"github.com/urfave/cli/v2"
)

func BuildFlags(opts *types.ApolloOpts) []cli.Flag {
	return []cli.Flag{
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
		&cli.IntFlag{
			Name:        "rate-limit",
			Usage:       "Rate limit `RPS` in max requests per second",
			Destination: &opts.RateLimit,
			Value:       100,
		},
		&cli.IntFlag{
			Name:        "log-level",
			Usage:       "log level from -1 to 5",
			Destination: &opts.LogLevel,
			Value:       1,
		},
	}
}
