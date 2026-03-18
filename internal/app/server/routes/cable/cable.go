package cable

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/sargassum-world/godest/actioncable"
	"github.com/sargassum-world/godest/handling"
	"github.com/sargassum-world/godest/turbostreams"
)

func serveWSConn(
	r *http.Request, wsc *websocket.Conn, channelFactories map[string]actioncable.ChannelFactory,
	wsu websocket.Upgrader,
	l godest.Logger,
) {
	conn, err := actioncable.Upgrade(wsc, actioncable.NewChannelDispatcher(
		channelFactories, make(map[string]actioncable.Channel),
	))
	if err != nil {
		l.Error(errors.Wrapf(
			err,
			"couldn't upgrade websocket connection to action cable connection "+
				"(client requested subprotocols %v, upgrader supports subprotocols %v)",
			websocket.Subprotocols(r),
			wsu.Subprotocols,
		))
		if cerr := wsc.Close(); cerr != nil {
			l.Error(errors.Wrapf(cerr, "couldn't close websocket"))
		}
		return
	}

	ctx := r.Context()
	serr := handling.Except(conn.Serve(ctx), context.Canceled)
	if serr != nil {
		// We can't return errors after the HTTP request is upgraded to a websocket, so we just log them
		l.Error(serr)
	}
	if err = conn.Close(serr); err != nil {
		// We can't return errors after the HTTP request is upgraded to a websocket, so we just log them
		l.Error(err)
	}
}

func (h *Handlers) HandleCableGet() echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: ensure that we do a CSRF check if/when users can send data over Action Cable
		wsc, err := h.wsu.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return errors.Wrap(err, "couldn't upgrade http request to websocket connection")
		}

		const wsMaxMessageSize = 512
		wsc.SetReadLimit(wsMaxMessageSize)
		sessionID := "global" // since we have no concept of user identity yet
		serveWSConn(
			c.Request(), wsc,
			map[string]actioncable.ChannelFactory{
				turbostreams.ChannelName: turbostreams.NewChannelFactory(h.tsb, sessionID, h.acs.Check),
			},
			h.wsu, h.l,
		)
		return nil
	}
}
