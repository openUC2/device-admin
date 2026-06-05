// Package conf supports environment variable-based application configuration
package conf

import (
	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"

	"github.com/openUC2/machine-admin/internal/clients/sidecar"
)

type Config struct {
	Cache   ristretto.Config
	HTTP    HTTPConfig
	Sidecar sidecar.Config
}

type HTTPConfig struct {
	Port      int
	BasePath  string
	GzipLevel int
}

func GetConfig() (c Config, err error) {
	c.Cache, err = getCacheConfig()
	if err != nil {
		return Config{}, errors.Wrap(err, "couldn't make cache config")
	}

	return c, nil
}
