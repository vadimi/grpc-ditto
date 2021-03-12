package main

import (
	"log"
	"os"

	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()
	app.Version = "0.8.1"
	app.Usage = "grpc mocking server"
	app.Flags = []cli.Flag{
		cli.StringSliceFlag{
			Name:     "proto",
			Required: true,
			Usage:    "proto files input directory",
		},
		cli.StringSliceFlag{
			Name:     "protoimports",
			Required: false,
			Usage:    "additional directories to search for dependencies",
		},
		cli.StringFlag{
			Name:     "mocks",
			Required: true,
			Usage:    "directory containing mocks in json format",
		},
		cli.StringFlag{
			Name:     "loglevel,l",
			Required: false,
			Value:    "debug",
			Usage:    "log level",
		},
		cli.IntFlag{
			Name:     "port,p",
			Required: false,
			Usage:    "grpc server port",
			Value:    51000,
		},
	}

	app.Action = newMockCmd()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
