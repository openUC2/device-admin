package routes

import (
	"github.com/pkg/errors"
	"github.com/varlink/go/varlink"

	"github.com/openUC2/device-admin/internal/app/sidecar/client"
	"github.com/openUC2/device-admin/internal/app/sidecar/routes/boot"
	"github.com/openUC2/device-admin/internal/app/sidecar/routes/networkmanager"
)

type Handlers struct {
	globals *client.Globals
}

func New(globals *client.Globals) *Handlers {
	return &Handlers{
		globals: globals,
	}
}

func (s *Handlers) Register(service *varlink.Service) error {
	l := s.globals.Base.Logger
	if err := boot.New(l).Register(service); err != nil {
		return errors.Wrap(err, "couldn't register boot handlers")
	}
	if err := networkmanager.New(s.globals.NetworkManager, l).Register(service); err != nil {
		return errors.Wrap(err, "couldn't register networkmanager handlers")
	}
	return nil
}
