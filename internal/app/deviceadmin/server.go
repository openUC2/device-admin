// Package deviceadmin provides the ImSwitch OS device-admin server.
package deviceadmin

import (
	"context"
	"fmt"
	"net/http"

	csrf "filippo.io/csrf/gorilla"
	"github.com/Masterminds/sprig/v3"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/pkg/errors"
	"github.com/sargassum-world/godest"
	gmw "github.com/sargassum-world/godest/middleware"
	"github.com/unrolled/secure"
	"github.com/unrolled/secure/cspbuilder"
	"golang.org/x/sync/errgroup"

	"github.com/openUC2/device-admin/internal/app/deviceadmin/client"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/conf"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/routes/assets"
	"github.com/openUC2/device-admin/internal/app/deviceadmin/tmplfunc"
	"github.com/openUC2/device-admin/web"
)

type Server struct {
	Globals  *client.Globals
	Embeds   godest.Embeds
	Inlines  godest.Inlines
	Renderer godest.TemplateRenderer
	Handlers *routes.Handlers
}

func NewServer(config conf.Config, logger godest.Logger) (s *Server, err error) {
	s = &Server{}
	if s.Globals, err = client.NewGlobals(config, logger); err != nil {
		return nil, errors.Wrap(err, "couldn't make app globals")
	}

	s.Embeds = web.NewEmbeds()
	templatesOverlay := &OverlayFS{
		Upper: s.Globals.Base.Templates.GetFS(),
		Lower: s.Embeds.TemplatesFS,
	}
	s.Embeds.TemplatesFS = templatesOverlay
	s.Inlines = web.NewInlines()
	if s.Renderer, err = godest.NewLazyTemplateRenderer(
		s.Embeds, s.Inlines, sprig.FuncMap(), tmplfunc.FuncMap(
			tmplfunc.NewHashedNamers(assets.AppURLPrefix, assets.StaticURLPrefix, s.Embeds),
			s.Globals.Base.ACSigner.Sign,
		),
	); err != nil {
		return nil, errors.Wrap(err, "couldn't make template renderer")
	}
	s.Renderer.BasePath = s.Globals.Config.HTTP.BasePath

	s.Handlers = routes.New(s.Renderer, s.Globals)
	return s, err
}

// Echo

func (s *Server) configureLogging(e *echo.Echo) {
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			err := ""
			if v.Error != nil {
				err = v.Error.Error()
			}
			fmt.Printf("%s %s %s => (%d after %s) %d %s\n",
				v.Method, v.URI, v.RemoteIP, v.ResponseSize, v.Latency, v.Status, err,
			)
			return nil
		},
		LogLatency:      true,
		LogRemoteIP:     true,
		LogMethod:       true,
		LogURI:          true,
		LogStatus:       true,
		LogError:        true,
		LogResponseSize: true,
	}))
	e.HideBanner = true
	e.HidePort = true
	e.Logger.SetLevel(log.INFO) // TODO: set level via env var
}

// turboDriveStyle is the stylesheet which Turbo Drive tries to install for its progress bar,
// assuming ProgressBar.animationDuration == 300, for computing a CSP hash for inline styles.
const turboDriveStyle = `.turbo-progress-bar {
  position: fixed;
  display: block;
  top: 0;
  left: 0;
  height: 3px;
  background: #0076ff;
  z-index: 2147483647;
  transition:
    width 300ms ease-out,
    opacity 150ms 150ms ease-in;
  transform: translate3d(0, 0, 0);
}
`

func (s *Server) configureHeaders(e *echo.Echo) error {
	cspBuilder := cspbuilder.Builder{
		Directives: map[string][]string{
			cspbuilder.DefaultSrc: {"'self'"},
			// Note: the following is needed for the Tailscale web GUI to check device status (but the GUI
			// still works without this permission, it just doesn't report device status):
			cspbuilder.ConnectSrc: {"*"},
			// Note: script-src "unsafe-inline" (which is ignored if we provide one or more hashes for
			// CSP) is needed by the Tailscale web GUI:
			cspbuilder.ScriptSrc: {"'self'", "'unsafe-inline'"},
			// cspbuilder.ScriptSrc: append(
			// 	// Warning: script-src 'self' may not be safe to use if we're hosting user-uploaded content.
			// 	// Then we'll need to provide hashes for scripts & styles we include by URL, and we'll need
			// 	// to add the SRI integrity attribute to the tags including those files; however, it's
			// 	// unclear how well-supported they are by browsers.
			// 	[]string{"'self'", "'unsafe-inline'"},
			// 	s.Inlines.ComputeJSHashesForCSP()...,
			// ),
			cspbuilder.StyleSrc: append(
				[]string{
					"'self'",
					"'unsafe-inline'",
					godest.ComputeCSPHash([]byte(turboDriveStyle)),
				},
				s.Inlines.ComputeCSSHashesForCSP()...,
			),
			cspbuilder.ObjectSrc: {"'none'"},
			cspbuilder.ChildSrc:  {"'self'"},
			// Note: img-src with scheme "data:" is needed by the Tailscale web GUI:
			cspbuilder.ImgSrc:         {"*", "data:"},
			cspbuilder.BaseURI:        {"'none'"},
			cspbuilder.FormAction:     {"'self'"},
			cspbuilder.FrameAncestors: {"'self'"},
			// TODO: add HTTPS-related settings for CSP, including upgrade-insecure-requests
		},
	}
	csp, err := cspBuilder.Build()
	if err != nil {
		return errors.Wrap(err, "couldn't build content security policy")
	}

	e.Use(echo.WrapMiddleware(secure.New(secure.Options{
		// TODO: add HTTPS options
		FrameDeny:               true,
		ContentTypeNosniff:      true,
		ContentSecurityPolicy:   csp,
		ReferrerPolicy:          "no-referrer",
		CrossOriginOpenerPolicy: "same-origin",
	}).Handler))
	e.Use(echo.WrapMiddleware(gmw.SetCORP("same-site")))
	e.Use(echo.WrapMiddleware(gmw.SetCOEP("require-corp")))
	return nil
}

func (s *Server) Register(e *echo.Echo) error {
	e.Use(middleware.Recover())
	s.configureLogging(e)
	if err := s.configureHeaders(e); err != nil {
		return errors.Wrap(err, "couldn't configure http headers")
	}

	// Compression Middleware
	e.Use(middleware.Decompress())
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level:   s.Globals.Config.HTTP.GzipLevel,
		Skipper: s.Handlers.GzipSkipper,
	}))

	// Other Middleware
	e.Pre(middleware.RemoveTrailingSlashWithConfig(middleware.TrailingSlashConfig{
		Skipper: s.Handlers.TrailingSlashSkipper,
	}))
	e.Use(echo.WrapMiddleware(
		csrf.Protect(nil, csrf.ErrorHandler(NewCSRFErrorHandler(s.Renderer, e.Logger)))))
	// application/JSON is needed by the Tailscale web GUI:
	e.Use(gmw.RequireContentTypes(echo.MIMEApplicationForm, echo.MIMEApplicationJSON))
	// TODO: enable Prometheus and rate-limiting

	// Handlers
	e.HTTPErrorHandler = NewHTTPErrorHandler(s.Renderer, s.Embeds.TemplatesFS)
	if err := s.Handlers.Register(e, s.Globals.Base.TSBroker, s.Embeds); err != nil {
		return errors.Wrap(err, "couldn't register HTTP route handlers")
	}

	return nil
}

// Running

func (s *Server) Run(e *echo.Echo) error {
	s.Globals.Base.Logger.Info("starting device-admin server")

	// The echo http server can't be canceled by context cancelation, so the API shouldn't promise to
	// stop blocking execution on context cancelation - so we use the background context here. The
	// http server should instead be stopped gracefully by calling the Shutdown method, or forcefully
	// by calling the Close method.
	eg, egctx := errgroup.WithContext(context.Background())
	eg.Go(func() error {
		s.Globals.Base.Logger.Info("starting background workers")
		if err := s.runWorkersInContext(egctx); err != nil {
			s.Globals.Base.Logger.Error(errors.Wrap(
				err, "background worker encountered error",
			))
		}
		return nil
	})
	eg.Go(func() error {
		address := fmt.Sprintf(":%d", s.Globals.Config.HTTP.Port)
		s.Globals.Base.Logger.Infof("starting http server on %s", address)
		return e.Start(address)
	})
	if err := eg.Wait(); err != http.ErrServerClosed {
		return errors.Wrap(err, "http server encountered error")
	}
	return nil
}

func (s *Server) runWorkersInContext(ctx context.Context) error {
	eg, _ := errgroup.WithContext(ctx) // Workers run independently, so we don't need egctx
	eg.Go(func() error {
		if err := s.Globals.Base.TSBroker.Serve(ctx); err != nil && err != context.Canceled {
			s.Globals.Base.Logger.Error(errors.Wrap(
				err, "turbo streams broker encountered error while serving",
			))
		}
		return nil
	})
	eg.Go(func() error {
		if err := s.Globals.NetworkManager.Open(ctx); err != nil {
			s.Globals.Base.Logger.Error("couldn't open NetworkManager client")
			// Even if NetworkManager is unavailable, other parts of device-admin are still useful
		}
		return nil
	})
	return eg.Wait()
}

func (s *Server) Shutdown(ctx context.Context, e *echo.Echo) (err error) {
	// FIXME: e.Shutdown calls e.Server.Shutdown, which doesn't wait for WebSocket connections. When
	// starting Echo, we need to call e.Server.RegisterOnShutdown with a function to gracefully close
	// WebSocket connections!
	s.Globals.Tailscale.Shutdown()
	if errEcho := e.Shutdown(ctx); errEcho != nil {
		s.Globals.Base.Logger.Error(errors.Wrap(errEcho, "couldn't shut down http server"))
		err = errEcho
	}
	return err
}

func (s *Server) Close(e *echo.Echo) error {
	return errors.Wrap(e.Close(), "http server encountered error when closing an underlying listener")
}
