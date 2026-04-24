// Package versioning loads and exposes versioning information about the machine
package versioning

import (
	"cmp"
	"os"

	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ForkliftPath string
}

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
	Forklift      string `yaml:"forklift,omitempty"`
	Factory       string `yaml:"factory,omitempty"`
	Current       string `yaml:"current,omitempty"`
	Previous      string `yaml:"previous,omitempty"`
	Pending       string `yaml:"pending,omitempty"`
	Pallet        string `yaml:"pallet,omitempty"`
	Changes       string `yaml:"changes,omitempty"`
	UpgradeSource string `yaml:"upgrade-source,omitempty"`
	Upgrade       string `yaml:"upgrade,omitempty"`
}

func (c *Client) GetForklift() (f Forklift, err error) {
	p := cmp.Or(c.Config.ForkliftPath, "/run/versioning/forklift.yml")
	if f, err = readForklift(p); err != nil {
		return f, errors.Wrapf(err, "couldn't read Forklift versioning file %s", p)
	}
	return f, nil
}

func readForklift(filePath string) (f Forklift, err error) {
	bytes, err := os.ReadFile(filePath) //nolint:gosec // We trust this file
	if err != nil {
		return f, errors.Wrapf(err, "couldn't read Forklift versioning report %s", filePath)
	}
	if err = yaml.Unmarshal(bytes, &f); err != nil {
		return f, errors.Wrap(err, "couldn't parse Forklift versioning report")
	}
	return f, nil
}
