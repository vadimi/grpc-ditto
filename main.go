package main

import (
	"grpc-ditto/internal/logger"
	"os"

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
