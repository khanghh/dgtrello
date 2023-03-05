package main

import (
	"github.com/urfave/cli/v2"
)

var (
	configFileFlag = &cli.StringFlag{
		Name:  "config",
		Usage: "JSON config file",
		Value: "config.json",
	}
	verbosityFlag = &cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
	}
)
