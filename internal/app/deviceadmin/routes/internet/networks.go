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

func (h *Handlers) HandleConnProfilesGetByUUID() echo.HandlerFunc {
	t := "internet/connection-profiles/main.page.tmpl"
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
}

func getConnProfileViewData(
	ctx context.Context,
	uid uuid.UUID,
) (vd ConnProfileViewData, err error) {
	if vd.ConnProfile, err = networkmanager.GetConnProfile(ctx, uid); err != nil {
		return vd, errors.Wrapf(err, "couldn't get connection profile %s", uid)
	}

	return vd, nil
}
