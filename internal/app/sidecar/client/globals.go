// Package client contains client code for external APIs
package client

import (
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"

	"github.com/openUC2/device-admin/internal/clients/networkmanager"
	"github.com/openUC2/device-admin/internal/clients/systemd"
)

// Server

type BaseGlobals struct {
	Logger godest.Logger
}

type Globals struct {
	Base *BaseGlobals

	Systemd        *systemd.Client
	NetworkManager *networkmanager.Client
}

func NewBaseGlobals(l godest.Logger) (g *BaseGlobals, err error) {
	g = &BaseGlobals{}

	g.Logger = l
	return g, nil
}

func NewGlobals(l godest.Logger) (g *Globals, err error) {
	g = &Globals{}
	if g.Base, err = NewBaseGlobals(l); err != nil {
		return nil, errors.Wrap(err, "couldn't set up base globals")
	}

	g.Systemd = systemd.NewClient(systemd.Config{}, g.Base.Logger)
	g.NetworkManager = networkmanager.NewClient(networkmanager.Config{}, g.Base.Logger)

	return g, nil
}
