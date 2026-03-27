package networkmanager

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/varlink/go/varlink"

	ipc "github.com/openUC2/device-admin/internal/app/ipc/networkmanager"
	"github.com/openUC2/device-admin/internal/app/sidecar/handling"
	nm "github.com/openUC2/device-admin/internal/clients/networkmanager"
)

type Handlers struct {
	ipc.VarlinkInterface

	nmc *nm.Client

	l godest.Logger
}

func New(nmc *nm.Client, l godest.Logger) *Handlers {
	return &Handlers{
		nmc: nmc,
		l:   l,
	}
}

func (h *Handlers) Register(service *varlink.Service) error {
	return service.RegisterInterface(ipc.VarlinkNew(h))
}

func (h *Handlers) ReloadConnProfiles(ctx context.Context, call ipc.VarlinkCall) error {
	handling.LogMethod(call.Request, h.l)

	if err := h.nmc.ReloadConnProfiles(ctx); err != nil {
		return handling.ReportUnknownError(ctx, &call, err, h.l)
	}
	return call.ReplyReloadConnProfiles(ctx)
}

func (h *Handlers) ReloadConnProfile(
	ctx context.Context, call ipc.VarlinkCall, rawUUID string,
) error {
	handling.LogMethod(call.Request, h.l)

	uid, err := uuid.Parse(rawUUID)
	if err != nil {
		return handling.ReportUnknownError(ctx, &call, errors.Wrapf(
			err, "couldn't parse uuid %s", rawUUID,
		), h.l)
	}
	if err := h.nmc.ReloadConnProfile(ctx, uid); err != nil {
		return handling.ReportUnknownError(ctx, &call, err, h.l)
	}
	return call.ReplyReloadConnProfiles(ctx)
}
