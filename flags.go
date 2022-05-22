package main

import (
	"github.com/XMonetae-DeFi/apollo/types"
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
	}
}
