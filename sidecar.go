package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/urfave/cli/v3"

	"github.com/openUC2/device-admin/internal/app/sidecar"
)

var sidecarCmd = &cli.Command{
	Name:   "sidecar",
	Action: sidecarMain,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "address",
			Value:   "tcp:127.0.0.1:2312",
			Usage:   "address of varlink service",
			Sources: cli.EnvVars("SIDECAR_ADDRESS"),
		},
	},
}

var config = sidecar.Config{
	Vendor:  "OpenUC2",
	Product: "device-admin sidecar",
	URL:     "https://github.com/openUC2/device-admin",
}

func sidecarMain(ctx context.Context, cmd *cli.Command) error {
	e := echo.New() // TODO: get rid of this by using a more standard logging interface
	e.Logger.SetLevel(log.INFO)

	// Prepare sidecar
	config.Version = toolVersion
	config.Address = cmd.String("address")
	s, err := sidecar.New(config, e.Logger)
	if err != nil {
		return err
	}

	// Run sidecar
	ctxRun, cancelRun := signal.NotifyContext(
		ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT,
	)
	go func() {
		if err = s.Run(ctxRun); err != nil {
			e.Logger.Error(err.Error())
		}
		cancelRun()
	}()
	<-ctxRun.Done()
	cancelRun()

	e.Logger.Info("finished shutdown")
	return nil
}
