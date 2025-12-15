package deviceadmin

import (
	"fmt"
	"io/fs"
	"net/http"

	csrf "filippo.io/csrf/gorilla"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	"github.com/sargassum-world/godest/httperr"
)

type ErrorData struct {
	Code     int
	Error    httperr.DescriptiveError
	Messages []string
}

func NewHTTPErrorHandler(tr godest.TemplateRenderer, templatesFS fs.FS) echo.HTTPErrorHandler {
	tr.MustHave("app/httperr.page.tmpl")
	return func(err error, c echo.Context) {
		c.Logger().Error(err)

		// Process error code
		code := http.StatusInternalServerError
		if herr, ok := err.(*echo.HTTPError); ok {
			code = herr.Code
		}
		errorData := ErrorData{
			Code:  code,
			Error: httperr.Describe(code),
		}

		// Produce output
		perr := tr.Page(
			c.Response(), c.Request(), code, "app/httperr.page.tmpl", errorData, struct{}{},
			godest.WithUncacheable(),
		)
		if perr != nil {
			c.Logger().Error(errors.Wrap(perr, "couldn't render templated error page in error handler"))
			fallbackErrorPage, ferr := fs.ReadFile(templatesFS, "app/httperr.html")
			if ferr != nil {
				c.Logger().Error(errors.Wrap(perr, "couldn't load fallback error page in error handler"))
			}
			perr = c.HTML(http.StatusInternalServerError, string(fallbackErrorPage))
			if perr != nil {
				c.Logger().Error(errors.Wrap(perr, "couldn't send fallback error page in error handler"))
			}
		}
	}
}

func NewCSRFErrorHandler(
	tr godest.TemplateRenderer, l echo.Logger,
) http.HandlerFunc {
	tr.MustHave("app/httperr.page.tmpl")
	return func(w http.ResponseWriter, r *http.Request) {
		l.Error(csrf.FailureReason(r))
		// Generate error code
		code := http.StatusForbidden
		errorData := ErrorData{
			Code:  code,
			Error: httperr.Describe(code),
			Messages: []string{
				fmt.Sprintf(
					"%s. If you disabled Javascript after signing in, "+
						"please clear your cookies for this site and sign in again.",
					csrf.FailureReason(r).Error(),
				),
			},
		}

		// Produce output

		if rerr := tr.Page(
			w, r, code, "app/httperr.page.tmpl", errorData, struct{}{}, godest.WithUncacheable(),
		); rerr != nil {
			l.Error(errors.Wrap(rerr, "couldn't render error page in error handler"))
		}
	}
}
