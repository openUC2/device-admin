// Package sidecar provides the privileged sidecar for the device-admin server
package sidecar

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/varlink/go/varlink"
	"golang.org/x/sync/errgroup"

	"github.com/openUC2/device-admin/internal/app/sidecar/client"
	"github.com/openUC2/device-admin/internal/app/sidecar/routes"
)

type Config struct {
	Vendor  string
	Product string
	Version string
	URL     string
	Address string
}

type Sidecar struct {
	Config  Config
	Globals *client.Globals
	service *varlink.Service

	Handlers *routes.Handlers
}

func New(config Config, logger godest.Logger) (s *Sidecar, err error) {
	s = &Sidecar{Config: config}
	if s.Globals, err = client.NewGlobals(logger); err != nil {
		return nil, errors.Wrap(err, "couldn't make app globals")
	}
	if s.service, err = varlink.NewService(
		config.Vendor, config.Product, config.Version, config.URL,
	); err != nil {
		return s, errors.Wrap(err, "couldn't create new varlink service")
	}

	s.Handlers = routes.New(s.Globals)
	if err := s.Handlers.Register(s.service, s.Globals.Base.Logger); err != nil {
		return s, errors.Wrap(err, "couldn't register varlink interfaces with service")
	}
	return s, nil
}

// Running

func (s *Sidecar) Run(ctx context.Context) error {
	s.Globals.Base.Logger.Info("starting device-admin sidecar")

	// The varlink listener can't be canceled by context cancelation, so the API shouldn't promise to
	// stop blocking execution on context cancelation - so we use the background context here. The
	// varlink listener should instead be stopped gracefully by calling the Shutdown method.
	eg, egctx := errgroup.WithContext(context.Background())
	eg.Go(func() error {
		s.Globals.Base.Logger.Info("starting background workers")
		if err := s.runWorkersInContext(egctx); err != nil {
			s.Globals.Base.Logger.Error(errors.Wrap(
				err, "background worker encountered error",
			))
		}
		return nil
	})
	eg.Go(func() error {
		s.Globals.Base.Logger.Infof("starting varlink listener on %s", s.Config.Address)
		return s.service.Listen(ctx, s.Config.Address, 0)
	})
	if err := eg.Wait(); err != nil {
		return errors.Wrap(err, "sidecar encountered error")
	}
	return nil
}

func (s *Sidecar) runWorkersInContext(ctx context.Context) error {
	eg, _ := errgroup.WithContext(ctx) // Workers run independently, so we don't need egctx
	eg.Go(func() error {
		if err := s.Globals.NetworkManager.Open(ctx); err != nil {
			s.Globals.Base.Logger.Error("couldn't open NetworkManager client")
			// Even if NetworkManager is unavailable, other parts of device-admin are still useful, so we
			// don't propagate the error from here
		}
		return nil
	})
	return eg.Wait()
}

func (s *Sidecar) Shutdown() (err error) {
	return s.service.Shutdown()
}
