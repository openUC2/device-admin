package networkmanager

const envPrefix = "NETWORKMANAGER_"

type Config struct{}

func GetConfig() (c Config, err error) {
	return c, nil
}
