package networkmanager

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type ConnProfile struct {
	Unsaved  bool
	Flags    ConnProfileFlags
	Filename string
	Settings ConnProfileSettings
}

type ConnProfileFlags uint32

func (f ConnProfileFlags) HasNone() bool {
	return f == 0
}

func (f ConnProfileFlags) Unsaved() bool {
	return f&0x1 > 0
}

func (f ConnProfileFlags) GeneratedByNM() bool {
	return f&0x2 > 0
}

func (f ConnProfileFlags) Volatile() bool {
	return f&0x4 > 0
}

func (f ConnProfileFlags) External() bool {
	return f&0x8 > 0
}

type ConnProfileSettings struct {
	Conn    ConnProfileSettingsConn    // connection
	Wifi    ConnProfileSettingsWifi    // 802-11-wireless
	WifiSec ConnProfileSettingsWifiSec // 802-11-wireless-security
	// WifiAuthn  ConnProfileSettings8021x // 802-1x
	// Ethernet ConnProfileSettings8023Ethernet // 802-3-ethernet
	IPv4 ConnProfileSettingsIPv4 // ipv4
	IPv6 ConnProfileSettingsIPv6 // ipv6
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

type ConnProfileSettingsWifi struct {
	// APIsolation int32 // TODO: change this to an int32 enum
	// AssignedMACAddress  string
	Band ConnProfileSettingsWifiBand
	// BSSID               []string
	Channel uint32 // TODO
	// ChannelWidth int32
	Hidden bool // TODO
	// MACAddress          []byte
	// MACAddressBlacklist []string
	// MACAddressDenylist  []string
	Mode ConnProfileSettingsWifiMode // TODO
	// MTU                 uint32
	// Powersave uint32 // TODO: change this to a uint32 enum
	// SeenBSSIDs          []string
	SSID []byte // TODO
}

type ConnProfileSettingsWifiBand string

var connProfileSettingsWifiBand = map[ConnProfileSettingsWifiBand]EnumInfo{
	"a": {
		Short:   "a",
		Details: "802.11a (5 GHz)",
	},
	"bg": {
		Short:   "bg",
		Details: "802.11b/g (2.4 GHz)",
	},
	"": {
		Short:   "any",
		Details: "any available band",
	},
}

func (b ConnProfileSettingsWifiBand) Info() EnumInfo {
	info, ok := connProfileSettingsWifiBand[b]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("unknown band (%s)", b),
			Level:   "error",
		}
	}
	return info
}

type ConnProfileSettingsWifiMode string

var connProfileSettingsWifiMode = map[ConnProfileSettingsWifiMode]EnumInfo{
	"": {
		Short:   "infrastructure",
		Details: "connect to an external Wi-Fi network",
	},
	"infrastructure": {
		Short:   "infrastructure",
		Details: "connect to an external Wi-Fi network",
	},
	"mesh": {
		Short:   "mesh",
		Details: "connect to a mesh Wi-Fi network",
	},
	"adhoc": {
		Short:   "ad-hoc",
		Details: "connect to an ad-hoc Wi-Fi network",
	},
	"ap": {
		Short:   "ap",
		Details: "create a Wi-Fi hotspot",
	},
}

func (m ConnProfileSettingsWifiMode) Info() EnumInfo {
	info, ok := connProfileSettingsWifiMode[m]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("unknown mode (%s)", m),
			Level:   "error",
		}
	}
	return info
}

type ConnProfileSettingsWifiSec struct {
	AuthAlg  string   // TODO: change this to a string enum
	Group    []string // TODO: change this to an array of string enums
	KeyMgmt  string   // TODO: change this to a string enum
	Pairwise []string // TODO: change this to an array of string enums
	Proto    []string // TODO: change this to an array of string enums
	PSK      string   // TODO
	PSKFlags uint32   // TODO: change this to a uint32 flags type
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

func GetConnProfileByUUID(ctx context.Context, uid uuid.UUID) (conn ConnProfile, err error) {
	conno, err := findConnProfileByUUID(ctx, uid)
	if err != nil {
		return ConnProfile{}, errors.Wrapf(err, "couldn't find connection profile with uuid %s", uid)
	}
	if conn, err = dumpConnProfile(ctx, conno); err != nil {
		return ConnProfile{}, errors.Wrapf(err, "couldn't dump connection profile with uuid %s", uid)
	}
	return conn, nil
}

func findConnProfileByUUID(ctx context.Context, uid uuid.UUID) (conno dbus.BusObject, err error) {
	nm, bus, err := getNetworkManagerSettings(ctx)
	if err != nil {
		return nil, err
	}

	var connPath dbus.ObjectPath
	if err = nm.CallWithContext(
		ctx, nmName+".Settings.GetConnectionByUuid", 0, uid.String(),
	).Store(&connPath); err != nil {
		return nil, errors.Wrapf(err, "couldn't query for connection profile with uuid %s", uid)
	}

	return bus.Object(nmName, connPath), nil
}

func dumpConnProfile(ctx context.Context, conno dbus.BusObject) (conn ConnProfile, err error) {
	const connName = nmName + ".Settings.Connection"

	if err = conno.StoreProperty(connName+".Unsaved", &conn.Unsaved); err != nil {
		return ConnProfile{}, errors.Wrap(err, "couldn't query for unsaved")
	}

	var rawFlags uint32
	if err = conno.StoreProperty(connName+".Flags", &rawFlags); err != nil {
		return ConnProfile{}, errors.Wrap(err, "couldn't query for flags")
	}
	conn.Flags = ConnProfileFlags(rawFlags)

	if err = conno.StoreProperty(connName+".Filename", &conn.Filename); err != nil {
		return ConnProfile{}, errors.Wrap(err, "couldn't query for filename")
	}

	if conn.Settings, err = dumpConnProfileSettings(ctx, conno); err != nil {
		return ConnProfile{}, errors.Wrap(err, "couldn't query for connection settings")
	}

	return conn, nil
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

	if s.Wifi, err = dumpConnProfileSettingsWifi(
		rawSettings["802-11-wireless"],
	); err != nil {
		return s, errors.Wrap(err, "couldn't parse '802-11-wireless' section")
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

func dumpConnProfileSettingsWifi(
	rawSettings map[string]dbus.Variant,
) (s ConnProfileSettingsWifi, err error) {
	rawBand, err := ensureVar(rawSettings, "band", "", false, "")
	if err != nil {
		return s, err
	}
	s.Band = ConnProfileSettingsWifiBand(rawBand)

	if s.Channel, err = ensureVar[uint32](rawSettings, "channel", "", false, 0); err != nil {
		return s, err
	}

	if s.Hidden, err = ensureVar(rawSettings, "hidden", "", false, false); err != nil {
		return s, err
	}

	rawMode, err := ensureVar(rawSettings, "mode", "", false, "")
	if err != nil {
		return s, err
	}
	if rawMode == "" {
		rawMode = "infrastructure"
	}
	s.Mode = ConnProfileSettingsWifiMode(rawMode)

	if s.SSID, err = ensureVar(rawSettings, "ssid", "SSID", false, []byte{}); err != nil {
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

func ListConnProfiles(ctx context.Context) (conns []ConnProfile, err error) {
	nm, bus, err := getNetworkManagerSettings(ctx)
	if err != nil {
		return nil, err
	}

	var connPaths []dbus.ObjectPath
	if err = nm.CallWithContext(
		ctx, nmName+".Settings.ListConnections", 0,
	).Store(&connPaths); err != nil {
		return nil, errors.Wrap(err, "couldn't list connection profiles")
	}

	for _, connPath := range connPaths {
		conn, err := dumpConnProfile(ctx, bus.Object(nmName, connPath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't dump connection profile %s", connPath)
		}
		conns = append(conns, conn)
	}

	slices.SortFunc(conns, func(a, b ConnProfile) int {
		return cmp.Compare(a.Settings.Conn.ID, b.Settings.Conn.ID)
	})

	return conns, nil
}

func ReloadConnProfiles(ctx context.Context) error {
	nm, _, err := getNetworkManagerSettings(ctx)
	if err != nil {
		return err
	}

	var status bool
	if err = nm.CallWithContext(
		ctx, nmName+".Settings.ReloadConnections", 0,
	).Store(&status); err != nil {
		return errors.Wrap(err, "couldn't reload connection profiles")
	}
	if !status {
		return errors.New("reload of connection profiles encountered an unexpected failure")
	}

	return nil
}

func ActivateConnProfile(ctx context.Context, uid uuid.UUID) error {
	nm, _, err := getNetworkManager(ctx)
	if err != nil {
		return err
	}
	conno, err := findConnProfileByUUID(ctx, uid)
	if err != nil {
		return errors.Wrap(err, "couldn't find connection profile to activate")
	}

	var activeConn dbus.ObjectPath
	if err = nm.CallWithContext(
		ctx, nmName+".ActivateConnection", 0, conno.Path(), dbus.ObjectPath("/"), dbus.ObjectPath("/"),
	).Store(&activeConn); err != nil {
		return errors.Wrapf(err, "couldn't activate connection profile with UUID %s", uid)
	}

	return nil
}

type ConnProfileSettingsKey struct {
	Section   string
	Key       string
	Remainder string
}

func ParseConnProfileSettingsKey(rawKey string) (k ConnProfileSettingsKey, err error) {
	var ok bool
	if k.Section, k.Key, ok = strings.Cut(rawKey, "."); !ok {
		return ConnProfileSettingsKey{}, errors.Errorf("key %s doesn't have a ", rawKey)
	}
	k.Key, k.Remainder, _ = strings.Cut(k.Key, ".")
	return k, nil
}

func (k ConnProfileSettingsKey) String() string {
	return fmt.Sprintf("%s.%s", k.Section, k.Key)
}

func UpdateConnProfileByUUID(
	ctx context.Context, uid uuid.UUID, updateType string, newSettings map[ConnProfileSettingsKey]any,
) error {
	conno, err := findConnProfileByUUID(ctx, uid)
	if err != nil {
		return errors.Wrapf(err, "couldn't find connection profile with uuid %s", uid.String())
	}

	var rawSettings map[string]map[string]dbus.Variant
	if err = conno.CallWithContext(
		ctx, nmName+".Settings.Connection.GetSettings", 0,
	).Store(&rawSettings); err != nil {
		return errors.Wrapf(err, "couldn't get settings of connection profile %s", uid.String())
	}
	// Remove deprecated fields which would override non-deprecated fields:
	delete(rawSettings["ipv4"], "addresses")
	delete(rawSettings["ipv4"], "routes")
	delete(rawSettings["ipv6"], "addresses")
	delete(rawSettings["ipv6"], "routes")

	for fullKey, value := range newSettings {
		if rawSettings[fullKey.Section][fullKey.Key], err = makeVariant(value); err != nil {
			return errors.Errorf("couldn't set value %+v for key %s", value, fullKey)
		}
		if fullKey.Section == "802-11-wireless" && fullKey.Key == "band" {
			if band, ok := value.(ConnProfileSettingsWifiBand); ok && band == "" {
				// NetworkManager rejects "" as band; instead, to set an empty band, we must omit it from
				// the settings:
				delete(rawSettings[fullKey.Section], fullKey.Key)
			}
		}
	}
	// TODO: handle secrets properly

	var flags UpdateFlags
	switch updateType {
	default:
		return errors.Errorf("unknown update type %s", updateType)
	case "apply":
		flags |= UpdateFlagInMemory
	case "save":
		flags |= UpdateFlagToDisk
	}

	args := make(map[string]dbus.Variant)
	// TODO: set plugin in args to store the password somehow
	// TODO: set version-id in args to detect data races (requires NetworkManager 1.44, which is too
	// recent)

	var rawResult map[string]dbus.Variant
	if err = conno.CallWithContext(
		ctx, nmName+".Settings.Connection.Update2", 0, rawSettings, uint32(flags), args,
	).Store(&rawResult); err != nil {
		return errors.Wrapf(err, "couldn't apply settings of connection profile %s", uid.String())
	}

	// TODO: add checkpointing behavior for auto-rollback in the absence of manual confirmation

	return nil
}

type UpdateFlags uint32

const (
	UpdateFlagToDisk   = 0x1
	UpdateFlagInMemory = 0x1
)

func makeVariant(value any) (variant dbus.Variant, err error) {
	defer func() {
		if r := recover(); r != nil {
			variant = dbus.Variant{}
			err = errors.Errorf("value %+v not representable in D-Bus", value)
		}
	}()
	variant = dbus.MakeVariant(value)
	return variant, err // note: these returns are modified by the defer when MakeVariant panics
}
