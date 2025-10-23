package networkmanager

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type ConnProfileSettings struct {
	Conn    ConnProfileSettingsConn    // connection
	Wifi    ConnProfileSettingsWifi    // 802-11-wireless
	WifiSec ConnProfileSettingsWifiSec // 802-11-wireless-security
	// WifiAuthn  ConnProfileSettings8021x // 802-1x
	// Ethernet ConnProfileSettings8023Ethernet // 802-3-ethernet
	IPv4 ConnProfileSettingsIPv4 // ipv4
	IPv6 ConnProfileSettingsIPv6 // ipv6
}

func (s ConnProfileSettings) HasData() bool {
	return s.Conn != ConnProfileSettingsConn{}
}

type ConnProfileSettingsConn struct {
	AuthRetries         int32
	Autoconnect         bool
	AutoconnectPriority int32
	AutoconnectRetries  int32
	ID                  string
	InterfaceName       string
	// IPPingAddresses     []netip.Addr
	// IPPingTimeout       time.Duration
	StableID  string
	Timestamp time.Time
	Type      ConnProfileSettingsConnType
	UUID      uuid.UUID
	// WaitActivationDelay time.Duration
	// WaitDeviceTimeout   time.Duration
	Zone string
}

type ConnProfileSettingsConnType string

var connProfileSettingsConnTypeInfo = map[ConnProfileSettingsConnType]EnumInfo{
	"802-11-wireless": {
		Short: "wifi",
	},
	"802-3-ethernet": {
		Short: "ethernet",
	},
}

func (t ConnProfileSettingsConnType) Info() EnumInfo {
	info, ok := connProfileSettingsConnTypeInfo[t]
	if !ok {
		return EnumInfo{Short: string(t)}
	}
	return info
}

type ConnProfileSettingsIPv4 struct {
	Addresses    []IPAddress
	Method       string // TODO: change this to a string enum // TODO
	NeverDefault bool   // TODO
}

type ConnProfileSettingsIPv6 struct {
	Addresses   []IPAddress
	AddrGenMode int32  // TODO: change this to an int32 enum // TODO
	Method      string // TODO: change this to a string enum // TODO
}

func dumpConnProfileSettings(
	ctx context.Context, conno dbus.BusObject,
) (s ConnProfileSettings, err error) {
	var rawSettings map[string]map[string]dbus.Variant
	if err = conno.CallWithContext(
		ctx, nmName+".Settings.Connection.GetSettings", 0,
	).Store(&rawSettings); err != nil {
		return s, errors.Wrap(err, "couldn't get settings")
	}

	if s.Conn, err = dumpConnProfileSettingsConn(
		rawSettings["connection"],
	); err != nil {
		return s, errors.Wrap(err, "couldn't parse 'connection' section")
	}

	if s.Conn.Type == "802-11-wireless" {
		rawSecrets := make(map[string]map[string]dbus.Variant)
		rawSecrets["802-11-wireless"] = make(map[string]dbus.Variant)
		if err = conno.CallWithContext(
			ctx, nmName+".Settings.Connection.GetSecrets", 0, "802-11-wireless-security",
		).Store(&rawSecrets); err != nil {
			// Note(ethanjli): this will fail if there are no secrets; for now, it's safe to assume that
			// this will only fail if there is no PSK, so we can interpret that accordingly.
			rawSecrets["802-11-wireless"]["psk"] = dbus.MakeVariant("")
		}
		if s.Wifi, err = dumpConnProfileSettingsWifi(
			rawSettings["802-11-wireless"],
		); err != nil {
			return s, errors.Wrap(err, "couldn't parse '802-11-wireless' section")
		}

		if s.WifiSec, err = dumpConnProfileSettingsWifiSec(
			rawSettings["802-11-wireless-security"], rawSecrets["802-11-wireless-security"],
		); err != nil {
			return s, errors.Wrap(err, "couldn't parse '802-11-wireless-security' section")
		}
	}

	if s.IPv4, err = dumpConnProfileSettingsIPv4(
		rawSettings["ipv4"],
	); err != nil {
		return s, errors.Wrap(err, "couldn't parse 'ipv4' section")
	}
	if s.IPv6, err = dumpConnProfileSettingsIPv6(
		rawSettings["ipv6"],
	); err != nil {
		return s, errors.Wrap(err, "couldn't parse 'ipv6' section")
	}

	return s, nil
}

func dumpConnProfileSettingsConn(
	rawSettings map[string]dbus.Variant,
) (s ConnProfileSettingsConn, err error) {
	if s.AuthRetries, err = ensureVar[int32](
		rawSettings, "auth-retries", "auth retries", false, -1,
	); err != nil {
		return s, err
	}
	if s.ID, err = ensureVar(rawSettings, "id", "ID", true, ""); err != nil {
		return s, err
	}

	if s.InterfaceName, err = ensureVar(rawSettings, "interface-name", "", true, ""); err != nil {
		return s, err
	}

	rawType, err := ensureVar(rawSettings, "type", "", true, "")
	if err != nil {
		return s, err
	}
	s.Type = ConnProfileSettingsConnType(rawType)

	if s.StableID, err = ensureVar(rawSettings, "stable-id", "stable ID", false, ""); err != nil {
		return s, err
	}

	rawUint, err := ensureVar[uint64](rawSettings, "timestamp", "", false, 0)
	if err != nil {
		return s, err
	}
	switch {
	case rawUint > math.MaxInt64:
		// TODO: log a warning!
	case rawUint == 0:
		s.Timestamp = time.Time{}
	default:
		s.Timestamp = time.Unix(int64(rawUint), 0)
	}

	rawUUID, err := ensureVar(rawSettings, "uuid", "UUID", true, "")
	if err != nil {
		return s, err
	}
	if s.UUID, err = uuid.Parse(rawUUID); err != nil {
		return s, errors.Wrapf(err, "couldn't parse UUID %s", rawUUID)
	}

	if s.Zone, err = ensureVar(rawSettings, "zone", "", false, ""); err != nil {
		return s, err
	}

	if s, err = dumpConnProfileSettingsConnAutoconnect(rawSettings, s); err != nil {
		return s, err
	}

	return s, nil
}

func ensureVar[T any](
	rawSettings map[string]dbus.Variant, key, errorName string, required bool, defaultResult T,
) (result T, err error) {
	if errorName == "" {
		errorName = strings.ReplaceAll(key, "-", " ")
	}

	variant, ok := rawSettings[key]
	if !ok {
		if !required {
			return defaultResult, nil
		}
		return result, errors.Errorf("no %s", errorName)
	}
	if err = variant.Store(&result); err != nil {
		return result, errors.Wrapf(err, "%s has unexpected type %T", errorName, variant.Value())
	}
	return result, nil
}

func dumpConnProfileSettingsConnAutoconnect(
	rawSettings map[string]dbus.Variant, s ConnProfileSettingsConn,
) (ConnProfileSettingsConn, error) {
	var err error

	if s.Autoconnect, err = ensureVar(rawSettings, "autoconnect", "", false, true); err != nil {
		return s, err
	}
	if s.AutoconnectPriority, err = ensureVar[int32](
		rawSettings, "autoconnect-priority", "", false, 0,
	); err != nil {
		return s, err
	}
	if s.AutoconnectRetries, err = ensureVar[int32](
		rawSettings, "autoconnect-retries", "", false, -1,
	); err != nil {
		return s, err
	}

	return s, nil
}

func dumpConnProfileSettingsIPv4(
	rawSettings map[string]dbus.Variant,
) (s ConnProfileSettingsIPv4, err error) {
	rawObjs, err := ensureVar[[]map[string]dbus.Variant](rawSettings, "address-data", "", false, nil)
	if err != nil {
		return s, nil
	}
	for _, obj := range rawObjs {
		address, err := parseIPAddress(obj)
		if err != nil {
			return s, errors.Wrapf(err, "couldn't parse IP address %+v", address)
		}
		s.Addresses = append(s.Addresses, address)
	}

	return s, nil
}

func dumpConnProfileSettingsIPv6(
	rawSettings map[string]dbus.Variant,
) (s ConnProfileSettingsIPv6, err error) {
	rawObjs, err := ensureVar[[]map[string]dbus.Variant](rawSettings, "address-data", "", false, nil)
	if err != nil {
		return s, nil
	}
	for _, obj := range rawObjs {
		address, err := parseIPAddress(obj)
		if err != nil {
			return s, errors.Wrapf(err, "couldn't parse IP address %+v", address)
		}
		s.Addresses = append(s.Addresses, address)
	}

	return s, nil
}
