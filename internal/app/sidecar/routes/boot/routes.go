package boot

import (
	"context"
	"fmt"

	"github.com/sargassum-world/godest"
	"github.com/varlink/go/varlink"

	ipc "github.com/openUC2/device-admin/internal/app/ipc/boot"
)

type Handlers struct {
	ipc.VarlinkInterface

	l godest.Logger
}

func New(l godest.Logger) *Handlers {
	return &Handlers{
		l: l,
	}
}

func (h *Handlers) Register(service *varlink.Service) error {
	return service.RegisterInterface(ipc.VarlinkNew(h))
}

func (h *Handlers) Poweroff(ctx context.Context, call ipc.VarlinkCall) error {
	fmt.Println("TODO: implement poweroff!")
	return call.ReplyPoweroff(ctx)
}

func (h *Handlers) Reboot(ctx context.Context, call ipc.VarlinkCall) error {
	fmt.Println("TODO: implement poweroff!")
	return call.ReplyReboot(ctx)
}

func (h *Handlers) SoftReboot(ctx context.Context, call ipc.VarlinkCall) error {
	fmt.Println("TODO: implement poweroff!")
	return call.ReplySoftReboot(ctx)
}
