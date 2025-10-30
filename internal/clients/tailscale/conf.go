package tailscale

import (
	"github.com/sargassum-world/godest/env"
)

const envPrefix = "TAILSCALE_"

type Config struct {
	KnownTailnet string
}

func GetConfig() (c Config, err error) {
	const defaultKnownTailnet = ""
	c.KnownTailnet = env.GetString(envPrefix+"KNOWN_TAILNET", defaultKnownTailnet)

	return c, nil
}
