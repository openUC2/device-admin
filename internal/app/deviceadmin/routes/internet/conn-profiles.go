package internet

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/openUC2/device-admin/internal/clients/networkmanager"
)

func (h *Handlers) HandleConnProfilesPost() echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse params
		state := c.FormValue("state")
		redirectTarget := c.FormValue("redirect-target")

		// Run queries
		switch state {
		default:
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf(
				"invalid connection profiles state %s", state,
			))
		case "reloaded":
			if err := networkmanager.ReloadConnProfiles(c.Request().Context()); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		}
	}
}

// by UUID

func (h *Handlers) HandleConnProfilesGetByUUID() echo.HandlerFunc {
	t := "internet/conn-profiles/index.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Parse params
		rawUUID := c.Param("uuid")
		uid, err := uuid.Parse(rawUUID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("unparsable UUID %s", rawUUID))
		}

		// Run queries
		vd, err := getConnProfileViewData(c.Request().Context(), uid)
		if err != nil {
			return err
		}

		// Produce output
		// Note: we don't cache this page because it's slower to serialize the data to cache than it is
		// to just send the page over the network
		return h.r.Page(c.Response(), c.Request(), http.StatusOK, t, vd, struct{}{})
	}
}

type ConnProfileViewData struct {
	ConnProfile networkmanager.ConnProfile
	Active      *networkmanager.ActiveConn
}

func getConnProfileViewData(
	ctx context.Context,
	uid uuid.UUID,
) (vd ConnProfileViewData, err error) {
	if vd.ConnProfile, err = networkmanager.GetConnProfileByUUID(ctx, uid); err != nil {
		return vd, errors.Wrapf(err, "couldn't get connection profile %s", uid)
	}

	activeConns, err := networkmanager.ListActiveConns(ctx)
	if err == nil { // vd.Active is nil if we can't determine the active conns
		activeConn := activeConns[vd.ConnProfile.Settings.Connection.UUID.String()]
		vd.Active = &(activeConn)
	}

	return vd, nil
}

func (h *Handlers) HandleConnProfilesPostByUUID() echo.HandlerFunc {
	t := "internet/conn-profiles/index.page.tmpl"
	h.r.MustHave(t)
	return func(c echo.Context) error {
		// Parse params
		rawUUID := c.Param("uuid")
		uid, err := uuid.Parse(rawUUID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("unparsable UUID %s", rawUUID))
		}
		state := c.FormValue("state")
		redirectTarget := c.FormValue("redirect-target")

		// Run queries
		switch state {
		default:
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf(
				"invalid connection profiles state %s", state,
			))
		case "activated-transiently":
			if err := networkmanager.ActivateConnProfile(c.Request().Context(), uid); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		}
	}
}
