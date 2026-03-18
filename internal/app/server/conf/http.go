package conf

import (
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest/env"
)

const httpEnvPrefix = "HTTP_"

type HTTPConfig struct {
	Port      int
	BasePath  string
	GzipLevel int
}

func getHTTPConfig() (c HTTPConfig, err error) {
	const defaultPort = 3001
	rawPort, err := env.GetInt64(httpEnvPrefix+"PORT", defaultPort)
	if err != nil {
		return HTTPConfig{}, errors.Wrap(err, "couldn't make port config")
	}
	c.Port = int(rawPort)

	const defaultBasePath = "/"
	c.BasePath = env.GetString(httpEnvPrefix+"BASEPATH", defaultBasePath)

	const defaultGzipLevel = 1
	rawGzipLevel, err := env.GetInt64(httpEnvPrefix+"GZIPLEVEL", defaultGzipLevel)
	if err != nil {
		return HTTPConfig{}, errors.Wrap(err, "couldn't make gzip level config")
	}
	c.GzipLevel = int(rawGzipLevel)
	return c, nil
}
