package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/location"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/bluemir/grim/internal/buildinfo"
	"github.com/bluemir/grim/internal/server/graceful"
	"github.com/bluemir/grim/internal/server/injector"
	"github.com/bluemir/grim/internal/server/middleware/cache"
	"github.com/bluemir/grim/internal/server/middleware/errs"
	"github.com/bluemir/grim/internal/server/middleware/prom"
)

func accessLogFormatter(param gin.LogFormatterParams) string {
	return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %#v\n%s",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		param.StatusCode,
		param.Latency,
		param.ClientIP,
		param.Method,
		param.Path,
		param.ErrorMessage,
	)
}

func (server *Server) RunServiceHTTPServer(ctx context.Context, bind string, tlsConf *tls.Config, extra ...gin.HandlerFunc) func() error {
	return func() error {
		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			gin.SetMode(gin.DebugMode)
		} else {
			gin.SetMode(gin.ReleaseMode)
		}

		// starting http server
		app := gin.New()

		// add template
		if html, err := NewRenderer(); err != nil {
			return err
		} else {
			app.SetHTMLTemplate(html)
		}

		// setup Logger
		writer := logrus.
			WithFields(logrus.Fields{}).
			WriterLevel(logrus.InfoLevel)
		defer writer.Close()
		app.Use(gin.LoggerWithConfig(gin.LoggerConfig{
			Formatter: accessLogFormatter,
			Output:    writer,
		}))

		// error handler
		app.Use(errs.Middleware)
		app.Use(gin.Recovery())

		// sessions
		store := cookie.NewStore([]byte(buildinfo.Signature))
		store.Options(sessions.Options{
			Path: "/",
		})
		app.Use(sessions.Sessions("grim_session", store))

		app.Use(location.Default(), fixURL)
		app.Use(cache.CacheBusting)

		app.Use(injector.InjectConfig(server.frontendConfig))
		app.Use(injector.Inject(server.backends))

		// prometheus for monitoring
		app.Use(prom.Metrics())

		// handle routes
		server.routes(app, app.NoRoute)

		// GRPC Gateway
		app.Use(extra...)
		// app.Group("/grpc/*any", extra...)

		return graceful.Run(ctx, &http.Server{
			Addr:              bind,
			Handler:           app,
			ReadHeaderTimeout: 1 * time.Minute,
			WriteTimeout:      3 * time.Minute,
			IdleTimeout:       3 * time.Minute,

			TLSConfig: tlsConf,
		})
	}
}
