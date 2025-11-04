// Package tailscale provides an interface for the Tailscale daemon's local API.
package tailscale

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	tcl "tailscale.com/client/local"
	tsw "tailscale.com/client/web"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
)

type Client struct {
	Config Config

	ts   *tcl.Client
	tsws *tsw.Server
	l    godest.Logger
}

func NewClient(c Config, l godest.Logger) *Client {
	ts := tcl.Client{}
	return &Client{
		Config: c,
		ts:     &ts,
		l:      l,
	}
}

func (c *Client) InitWebServer(basePath string) (tsws *tsw.Server, err error) {
	if c.tsws, err = tsw.NewServer(tsw.ServerOpts{
		Mode:        tsw.LoginServerMode,
		CGIMode:     true,
		PathPrefix:  basePath,
		LocalClient: c.ts,
		Logf:        c.l.Printf,
	}); err != nil {
		return nil, errors.Wrap(err, "couldn't initialize server for Tailscale web GUI")
	}
	return c.tsws, nil
}

func (c *Client) Shutdown() {
	if c.tsws != nil {
		c.tsws.Shutdown()
	}
}

func (c *Client) Provision(ctx context.Context, deviceAuthKey string) error {
	prefs, err := c.ts.GetPrefs(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't check current Tailscale preferences")
	}
	if err = c.ts.CheckPrefs(ctx, prefs); err != nil {
		return errors.Wrap(err, "couldn't validate Tailscale preferences")
	}
	prefs.WantRunning = true
	if err := c.ts.Start(ctx, ipn.Options{
		AuthKey:     deviceAuthKey,
		UpdatePrefs: prefs,
	}); err != nil {
		return errors.Wrap(err, "couldn't make Tailscale start with the provided device auth key")
	}
	if err := c.ts.StartLoginInteractive(ctx); err != nil {
		return err
	}
	return nil
}

func (c *Client) Deprovision(ctx context.Context) error {
	return c.ts.Logout(ctx)
}

func (c *Client) GetStatus(ctx context.Context) (status *ipnstate.Status, err error) {
	return c.ts.Status(ctx)
}

// State

type State string

var stateInfo = map[State]EnumInfo{
	"NoState": {
		Short: "none",
		Level: "error",
	},
	"InUseOtherUser": {
		Short:   "other user",
		Details: "already in use by another user",
		Level:   "error",
	},
	"NeedsLogin": {
		Short:   "needs login",
		Details: "requires further action to log in",
		Level:   "warning",
	},
	"NeedsMachineAuth": {
		Short:   "needs login",
		Details: "requires further action to authenticate/authorize the machine",
		Level:   "warning",
	},
	"Stopped": {
		Short: "stopped",
		Level: "info",
	},
	"Starting": {
		Short: "starting",
		Level: "info",
	},
	"Running": {
		Short: "running",
		Level: "success",
	},
}

func (s State) Info() EnumInfo {
	info, ok := stateInfo[s]
	if !ok {
		return EnumInfo{
			Short:   "unknown",
			Details: fmt.Sprintf("state (%s) was reported but could not be determined", s),
			Level:   "error",
		}
	}
	return info
}

// EnumInfo

type EnumInfo struct {
	Short   string
	Details string
	Level   string
}

const EnumInfoLevelError = "error"
