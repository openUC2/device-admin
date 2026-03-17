// Package deviceadmin provides the ImSwitch OS device-admin server.
package deviceadmin

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"golang.org/x/sync/errgroup"

	"github.com/openUC2/device-admin/internal/app/deviceadmin/client"
)

type Sidecar struct {
	Globals *client.SidecarGlobals
}

func NewSidecar(logger godest.Logger) (s *Sidecar, err error) {
	s = &Sidecar{}
	if s.Globals, err = client.NewSidecarGlobals(logger); err != nil {
		return nil, errors.Wrap(err, "couldn't make app globals")
	}

	return s, err
}

// Running

func (s *Sidecar) Run(ctx context.Context) error {
	s.Globals.Logger.Info("starting device-admin sidecar")

	eg, egctx := errgroup.WithContext(context.Background())
	eg.Go(func() error {
		s.Globals.Logger.Info("starting background workers")
		if err := s.runWorkersInContext(egctx); err != nil {
			s.Globals.Logger.Error(errors.Wrap(
				err, "background worker encountered error",
			))
		}
		return nil
	})
	eg.Go(func() error {
		s.Globals.Logger.Infof("starting server")
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				time.Sleep(1)
			}
		}
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
			s.Globals.Logger.Error("couldn't open NetworkManager client")
			// Even if NetworkManager is unavailable, other parts of device-admin are still useful, so we
			// don't propagate the error from here
		}
		return nil
	})
	return eg.Wait()
}

func (s *Sidecar) Shutdown(ctx context.Context) (err error) {
	return nil
}
