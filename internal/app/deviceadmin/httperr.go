package deviceadmin

import (
	"io/fs"
	"net/http"

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
