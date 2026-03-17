package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/urfave/cli/v3"

	"github.com/openUC2/device-admin/internal/app/deviceadmin"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/conf"
)

var serverCmd = &cli.Command{
	Name:   "server",
	Action: serverMain,
	Flags: []cli.Flag{
		&cli.DurationFlag{
			Name:    "shutdown-timeout",
			Value:   5 * time.Second,
			Usage:   "timeout for graceful shutdown before hard shutdown",
			Sources: cli.EnvVars("SERVER_SHUTDOWNTIMEOUT"),
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
	server, err := deviceadmin.NewServer(config, e.Logger)
	if err != nil {
		return err
	}
	if err = server.Register(e); err != nil {
		return err
	}

	// Run server
	ctxRun, cancelRun := signal.NotifyContext(
		ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT,
	)
	go func() {
		if err = server.Run(e); err != nil {
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
	if err := server.Shutdown(ctxShutdown, e); err != nil {
		e.Logger.Warn("forcibly closing http server due to failure of graceful shutdown")
		if closeErr := server.Close(e); closeErr != nil {
			return closeErr
		}
	}
	e.Logger.Info("finished shutdown")
	return nil
}
