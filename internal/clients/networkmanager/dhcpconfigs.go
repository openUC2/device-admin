package networkmanager

import (
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/pkg/errors"
)

type DHCPConfig struct {
	Options map[string]string
}

func dumpDHCPConfig(confo dbus.BusObject, ipVersion uint8) (conf DHCPConfig, err error) {
	dhcpConfigName := fmt.Sprintf(".DHCP%dConfig", ipVersion)

	var rawMap map[string]dbus.Variant
	if err = confo.StoreProperty(nmName+dhcpConfigName+".Options", &rawMap); err != nil {
		return DHCPConfig{}, errors.Wrap(err, "couldn't query for DHCP options")
	}
	conf.Options = make(map[string]string)
	for key, value := range rawMap {
		conf.Options[key] = fmt.Sprint(value.Value())
	}

	return conf, nil
}

func (c DHCPConfig) HasData() bool {
	return len(c.Options) > 0
}
