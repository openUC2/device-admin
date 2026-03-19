package networkmanager

import (
	"context"
	"encoding/json"

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

func (h *Handlers) ReloadConnections(ctx context.Context, call ipc.VarlinkCall) error {
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
	return call.ReplyReloadConnections(ctx)
}
