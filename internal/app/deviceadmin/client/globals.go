// Package client contains client code for external APIs
package client

import (
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/sargassum-world/godest/clientcache"

	"github.com/openUC2/device-admin/internal/app/deviceadmin/conf"
	"github.com/openUC2/device-admin/internal/clients/networkmanager"
	"github.com/openUC2/device-admin/internal/clients/tailscale"
	"github.com/openUC2/device-admin/internal/clients/templates"
)

type BaseGlobals struct {
	Cache clientcache.Cache

	Logger godest.Logger
}

type Globals struct {
	Config conf.Config
	Base   *BaseGlobals

	Templates      *templates.Client
	NetworkManager *networkmanager.Client
	Tailscale      *tailscale.Client
}

func NewBaseGlobals(config conf.Config, l godest.Logger) (g *BaseGlobals, err error) {
	g = &BaseGlobals{}
	if g.Cache, err = clientcache.NewRistrettoCache(config.Cache); err != nil {
		return nil, errors.Wrap(err, "couldn't set up client cache")
	}
	g.Logger = l
	return g, nil
}

func NewGlobals(config conf.Config, l godest.Logger) (g *Globals, err error) {
	g = &Globals{
		Config: config,
	}
	g.Base, err = NewBaseGlobals(config, l)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't set up base globals")
	}

	templatesConfig, err := templates.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't set up templates config")
	}
	g.Templates = templates.NewClient(templatesConfig)

	networkManagerConfig, err := networkmanager.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't set up networkmanager config")
	}
	g.NetworkManager = networkmanager.NewClient(networkManagerConfig, g.Base.Logger)

	tailscaleConfig, err := tailscale.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't set up tailscale config")
	}
	g.Tailscale = tailscale.NewClient(tailscaleConfig, g.Base.Logger)

	return g, nil
}
