package templates

import (
	"github.com/sargassum-world/godest/env"
)

const envPrefix = "TEMPLATES_"

type Config struct {
	Path string
}

func GetConfig() (c Config, err error) {
	// This is a file path specific to ImSwitch OS
	const defaultPath = ""
	c.Path = env.GetString(envPrefix+"PATH", defaultPath)

	return c, nil
}
