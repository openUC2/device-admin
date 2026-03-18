package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/urfave/cli/v3"

	"github.com/openUC2/device-admin/internal/app/server"
	"github.com/openUC2/device-admin/internal/app/server/conf"
)

const defaultShutdownTimeout = 5 * time.Second

var serverCmd = &cli.Command{
	Name:   "server",
	Action: serverMain,
	Flags: []cli.Flag{
		&cli.DurationFlag{
			Name:    "shutdown-timeout",
			Value:   defaultShutdownTimeout,
			Usage:   "timeout for graceful shutdown before hard shutdown",
			Sources: cli.EnvVars("SHUTDOWNTIMEOUT"),
		},
		&cli.StringFlag{
			Name:    "sidecar",
			Value:   "tcp:127.0.0.1:2312",
			Usage:   "address of varlink service",
			Sources: cli.EnvVars("SIDECAR_ADDRESS"),
		},
	},
}

func serverMain(ctx context.Context, cmd *cli.Command) error {
	e := echo.New()

	// Get config
	config, err := conf.GetConfig()
	if err != nil {
		return err
	}

	// Prepare server
	s, err := server.New(config, e.Logger)
	if err != nil {
		return err
	}
	if err = s.Register(e); err != nil {
		return err
	}

	// Run server
	ctxRun, cancelRun := signal.NotifyContext(
		ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT,
	)
	go func() {
		if err = s.Run(e); err != nil {
			e.Logger.Error(err)
		}
		cancelRun()
	}()
	<-ctxRun.Done()
	cancelRun()

	// Shut down server
	shutdownTimeout := cmd.Duration("shutdown-timeout")
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()
	e.Logger.Infof("attempting to shut down gracefully within %.1f sec", shutdownTimeout.Seconds())
	if err := s.Shutdown(ctxShutdown, e); err != nil {
		e.Logger.Warn("forcibly closing http server due to failure of graceful shutdown")
		if closeErr := s.Close(e); closeErr != nil {
			return closeErr
		}
	}
	e.Logger.Info("finished shutdown")
	return nil
}
