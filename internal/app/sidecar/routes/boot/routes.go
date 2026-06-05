package boot

import (
	"context"

	"github.com/sargassum-world/godest"
	"github.com/varlink/go/varlink"

	ipc "github.com/openUC2/machine-admin/internal/app/ipc/boot"
	"github.com/openUC2/machine-admin/internal/app/sidecar/handling"
	sd "github.com/openUC2/machine-admin/internal/clients/systemd"
)

type Handlers struct {
	ipc.VarlinkInterface

	sdc *sd.Client

	l godest.Logger
}

func New(sdc *sd.Client, l godest.Logger) *Handlers {
	return &Handlers{
		sdc: sdc,
		l:   l,
	}
}

func (h *Handlers) Register(service *varlink.Service) error {
	return service.RegisterInterface(ipc.VarlinkNew(h))
}

func (h *Handlers) Poweroff(ctx context.Context, call ipc.VarlinkCall) error {
	handling.LogMethod(call.Request, h.l)

	if err := h.sdc.Poweroff(ctx); err != nil {
		return handling.ReportUnknownError(ctx, &call, err, h.l)
	}
	return call.ReplyPoweroff(ctx)
}

func (h *Handlers) Reboot(ctx context.Context, call ipc.VarlinkCall) error {
	handling.LogMethod(call.Request, h.l)

	if err := h.sdc.Reboot(ctx); err != nil {
		return handling.ReportUnknownError(ctx, &call, err, h.l)
	}
	return call.ReplyReboot(ctx)
}

func (h *Handlers) SoftReboot(ctx context.Context, call ipc.VarlinkCall) error {
	handling.LogMethod(call.Request, h.l)

	if err := h.sdc.SoftReboot(ctx); err != nil {
		return handling.ReportUnknownError(ctx, &call, err, h.l)
	}
	return call.ReplySoftReboot(ctx)
}
