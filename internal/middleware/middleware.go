package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

// Logger - middleware that logs requests
func Logger(ctx context.Context, next http.Handler) http.Handler {
	c := alice.New()

	log := zerolog.Ctx(ctx)
	// Install the logger handler with default output on the console
	c = c.Append(hlog.NewHandler(*log))
	c = c.Append(hlog.URLHandler("path"))
	c = c.Append(hlog.MethodHandler("method"))
	c = c.Append(hlog.RemoteAddrHandler("client_ip"))
	c = c.Append(hlog.UserAgentHandler("user_agent"))
	c = c.Append(hlog.RefererHandler("referer"))
	// Install some provided extra handler to set some request's context fields
	// Thanks to that handler, all our logs will come with some prepopulated fields
	c = c.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		l := zerolog.DebugLevel
		if status == 200 {
			l = zerolog.InfoLevel
		}
		if status > 399 {
			l = zerolog.WarnLevel
		}
		if status > 499 {
			l = zerolog.ErrorLevel
		}
		if status == 429 {
			// Rate-limited request floods should only show when debugging
			l = zerolog.DebugLevel
		}
		switch r.URL.String() {
		case "/metrics", "/live", "/ready":
			l = zerolog.DebugLevel
		}
		hlog.FromRequest(r).WithLevel(l).
			Str("host", r.Host).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Str("method", r.Method).
			Stringer("url", r.URL).
			Msg("")
	}))
	return c.Then(next)
}
