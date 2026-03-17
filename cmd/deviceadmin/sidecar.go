package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/urfave/cli/v3"

	"github.com/openUC2/device-admin/internal/app/deviceadmin"
)

var sidecarCmd = &cli.Command{
	Name:   "sidecar",
	Action: sidecarMain,
	Flags: []cli.Flag{
		&cli.DurationFlag{
			Name:    "shutdown-timeout",
			Value:   5 * time.Second,
			Usage:   "timeout for graceful shutdown before hard shutdown",
			Sources: cli.EnvVars("SERVER_SHUTDOWNTIMEOUT"),
		},
	},
}

func sidecarMain(ctx context.Context, cmd *cli.Command) error {
	e := echo.New() // TODO: get rid of this by using a more standard logging interface
	e.Logger.SetLevel(log.INFO)

	// Prepare sidecar
	sidecar, err := deviceadmin.NewSidecar(e.Logger)
	if err != nil {
		return err
	}

	// Run sidecar
	ctxRun, cancelRun := signal.NotifyContext(
		ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT,
	)
	go func() {
		if err = sidecar.Run(ctxRun); err != nil {
			e.Logger.Error(err.Error())
		}
		cancelRun()
	}()
	<-ctxRun.Done()
	cancelRun()

	// Shut down sidecar
	shutdownTimeout := cmd.Duration("shutdown-timeout")
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()
	e.Logger.Infof("attempting to shut down gracefully within %.1f sec", shutdownTimeout.Seconds())
	if err := sidecar.Shutdown(ctxShutdown); err != nil {
		return err
	}
	e.Logger.Info("finished shutdown")
	return nil
}
