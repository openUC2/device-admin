package networkmanager

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

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

func NewConnProfileSettingsKey(section, key string) ConnProfileSettingsKey {
	return ConnProfileSettingsKey{
		Section: section,
		Key:     key,
	}
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
		if err := handleField(fullKey, value, rawSettings); err != nil {
			return errors.Errorf("couldn't handle (key, value) pair: (%s, %+v)", fullKey, value)
		}
	}
	if keyMgmt, ok := newSettings[ConnProfileSettingsKey{
		Section: "802-11-wireless-security", Key: "key-mgmt",
	}].(ConnProfileSettingsWifiSecKeyMgmt); ok && keyMgmt == "" {
		// Note(ethanjli): if the caller wants an unsecured network without any password, the caller
		// should set key-mgmt to ""; then NetworkManager should have an empty 802-11-wireless-security
		// section (because it rejects an empty string for psk):
		delete(rawSettings, "802-11-wireless-security")
	}

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
	// TODO: set plugin in args to store the password somehow?
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

func handleField(
	key ConnProfileSettingsKey, value any, settings map[string]map[string]dbus.Variant,
) (err error) {
	if key.Section == "802-11-wireless" && key.Key == "band" {
		if band, ok := value.(ConnProfileSettingsWifiBand); ok && band == "" {
			// NetworkManager rejects band=""; instead, to set an empty band, we must omit it from the
			// settings:
			delete(settings[key.Section], key.Key)
			return nil
		}
	}

	result, err := makeVariant(value)
	if err != nil {
		return err
	}
	if settings[key.Section] == nil {
		settings[key.Section] = make(map[string]dbus.Variant)
	}
	settings[key.Section][key.Key] = result
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
