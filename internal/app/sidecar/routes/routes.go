package routes

import (
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/varlink/go/varlink"

	"github.com/openUC2/device-admin/internal/app/sidecar/client"
	"github.com/openUC2/device-admin/internal/app/sidecar/routes/boot"
)

type Handlers struct {
	globals *client.Globals
}

func New(globals *client.Globals) *Handlers {
	return &Handlers{
		globals: globals,
	}
}

func (s *Handlers) Register(service *varlink.Service, l godest.Logger) error {
	if err := boot.New(l).Register(service); err != nil {
		return errors.Wrap(err, "couldn't register boot handlers")
	}
	return nil
}
