package networkmanager

import (
	"fmt"
	"net/netip"

	"github.com/godbus/dbus/v5"
	"github.com/pkg/errors"
)

type IPAddress struct {
	Prefix     netip.Prefix
	Attributes map[string]string
}

type IPRoute struct {
	Destination netip.Prefix
	NextHop     netip.Addr
	Metric      uint32
	Attributes  map[string]string
}

type DNSConfig struct {
	Nameservers []netip.Addr
	Domains     []string
	Searches    []string
	Options     []string
	Priority    int32
}

func (c DNSConfig) HasData() bool {
	return len(c.Nameservers)+len(c.Domains)+len(c.Searches)+len(c.Options) > 0
}

type IPConfig struct {
	Addresses []IPAddress
	Gateway   string
	Routes    []IPRoute
	DNS       DNSConfig
}

func (c IPConfig) HasData() bool {
	return len(c.Addresses)+len(c.Routes) > 0 || c.Gateway != "" || c.DNS.HasData()
}

const (
	ipv4Version = 4
	ipv6Version = 6
)

func dumpIPConfig(confo dbus.BusObject, ipVersion uint8) (conf IPConfig, err error) {
	ipConfigName := fmt.Sprintf(".IP%dConfig", ipVersion)

	var rawObjects []map[string]dbus.Variant
	if err = confo.StoreProperty(nmName+ipConfigName+".AddressData", &rawObjects); err != nil {
		return IPConfig{}, errors.Wrap(err, "couldn't query for IP addresses")
	}
	for _, obj := range rawObjects {
		address, err := parseIPAddress(obj)
		if err != nil {
			return IPConfig{}, errors.Wrapf(err, "couldn't parse IP address %+v", address)
		}
		conf.Addresses = append(conf.Addresses, address)
	}

	if err = confo.StoreProperty(nmName+ipConfigName+".Gateway", &conf.Gateway); err != nil {
		return IPConfig{}, errors.Wrap(err, "couldn't query flag for gateway")
	}

	if err = confo.StoreProperty(nmName+ipConfigName+".RouteData", &rawObjects); err != nil {
		return IPConfig{}, errors.Wrap(err, "couldn't query for IP routes")
	}
	for _, obj := range rawObjects {
		route, err := parseIPRoute(obj)
		if err != nil {
			return IPConfig{}, errors.Wrapf(err, "couldn't parse IP route %+v", route)
		}
		conf.Routes = append(conf.Routes, route)
	}

	if conf.DNS.Nameservers, err = parseNameservers(confo, ipVersion); err != nil {
		return IPConfig{}, errors.New("couldn't parse nameservers")
	}

	if err = confo.StoreProperty(
		nmName+ipConfigName+".Domains", &conf.DNS.Domains,
	); err != nil {
		return IPConfig{}, errors.Wrap(err, "couldn't query for DNS domains")
	}
	if err = confo.StoreProperty(
		nmName+ipConfigName+".Searches", &conf.DNS.Searches,
	); err != nil {
		return IPConfig{}, errors.Wrap(err, "couldn't query for DNS searches")
	}
	if err = confo.StoreProperty(
		nmName+ipConfigName+".DnsOptions", &conf.DNS.Options,
	); err != nil {
		return IPConfig{}, errors.Wrap(err, "couldn't query for DNS options")
	}
	if err = confo.StoreProperty(
		nmName+ipConfigName+".DnsPriority", &conf.DNS.Priority,
	); err != nil {
		return IPConfig{}, errors.Wrap(err, "couldn't query for DNS priority")
	}

	// Note: we don't dump the DHCP config here, because that's a separate D-Bus object.
	return conf, nil
}

func parseIPAddress(rawAddress map[string]dbus.Variant) (address IPAddress, err error) {
	raw, ok := rawAddress["address"]
	if !ok {
		return IPAddress{}, errors.New("no IP address string")
	}
	parsedAddr, err := parseIPAddressString(raw)
	if err != nil {
		return IPAddress{}, errors.Wrap(err, "couldn't parse IP address string")
	}

	raw, ok = rawAddress["prefix"]
	if !ok {
		return IPAddress{}, errors.New("no IP address prefix")
	}
	prefix, ok := raw.Value().(uint32)
	if !ok {
		return IPAddress{}, errors.Errorf("IP address prefix has unexpected type %T", raw.Value())
	}

	address.Prefix = netip.PrefixFrom(parsedAddr, int(prefix))

	address.Attributes = make(map[string]string)
	for key, value := range rawAddress {
		if key == "address" || key == "prefix" {
			continue
		}
		address.Attributes[key] = fmt.Sprint(value.Value())
	}
	return address, nil
}

func parseIPAddressString(raw dbus.Variant) (parsed netip.Addr, err error) {
	addr, ok := raw.Value().(string)
	if !ok {
		return netip.Addr{}, errors.Errorf("IP address string has unexpected type %T", raw.Value())
	}
	if parsed, err = netip.ParseAddr(addr); err != nil {
		return netip.Addr{}, errors.Errorf("couldn't parse IP address string %s", addr)
	}
	return parsed, nil
}

func parseIPRoute(rawRoute map[string]dbus.Variant) (route IPRoute, err error) {
	raw, ok := rawRoute["dest"]
	if !ok {
		return IPRoute{}, errors.New("no destination IP address string")
	}
	parsedAddr, err := parseIPAddressString(raw)
	if err != nil {
		return IPRoute{}, errors.Wrap(err, "couldn't parse IP address string")
	}

	raw, ok = rawRoute["prefix"]
	if !ok {
		return IPRoute{}, errors.New("no destination IP address prefix")
	}
	prefix, ok := raw.Value().(uint32)
	if !ok {
		return IPRoute{}, errors.Errorf(
			"destination IP address prefix has unexpected type %T", raw.Value(),
		)
	}

	route.Destination = netip.PrefixFrom(parsedAddr, int(prefix))

	if raw, ok = rawRoute["next-hop"]; ok {
		if route.NextHop, err = parseIPAddressString(raw); err != nil {
			return IPRoute{}, errors.Wrap(err, "couldn't parse next-hop IP address string")
		}
	}

	if raw, ok = rawRoute["metric"]; ok {
		if route.Metric, ok = raw.Value().(uint32); !ok {
			return IPRoute{}, errors.Errorf("metric has unexpected type %T", raw.Value())
		}
	}

	route.Attributes = make(map[string]string)
	for key, value := range rawRoute {
		if key == "dest" || key == "prefix" || key == "next-hop" || key == "metric" {
			continue
		}
		route.Attributes[key] = fmt.Sprint(value.Value())
	}
	return route, nil
}

func parseNameservers(confo dbus.BusObject, ipVersion uint8) (nameservers []netip.Addr, err error) {
	ipConfigName := fmt.Sprintf(".IP%dConfig", ipVersion)

	switch ipVersion {
	case ipv4Version:
		var rawObjects []map[string]dbus.Variant
		if err = confo.StoreProperty(
			nmName+ipConfigName+".NameserverData", &rawObjects,
		); err != nil {
			return nil, errors.Wrap(err, "couldn't query for nameservers")
		}
		for _, obj := range rawObjects {
			parsed, err := parseIPAddressString(obj["address"])
			if err != nil {
				return nil, errors.Wrapf(err, "couldn't parse nameserver %+v", obj["address"])
			}
			nameservers = append(nameservers, parsed)
		}
	case ipv6Version:
		var rawAddrs [][]byte
		if err = confo.StoreProperty(
			nmName+ipConfigName+".Nameservers", &rawAddrs,
		); err != nil {
			return nil, errors.Wrap(err, "couldn't query for nameservers")
		}
		for _, addr := range rawAddrs {
			parsed, ok := netip.AddrFromSlice(addr)
			if !ok {
				return nil, errors.Wrapf(err, "couldn't parse nameserver %+v", addr)
			}
			nameservers = append(nameservers, parsed)
		}
	}

	return nameservers, nil
}
