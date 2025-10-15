package tmplfunc

import (
	"net/netip"
)

func IsIPAddr(value string) bool {
	_, err := netip.ParseAddr(value)
	return err == nil
}
