package boot

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/varlink/go/varlink"

	ipc "github.com/openUC2/device-admin/internal/app/ipc/boot"
	sd "github.com/openUC2/device-admin/internal/clients/systemd"
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
	if call.Request != nil {
		var req struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal(*call.Request, &req); err == nil {
			h.l.Info(req.Method)
		}
	}

	if err := h.sdc.Poweroff(ctx); err != nil {
		if replyErr := call.ReplyError(
			ctx, "com.openuc2.deviceadmin.boot.Unknown", ipc.Unknown{Description: err.Error()},
		); replyErr != nil {
			h.l.Error(err)
			return errors.Wrapf(replyErr, "couldn't report error (%s) in method call reply", err.Error())
		}
	}
	return call.ReplyPoweroff(ctx)
}

func (h *Handlers) Reboot(ctx context.Context, call ipc.VarlinkCall) error {
	if call.Request != nil {
		var req struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal(*call.Request, &req); err == nil {
			h.l.Info(req.Method)
		}
	}

	if err := h.sdc.Reboot(ctx); err != nil {
		if replyErr := call.ReplyError(
			ctx, "com.openuc2.deviceadmin.boot.Unknown", ipc.Unknown{Description: err.Error()},
		); replyErr != nil {
			h.l.Error(err)
			return errors.Wrapf(replyErr, "couldn't report error (%s) in method call reply", err.Error())
		}
	}
	return call.ReplyReboot(ctx)
}

func (h *Handlers) SoftReboot(ctx context.Context, call ipc.VarlinkCall) error {
	if call.Request != nil {
		var req struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal(*call.Request, &req); err == nil {
			h.l.Info(req.Method)
		}
	}

	if err := h.sdc.SoftReboot(ctx); err != nil {
		if replyErr := call.ReplyError(
			ctx, "com.openuc2.deviceadmin.boot.Unknown", ipc.Unknown{Description: err.Error()},
		); replyErr != nil {
			h.l.Error(err)
			return errors.Wrapf(replyErr, "couldn't report error (%s) in method call reply", err.Error())
		}
	}
	return call.ReplySoftReboot(ctx)
}
