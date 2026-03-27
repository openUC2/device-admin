// Package handling provides utilities for handlers
package handling

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
)

func LogMethod(request *[]byte, l godest.Logger) {
	if request != nil {
		var req struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal(*request, &req); err == nil {
			l.Info(req.Method)
		}
	}
}

type UnknownErrorReplier interface {
	ReplyUnknown(ctx context.Context, description string) error
}

func ReportUnknownError(
	ctx context.Context, errReplier UnknownErrorReplier, err error, l godest.Logger,
) error {
	l.Error(err)
	if replyErr := errReplier.ReplyUnknown(ctx, err.Error()); replyErr != nil {
		return errors.Wrapf(replyErr, "couldn't report error (%s)", err.Error())
	}
	return nil
}
