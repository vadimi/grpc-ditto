package main

import (
	"os"

	"github.com/vadimi/grpc-ditto/internal/logger"

	"github.com/urfave/cli"
)

func main() {
	logger.Init("debug")
	log := logger.NewLogger()

	app := cli.NewApp()
	app.Version = "1.0.0"
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
		cli.IntFlag{
			Name:     "port,p",
			Required: false,
			Usage:    "grpc server port",
			Value:    51000,
		},
	}
	app.Action = mockCmd
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
