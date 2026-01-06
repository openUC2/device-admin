package udisks2

import (
	"bytes"
	"cmp"
	"context"
	"slices"

	"github.com/godbus/dbus/v5"
	"github.com/pkg/errors"
)

type BlockDevice struct {
	Device          string
	PreferredDevice string
	ID              string
	Size            uint64
	ReadOnly        bool
	Drive           Drive
	FSVersion       string
	FSLabel         string
	FSUUID          string // Note: format depends on the type of data in the device, so we don't parse it
	HintSystem      bool
	HintIgnore      bool
	HintName        string
	Filesystem      Filesystem
}

type Filesystem struct {
	MountPoints []string
	Size        uint64
}

func (f Filesystem) HasData() bool {
	return f.Size > 0 || len(f.MountPoints) > 0
}

func (c *Client) GetBlockDevices(ctx context.Context) (devs []BlockDevice, err error) {
	udm := c.getUDisks2Manager()
	devPaths := make([]dbus.ObjectPath, 0)
	options := make(map[string]dbus.Variant)
	options["auth.no_user_interaction"] = dbus.MakeVariant(false)
	if err = udm.CallWithContext(
		ctx, udName+".Manager.GetBlockDevices", 0, options,
	).Store(&devPaths); err != nil {
		return nil, errors.Wrap(err, "couldn't query for block devices")
	}
	for _, devPath := range devPaths {
		device, err := dumpBlockDevice(c.bus.Object(udName, devPath), c.bus)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't dump block device %s", devPath)
		}
		devs = append(devs, device)
	}
	slices.SortFunc(devs, func(a, b BlockDevice) int {
		return cmp.Compare(a.Device, b.Device)
	})
	return devs, nil
}

func dumpBlockDevice(devo dbus.BusObject, bus *dbus.Conn) (dev BlockDevice, err error) {
	var rawBytes []byte
	if err = devo.StoreProperty(udName+".Block.Device", &rawBytes); err != nil {
		return BlockDevice{}, errors.Wrap(err, "couldn't query for special device file of device")
	}
	dev.Device = string(bytes.Trim(rawBytes, "\x00"))

	if err = devo.StoreProperty(udName+".Block.PreferredDevice", &rawBytes); err != nil {
		return BlockDevice{}, errors.Wrap(
			err, "couldn't query for presentable special device file of device",
		)
	}
	dev.Device = string(bytes.Trim(rawBytes, "\x00"))

	if err = devo.StoreProperty(udName+".Block.Id", &dev.ID); err != nil {
		return BlockDevice{}, errors.Wrap(err, "couldn't query for persistent identifier of device")
	}
	if err = devo.StoreProperty(udName+".Block.Size", &dev.Size); err != nil {
		return BlockDevice{}, errors.Wrap(err, "couldn't query for size of device")
	}
	if err = devo.StoreProperty(udName+".Block.ReadOnly", &dev.ReadOnly); err != nil {
		return BlockDevice{}, errors.Wrap(err, "couldn't query for writeability of device")
	}

	if dev, err = dumpDataStructureIdentifiers(devo, dev); err != nil {
		return BlockDevice{}, err
	}

	if dev, err = dumpHints(devo, dev); err != nil {
		return BlockDevice{}, err
	}

	var drivePath dbus.ObjectPath
	if err = devo.StoreProperty(udName+".Block.Drive", &drivePath); err != nil {
		return BlockDevice{}, errors.Wrap(err, "couldn't query for D-Bus path of drive with device")
	}
	if drivePath != "" && drivePath != "/" {
		dev.Drive, err = dumpDrive(bus.Object(udName, drivePath))
		if err != nil {
			return BlockDevice{}, errors.Wrapf(err, "couldn't dump drive %s", drivePath)
		}
	}

	if dev.Filesystem, err = dumpFilesystem(devo); err != nil {
		return BlockDevice{}, errors.Wrap(err, "couldn't dump device as a filesystem")
	}

	return dev, nil
}

func dumpDataStructureIdentifiers(devo dbus.BusObject, dev BlockDevice) (BlockDevice, error) {
	var err error

	if err = devo.StoreProperty(udName+".Block.IdVersion", &dev.FSVersion); err != nil {
		return BlockDevice{}, errors.Wrap(
			err, "couldn't query for version of filesystem or structured data on device",
		)
	}
	if err = devo.StoreProperty(udName+".Block.IdLabel", &dev.FSLabel); err != nil {
		return BlockDevice{}, errors.Wrap(
			err, "couldn't query for label of filesystem or other structured data on device",
		)
	}
	if err = devo.StoreProperty(udName+".Block.IdUUID", &dev.FSUUID); err != nil {
		return BlockDevice{}, errors.Wrap(
			err, "couldn't query for uuid of filesystem or other structured data on device",
		)
	}

	return dev, nil
}

func dumpHints(devo dbus.BusObject, dev BlockDevice) (BlockDevice, error) {
	var err error

	if err = devo.StoreProperty(udName+".Block.HintSystem", &dev.HintSystem); err != nil {
		return BlockDevice{}, errors.Wrap(
			err, "couldn't query for hinted systemness of device",
		)
	}
	if err = devo.StoreProperty(udName+".Block.HintIgnore", &dev.HintIgnore); err != nil {
		return BlockDevice{}, errors.Wrap(
			err, "couldn't query for hinted hiddenness for presenting device",
		)
	}
	if err = devo.StoreProperty(udName+".Block.HintName", &dev.HintName); err != nil {
		return BlockDevice{}, errors.Wrap(err, "couldn't query for hinted name for presenting device")
	}

	return dev, nil
}

func dumpFilesystem(devo dbus.BusObject) (f Filesystem, err error) {
	var properties map[string]dbus.Variant
	if err = devo.Call(
		"org.freedesktop.DBus.Properties.GetAll", 0, udName+".Filesystem",
	).Store(&properties); err != nil {
		return Filesystem{}, nil
	}

	if err = devo.StoreProperty(udName+".Filesystem.Size", &f.Size); err != nil {
		return Filesystem{}, errors.Wrap(err, "couldn't query for size of filesystem")
	}
	var rawMountPoints [][]byte
	if err = devo.StoreProperty(udName+".Filesystem.MountPoints", &rawMountPoints); err != nil {
		return Filesystem{}, errors.Wrap(err, "couldn't query for mount points of filesystem")
	}
	for _, rawMountPoint := range rawMountPoints {
		if rawMountPoint[len(rawMountPoint)-1] == 0 {
			rawMountPoint = rawMountPoint[:len(rawMountPoint)-1]
		}
		f.MountPoints = append(f.MountPoints, string(rawMountPoint))
	}

	return f, nil
}

func (c *Client) UnmountBlockDevice(ctx context.Context, id string) error {
	devo, err := c.findBlockDeviceByID(ctx, id)
	if err != nil {
		return err
	}

	options := make(map[string]dbus.Variant)
	options["auth.no_user_interaction"] = dbus.MakeVariant(false)
	if err = devo.CallWithContext(
		ctx, udName+".Filesystem.Unmount", 0, options,
	).Store(); err != nil {
		return errors.Wrapf(err, "couldn't unmount block device %s", id)
	}
	return nil
}

func (c *Client) findBlockDeviceByID(
	ctx context.Context, id string,
) (devo dbus.BusObject, err error) {
	udm := c.getUDisks2Manager()
	devPaths := make([]dbus.ObjectPath, 0)
	options := make(map[string]dbus.Variant)
	options["auth.no_user_interaction"] = dbus.MakeVariant(false)
	if err = udm.CallWithContext(
		ctx, udName+".Manager.GetBlockDevices", 0, options,
	).Store(&devPaths); err != nil {
		return nil, errors.Wrap(err, "couldn't query for block devices")
	}
	for _, devPath := range devPaths {
		devo = c.bus.Object(udName, devPath)
		device, err := dumpBlockDevice(devo, c.bus)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't dump block device %s", devPath)
		}
		if device.ID == id {
			return devo, nil
		}
	}
	return nil, errors.Errorf("couldn't find block device with ID %s", id)
}

func (c *Client) MountBlockDevice(
	ctx context.Context, id string, asUser string,
) (mountedPath string, err error) {
	devo, err := c.findBlockDeviceByID(ctx, id)
	if err != nil {
		return "", err
	}

	options := make(map[string]dbus.Variant)
	options["auth.no_user_interaction"] = dbus.MakeVariant(false)
	if asUser != "" {
		// FIXME: for some reason, "as-user" doesn't seem to work (tested on Debian Bookworm)
		options["as-user"] = dbus.MakeVariant(asUser)
	}
	if err = devo.CallWithContext(
		ctx, udName+".Filesystem.Mount", 0, options,
	).Store(&mountedPath); err != nil {
		return "", errors.Wrapf(err, "couldn't mount block device %s", id)
	}
	return mountedPath, nil
}
