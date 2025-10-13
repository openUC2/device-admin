// Package internet contains the route handlers related to internet access.
package internet

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"

	"github.com/openUC2/device-admin/internal/clients/networkmanager"
)

type Handlers struct {
	r godest.TemplateRenderer

	nmc *networkmanager.Client
}

func New(r godest.TemplateRenderer, nmc *networkmanager.Client) *Handlers {
	return &Handlers{
		r:   r,
		nmc: nmc,
	}
}

func (h *Handlers) Register(er godest.EchoRouter) {
	er.GET(h.r.BasePath+"internet", h.HandleInternetGet())
	er.POST(h.r.BasePath+"internet/wifi/networks", h.HandleWiFiNetworksPost())
}

type InternetViewData struct {
	SSIDs []string
}

func (h *Handlers) HandleInternetGet() echo.HandlerFunc {
	t := "internet/main.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Run queries
		internetViewData, err := getInternetViewData(h.nmc)
		if err != nil {
			return err
		}
		// Produce output
		return h.r.CacheablePage(c.Response(), c.Request(), t, internetViewData, struct{}{})
	}
}

func getInternetViewData(nmc *networkmanager.Client) (vd InternetViewData, err error) {
	vd.SSIDs, err = nmc.ScanNetworks()
	if err != nil {
		return vd, errors.Wrap(err, "couldn't scan for Wi-Fi networks")
	}
	return vd, nil
}

func (h *Handlers) HandleWiFiNetworksPost() echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse params
		state := c.FormValue("state")

		// Run queries
		switch state {
		default:
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf(
				"invalid Wi-Fi networks state %s", state,
			))
		case "refreshed":
			if err := h.nmc.RescanNetworks(); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, h.r.BasePath+"internet")
		}
	}
}
