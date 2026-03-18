// Package client contains client code for external APIs
package client

import (
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"

	"github.com/openUC2/device-admin/internal/clients/networkmanager"
)

// Server

type BaseGlobals struct {
	Logger godest.Logger
}

type Globals struct {
	Base *BaseGlobals

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

	networkManagerConfig, err := networkmanager.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't set up networkmanager config")
	}
	g.NetworkManager = networkmanager.NewClient(networkManagerConfig, g.Base.Logger)

	return g, nil
}
