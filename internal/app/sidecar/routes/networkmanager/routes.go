package networkmanager

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/varlink/go/varlink"

	ipc "github.com/openUC2/device-admin/internal/app/ipc/networkmanager"
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
	if call.Request != nil {
		var req struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal(*call.Request, &req); err == nil {
			h.l.Info(req.Method)
		}
	}

	if err := h.nmc.ReloadConnProfiles(ctx); err != nil {
		if replyErr := call.ReplyError(
			ctx, "com.openuc2.deviceadmin.networkmanager.Unknown", ipc.Unknown{Description: err.Error()},
		); replyErr != nil {
			h.l.Error(err)
			return errors.Wrapf(replyErr, "couldn't report error (%s) in method call reply", err.Error())
		}
		return err
	}
	return call.ReplyReloadConnProfiles(ctx)
}

func (h *Handlers) ReloadConnProfile(
	ctx context.Context, call ipc.VarlinkCall, rawUUID string,
) error {
	if call.Request != nil {
		var req struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal(*call.Request, &req); err == nil {
			h.l.Info(req.Method)
		}
	}

	uid, err := uuid.Parse(rawUUID)
	if err != nil {
		err = errors.Wrapf(err, "couldn't parse uuid %s", rawUUID)
		if replyErr := call.ReplyError(
			ctx, "com.openuc2.deviceadmin.networkmanager.InvalidUUID",
			ipc.InvalidUUID{Description: err.Error()},
		); replyErr != nil {
			h.l.Error(err)
			return errors.Wrapf(replyErr, "couldn't report error (%s) in method call reply", err.Error())
		}
		return err
	}
	if err := h.nmc.ReloadConnProfile(ctx, uid); err != nil {
		if replyErr := call.ReplyError(
			ctx, "com.openuc2.deviceadmin.networkmanager.Unknown", ipc.Unknown{Description: err.Error()},
		); replyErr != nil {
			h.l.Error(err)
			return errors.Wrapf(replyErr, "couldn't report error (%s) in method call reply", err.Error())
		}
		return err
	}
	return call.ReplyReloadConnProfiles(ctx)
}
