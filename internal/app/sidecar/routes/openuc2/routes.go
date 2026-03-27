package openuc2

import (
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/varlink/go/varlink"

	ipc "github.com/openUC2/device-admin/internal/app/ipc/openuc2"
	"github.com/openUC2/device-admin/internal/app/sidecar/handling"
	sd "github.com/openUC2/device-admin/internal/clients/systemd"
)

type Handlers struct {
	ipc.VarlinkInterface

	sdc *sd.Client

	l godest.Logger
}

func New(sdc *sd.Client, l godest.Logger) *Handlers {
	return &Handlers{
		sdc: sdc,
		l:   l,
	}
}

func (h *Handlers) Register(service *varlink.Service) error {
	return service.RegisterInterface(ipc.VarlinkNew(h))
}

func (h *Handlers) UpdatePSKDropInFile(
	ctx context.Context, call ipc.VarlinkCall, connProfile string, newPw string,
) error {
	handling.LogMethod(call.Request, h.l)

	dropInDir := path.Join("/etc/NetworkManager/system-connections.d", connProfile)
	fsys, err := os.OpenRoot(dropInDir)
	if err != nil {
		return errors.Wrapf(err, "couldn't open drop-in directory %s", dropInDir)
	}
	const dropInFile = "51-wifi-security-password.nmconnection"
	lines, err := readLines(fsys, dropInFile)
	if err != nil {
		return handling.ReportUnknownError(ctx, &call, errors.Wrapf(
			err, "couldn't read PSK drop-in file %s", path.Join(dropInDir, dropInFile),
		), h.l)
	}

	if lines, err = setKey(lines, "psk", newPw); err != nil {
		return handling.ReportUnknownError(ctx, &call, errors.Wrapf(
			err, "couldn't update PSK for drop-in file %s", path.Join(dropInDir, dropInFile),
		), h.l)
	}

	const mode = 0o600 // -rw-------
	if err = writeAtomically(fsys, dropInFile, lines, mode); err != nil {
		return handling.ReportUnknownError(ctx, &call, errors.Wrapf(
			err, "couldn't atomically write updated drop-in file %s", path.Join(dropInDir, dropInFile),
		), h.l)
	}

	return call.ReplyUpdatePSKDropInFile(ctx)
}

func readLines(fsys *os.Root, filePath string) ([]string, error) {
	contents, err := fsys.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't read file %s", filePath)
	}

	return strings.Split(string(contents), "\n"), nil
}

func setKey(lines []string, key string, newValue string) ([]string, error) {
	pattern := fmt.Sprintf("^%s[ ]*=", key)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't compile regexp for key %s", key)
	}
	setLine := fmt.Sprintf("%s=%s", key, newValue)

	hasKey := false
	for i, line := range lines {
		if !re.MatchString(line) {
			continue
		}
		hasKey = true
		lines[i] = setLine
	}
	if !hasKey {
		lines = append(lines, setLine)
	}
	return lines, nil
}

func writeAtomically(fsys *os.Root, filePath string, lines []string, perm os.FileMode) error {
	data := []byte(strings.Join(lines, "\n"))
	swapFilePath := filePath + ".swp"
	if err := fsys.WriteFile(swapFilePath, data, perm.Perm()); err != nil {
		return errors.Wrapf(
			err, "couldn't write drop-in file %s to swap file %s", filePath, swapFilePath,
		)
	}
	if err := fsys.Rename(swapFilePath, filePath); err != nil {
		return errors.Wrapf(
			err, "couldn't move temporary drop-in file %s to %s", swapFilePath, filePath,
		)
	}
	return nil
}

func (h *Handlers) RegenerateDropInConnProfile(
	ctx context.Context, call ipc.VarlinkCall, connProfile string,
) error {
	handling.LogMethod(call.Request, h.l)

	templatedAssembleUnit := fmt.Sprintf(
		"assemble-networkmanager-connection-templated@%s.service", connProfile,
	)
	hasTemplatedAssemble, err := h.sdc.UnitExists(ctx, templatedAssembleUnit)
	if err != nil {
		return handling.ReportUnknownError(ctx, &call, errors.Wrapf(
			err, "couldn't check whether templated drop-in assembly service %s exists for %s",
			templatedAssembleUnit, connProfile,
		), h.l)
	}
	if hasTemplatedAssemble {
		if err := h.sdc.RestartUnit(ctx, templatedAssembleUnit); err != nil {
			return handling.ReportUnknownError(ctx, &call, errors.Wrapf(
				err, "couldn't restart templated drop-in assembly service %s for %s",
				templatedAssembleUnit, connProfile,
			), h.l)
		}
	}

	assembleUnit := fmt.Sprintf("assemble-networkmanager-connection@%s.service", connProfile)
	if err := h.sdc.RestartUnit(ctx, assembleUnit); err != nil {
		return handling.ReportUnknownError(ctx, &call, errors.Wrapf(
			err, "couldn't restart drop-in assembly service %s for %s", assembleUnit, connProfile,
		), h.l)
	}

	return call.ReplyRegenerateDropInConnProfile(ctx)
}
