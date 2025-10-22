package internet

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	nm "github.com/openUC2/device-admin/internal/clients/networkmanager"
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
			if err := nm.ReloadConnProfiles(c.Request().Context()); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		}
	}
}

// by UUID

func (h *Handlers) HandleConnProfileGetByUUID() echo.HandlerFunc {
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
	ConnProfile nm.ConnProfile
	Active      *nm.ActiveConn
}

func getConnProfileViewData(
	ctx context.Context,
	uid uuid.UUID,
) (vd ConnProfileViewData, err error) {
	if vd.ConnProfile, err = nm.GetConnProfileByUUID(ctx, uid); err != nil {
		return vd, errors.Wrapf(err, "couldn't get connection profile %s", uid)
	}

	activeConns, err := nm.ListActiveConns(ctx)
	if err == nil { // vd.Active is nil if we can't determine the active conns
		activeConn := activeConns[vd.ConnProfile.Settings.Conn.UUID.String()]
		vd.Active = &(activeConn)
	}

	return vd, nil
}

func (h *Handlers) HandleConnProfilePostByUUID() echo.HandlerFunc {
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
			if err := nm.ActivateConnProfile(c.Request().Context(), uid); err != nil {
				return err
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		case "updated":
			formValues, err := c.FormParams()
			if err != nil {
				return errors.Wrap(err, "couldn't load form parameters")
			}
			if err := updateConnProfile(
				c.Request().Context(), uid, c.FormValue("update-type"), formValues,
			); err != nil {
				return errors.Wrapf(err, "couldn't update connection profile %s", rawUUID)
			}
			// Redirect user
			return c.Redirect(http.StatusSeeOther, redirectTarget)
		}
	}
}

func updateConnProfile(
	ctx context.Context, uid uuid.UUID, updateType string, formValues url.Values,
) error {
	updateValues := make(map[nm.ConnProfileSettingsKey]any)

	switch strings.ToLower(updateType) {
	default:
		return errors.Errorf("unknown update type: %s", updateType)
	case "apply temporarily":
		updateType = "apply"
	case "save and apply":
		updateType = "save"
	}

	for rawKey, values := range formValues {
		if len(values) < 1 {
			continue
		}
		key, err := nm.ParseConnProfileSettingsKey(rawKey)
		if err != nil {
			continue
		}
		rawValue := values[len(values)-1] // selects the last value to account for checkboxes
		switch key.Section {
		case "connection":
			if updateValues[key], err = parseConnProfileSettingsConnField(key, rawValue); err != nil {
				return errors.Wrapf(err, "couldn't parse (key, value) pair: (%s, %+v)", key, rawValue)
			}
		case "802-11-wireless":
			if updateValues[key], err = parseConnProfileSettingsWifiField(key, rawValue); err != nil {
				return errors.Wrapf(err, "couldn't parse (key, value) pair: (%s, %+v)", key, rawValue)
			}
		}
	}
	wifiChannel := formValues["802-11-wireless.channel"][0]
	wifiBand := formValues["802-11-wireless.band"][0]
	if wifiChannel != "0" && wifiBand == "" {
		return errors.Errorf("setting a non-zero channel (%s) requires setting a band", wifiChannel)
	}
	return nm.UpdateConnProfileByUUID(ctx, uid, updateType, updateValues)
}

func parseConnProfileSettingsConnField(
	key nm.ConnProfileSettingsKey, rawValue string,
) (parsedValue any, err error) {
	switch key.Key {
	default:
		return nil, errors.Errorf("unimplemented or unknown key %s", key)
	case "autoconnect":
		autoconnect, err := parseCheckbox(rawValue, "on", "off")
		if err != nil {
			return false, errors.Wrapf(err, "couldn't parse value for %s", key)
		}
		return autoconnect, nil
	case "autoconnect-priority":
		value, err := strconv.Atoi(rawValue)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't parse %s as integer", rawValue)
		}
		if value < -999 || value > 999 {
			return nil, errors.Errorf("autoconnect priority %d out of range [-999, 999]", value)
		}
		return value, nil
	}
}

func parseCheckbox(rawValue, checkedValue, uncheckedValue string) (parsedValue bool, err error) {
	switch rawValue {
	default:
		return false, errors.Errorf(
			"value %s must be '%s' or '%s'", rawValue, checkedValue, uncheckedValue,
		)
	case checkedValue:
		return true, nil
	case uncheckedValue:
		return false, nil
	}
}

func parseConnProfileSettingsWifiField(
	key nm.ConnProfileSettingsKey, rawValue string,
) (parsedValue any, err error) {
	switch key.Key {
	default:
		return nil, errors.Errorf("unimplemented or unknown key %s", key)
	case "band":
		band := nm.ConnProfileSettingsWifiBand(rawValue)
		if info := band.Info(); info.Level == "error" {
			return nil, errors.New(info.Details)
		}
		return band, nil
	case "channel":
		value, err := strconv.Atoi(rawValue)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't parse %s as integer", rawValue)
		}
		if value < -999 || value > 999 {
			return nil, errors.Errorf("autoconnect priority %d out of range [-999, 999]", value)
		}
		return value, nil
	case "hidden":
		hidden, err := parseCheckbox(rawValue, "true", "false")
		if err != nil {
			return false, errors.Wrapf(err, "couldn't parse value for %s", key)
		}
		return hidden, nil
	case "mode":
		mode := nm.ConnProfileSettingsWifiMode(rawValue)
		if info := mode.Info(); info.Level == "error" {
			return nil, errors.New(info.Details)
		}
		return mode, nil
	case "ssid":
		ssid := []byte(rawValue)
		const maxLen = 32
		if len(ssid) > maxLen {
			return nil, errors.Errorf("SSID %s is longer than %d bytes!", rawValue, maxLen)
		}
		return ssid, nil
	}
}
