package bot

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/karl-johan-grahn/devopsbot/internal/middleware"
	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
)

// mwVerify - middleware to verify incoming request against the signing secret,
// to prevent man-in-the-middle attacks
func mwVerify(signingSecret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := zerolog.Ctx(ctx)

		sv, err := slack.NewSecretsVerifier(r.Header, signingSecret)
		if err != nil {
			err = middleware.NewHTTPError(fmt.Errorf("invalid or missing signature: %w", err), r)
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusNotFound)
			return
		}

		r.Body = ioutil.NopCloser(io.TeeReader(r.Body, &sv))
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			err = middleware.NewHTTPError(fmt.Errorf("failed to read request body: %w", err), r)
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = sv.Ensure()
		if err != nil {
			err = middleware.NewHTTPError(fmt.Errorf("invalid signature: %w", err), r)
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusNotFound)
			return
		}

		r.Body = ioutil.NopCloser(bytes.NewBuffer(b))
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
