// Package versioning loads and exposes versioning information about the machine
package versioning

import (
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
)

type Config struct{}

type Client struct {
	Config Config

	l godest.Logger
}

func NewClient(c Config, l godest.Logger) *Client {
	return &Client{
		Config: c,
		l:      l,
	}
}

type Forklift struct {
	Factory       string
	Current       string
	Pallet        string
	UpgradeSource string
	Upgrade       string
}

func (c *Client) GetForklift() (f Forklift, err error) {
	return Forklift{}, errors.New("unimplemented")
}
