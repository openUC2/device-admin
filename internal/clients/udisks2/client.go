// Package udisks2 provides an interface for UDisks2 via its D-Bus API.
package udisks2

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"golang.org/x/sync/errgroup"
)

type Client struct {
	Config Config

	bus *dbus.Conn

	l godest.Logger
}

func NewClient(c Config, l godest.Logger) *Client {
	return &Client{
		Config: c,
		l:      l,
	}
}

func (c *Client) Open(ctx context.Context) (err error) {
	if c.bus, err = dbus.ConnectSystemBus(dbus.WithContext(ctx)); err != nil {
		return errors.Wrap(err, "couldn't connect to SystemBus bus to interact with NetworkManager")
	}
	return nil
}

func (c *Client) getUDisks2() dbus.BusObject {
	return c.bus.Object(udName, "/org/freedesktop/UDisks2")
}

const udName = "org.freedesktop.UDisks2"

func (c *Client) getUDisks2Manager() dbus.BusObject {
	return c.bus.Object(udName, "/org/freedesktop/UDisks2/Manager")
}

type Drive struct {
	Vendor      string
	Model       string
	FirmwareRev string
	SerialNum   string
	ID          string
	MediaType   string // TODO: make this an open-ended string enum
	// MediaRemovable bool
	// Size uint64
	Seat string
	// Removable bool
	SortKey string
}

func (c *Client) GetDrives(ctx context.Context) (drives []Drive, err error) {
	ud := c.getUDisks2()
	drivePaths, err := listDrives(ctx, ud)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't list drives")
	}
	for _, drivePath := range drivePaths {
		drive, err := dumpDrive(c.bus.Object(udName, drivePath))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't dump drive %s", drivePath)
		}
		drives = append(drives, drive)
	}
	slices.SortFunc(drives, func(a, b Drive) int {
		return cmp.Compare(a.SortKey, b.SortKey)
	})
	return drives, nil
}

func listDrives(ctx context.Context, ud dbus.BusObject) (drivePaths []dbus.ObjectPath, err error) {
	// Note: the org.freedesktop.UDisks2.Manager.GetDrives method is not implemented in the version of
	// UDisks2 provided with RPi OS 12 (bookworm), so we have to list the objects ourselves.
	objects := make(map[dbus.ObjectPath]map[string]map[string]dbus.Variant, 0)
	if err = ud.CallWithContext(
		ctx, "org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0,
	).Store(&objects); err != nil {
		return nil, errors.Wrap(err, "couldn't query for managed objects")
	}
	for objectPath := range objects {
		if !strings.HasPrefix(string(objectPath), "/org/freedesktop/UDisks2/drives/") {
			continue
		}
		drivePaths = append(drivePaths, objectPath)
	}
	slices.Sort(drivePaths)
	return drivePaths, nil
}

func dumpDrive(driveo dbus.BusObject) (drive Drive, err error) {
	if err = driveo.StoreProperty(udName+".Drive.Vendor", &drive.Vendor); err != nil {
		return Drive{}, errors.Wrap(err, "couldn't query for vendor")
	}
	if err = driveo.StoreProperty(udName+".Drive.Model", &drive.Model); err != nil {
		return Drive{}, errors.Wrap(err, "couldn't query for model")
	}
	if err = driveo.StoreProperty(udName+".Drive.Revision", &drive.FirmwareRev); err != nil {
		return Drive{}, errors.Wrap(err, "couldn't query for firmware revision")
	}
	if err = driveo.StoreProperty(udName+".Drive.Serial", &drive.SerialNum); err != nil {
		return Drive{}, errors.Wrap(err, "couldn't query for serial number")
	}
	if err = driveo.StoreProperty(udName+".Drive.Id", &drive.ID); err != nil {
		return Drive{}, errors.Wrap(err, "couldn't query for id")
	}
	if err = driveo.StoreProperty(udName+".Drive.Media", &drive.MediaType); err != nil {
		return Drive{}, errors.Wrap(err, "couldn't query for media type")
	}
	if err = driveo.StoreProperty(udName+".Drive.Seat", &drive.Seat); err != nil {
		return Drive{}, errors.Wrap(err, "couldn't query for seat")
	}
	if err = driveo.StoreProperty(udName+".Drive.SortKey", &drive.SortKey); err != nil {
		return Drive{}, errors.Wrap(err, "couldn't query for sort key")
	}

	return drive, nil
}

func (c *Client) UnmountDrive(ctx context.Context, id string) error {
	devs, err := c.GetBlockDevices(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't get block devices")
	}
	driveDevs := make([]BlockDevice, 0)
	for _, dev := range devs {
		if dev.Drive.ID != id || dev.ID == "" {
			continue
		}
		if len(dev.Filesystem.MountPoints) == 0 {
			continue
		}
		driveDevs = append(driveDevs, dev)
	}

	// We discard the context returned by WithContext so that a failure in unmounting one block device
	// will not prevent completion of unmounting other block devices:
	eg, _ := errgroup.WithContext(ctx)
	for _, dev := range driveDevs {
		eg.Go(func() error {
			if err := c.UnmountBlockDevice(ctx, dev.ID); err != nil {
				return errors.Wrapf(err, "couldn't unmount block device %s of drive %s", dev.ID, id)
			}
			return nil
		})
	}
	return eg.Wait()
}
