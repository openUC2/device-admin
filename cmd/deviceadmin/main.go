package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

var cmd = &cli.Command{
	Name:  "device-admin",
	Usage: "Provides a web browser interface for system administration",
	Commands: []*cli.Command{
		serverCmd,
		sidecarCmd,
	},
}
