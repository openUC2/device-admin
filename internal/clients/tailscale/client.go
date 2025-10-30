// Package tailscale provides an interface for the Tailscale daemon's local API.
package tailscale

import (
	"context"
	"fmt"

	"github.com/sargassum-world/godest"
	tcl "tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
)

type Client struct {
	Config Config

	ts *tcl.Client
	l  godest.Logger
}

func NewClient(c Config, l godest.Logger) *Client {
	ts := tcl.Client{}
	return &Client{
		Config: c,
		ts:     &ts,
		l:      l,
	}
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
