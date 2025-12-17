// Package storage contains the route handlers related to storage devices.
package storage

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/sargassum-world/godest/handling"
	"github.com/sargassum-world/godest/turbostreams"

	dah "github.com/openUC2/device-admin/internal/app/deviceadmin/handling"
	du "github.com/openUC2/device-admin/internal/clients/diskusage"
	ud "github.com/openUC2/device-admin/internal/clients/udisks2"
)

type Handlers struct {
	r   godest.TemplateRenderer
	udc *ud.Client

	l godest.Logger
}

func New(r godest.TemplateRenderer, udc *ud.Client, l godest.Logger) *Handlers {
	return &Handlers{
		r:   r,
		udc: udc,
		l:   l,
	}
}

func (h *Handlers) Register(er godest.EchoRouter, tr turbostreams.Router) {
	er.GET(h.r.BasePath+"storage", h.HandleStorageGet())
	tr.SUB(h.r.BasePath+"storage", dah.AllowTSSub())
	tr.PUB(h.r.BasePath+"storage", h.HandleStoragePub())
	// drives
	er.POST(h.r.BasePath+"storage/drives/:id", h.HandleDrivePostByID())
	// block devices
	er.POST(h.r.BasePath+"storage/block-devices/:id", h.HandleBlockDevicePostByID())
}

func (h *Handlers) HandleStorageGet() echo.HandlerFunc {
	t := "storage/index.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Run queries
		remoteViewData, err := getStorageViewData(c.Request().Context(), h.l)
		if err != nil {
			return err
		}
		// Produce output
		return h.r.CacheablePage(c.Response(), c.Request(), t, remoteViewData, struct{}{})
	}
}

type StorageViewData struct {
	SystemDrives    []ud.Drive
	RemovableDrives []ud.Drive
	BlockDevices    map[string][]ud.BlockDevice    // keyed by drive ID
	DiskUsages      map[string]map[string]du.Usage // keyed by drive ID, then mount point

	IsStreamPage bool
}

func getStorageViewData(ctx context.Context, l godest.Logger) (vd StorageViewData, err error) {
	drives, err := ud.GetDrives(ctx)
	if err != nil {
		return vd, errors.Wrap(err, "couldn't list storage drives")
	}
	blockDevs, err := ud.GetBlockDevices(ctx)
	if err != nil {
		return vd, errors.Wrap(err, "couldn't list block devices")
	}
	vd.BlockDevices = make(map[string][]ud.BlockDevice)
	vd.DiskUsages = make(map[string]map[string]du.Usage)
	for _, dev := range blockDevs {
		vd.BlockDevices[dev.Drive.ID] = append(vd.BlockDevices[dev.Drive.ID], dev)
		vd.DiskUsages[dev.Drive.ID] = make(map[string]du.Usage)
		for _, mp := range dev.Filesystem.MountPoints {
			if vd.DiskUsages[dev.Drive.ID][mp], err = du.GetUsage(mp); err != nil {
				l.Warn(errors.Wrapf(err, "couldn't check disk usage of %s", mp))
			}
		}
	}

	isSystem := make(map[string]bool)
	for _, dev := range blockDevs {
		for _, mp := range dev.Filesystem.MountPoints {
			switch mp {
			case "/":
				isSystem[dev.Drive.ID] = true
			case "/boot", "/boot/firmware":
				isSystem[dev.Drive.ID] = true
			}
		}
	}
	for _, drive := range drives {
		if isSystem[drive.ID] {
			vd.SystemDrives = append(vd.SystemDrives, drive)
			continue
		}
		vd.RemovableDrives = append(vd.RemovableDrives, drive)
	}
	return vd, nil
}

func (h *Handlers) HandleStoragePub() turbostreams.HandlerFunc {
	t := "storage/index.page.tmpl"
	h.r.MustHave(t)
	return func(c *turbostreams.Context) error {
		// Publish periodically
		const pubInterval = 4 * time.Second
		return handling.RepeatImmediate(c.Context(), pubInterval, func() (done bool, err error) {
			// Run queries
			vd, err := getStorageViewData(c.Context(), h.l)
			if err != nil {
				return false, err
			}
			// Produce output
			vd.IsStreamPage = true
			return false, dah.PublishPageReload(c, h.r, t, vd)
		})
	}
}

func (h *Handlers) HandleDrivePostByID() echo.HandlerFunc {
	t := "internet/conn-profiles/index.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Parse params
		id := c.Param("id")
		state := c.FormValue("state")
		redirectTarget := c.FormValue("redirect-target")

		// Run queries
		switch state {
		default:
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid drive state %s", state))
		case "unmounted":
			if err := ud.UnmountDrive(c.Request().Context(), id); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		}
	}
}

func (h *Handlers) HandleBlockDevicePostByID() echo.HandlerFunc {
	t := "internet/conn-profiles/index.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Parse params
		id := c.Param("id")
		state := c.FormValue("state")
		redirectTarget := c.FormValue("redirect-target")

		// Run queries
		switch state {
		default:
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid device state %s", state))
		case "unmounted":
			if err := ud.UnmountBlockDevice(c.Request().Context(), id); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		case "mounted":
			if _, err := ud.MountBlockDevice(c.Request().Context(), id, ""); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		}
	}
}
