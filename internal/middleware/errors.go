package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/zerolog"
)

// HTTPError records an HTTP error and the request that caused it
type HTTPError struct {
	wrapped error
	r       *http.Request
	Code    int
}

// NewHTTPError - wrap an error as an HTTP error
func NewHTTPError(err error, r *http.Request, code ...int) *HTTPError {
	he := &HTTPError{err, r, http.StatusInternalServerError}
	if len(code) == 1 {
		he.Code = code[0]
	}
	return he
}

func (e *HTTPError) Error() string {
	return e.r.Method + " " + e.r.URL.String() + " - " + strconv.Itoa(e.Code) + ": " + e.wrapped.Error()
}

func (e *HTTPError) Unwrap() error { return e.wrapped }

// Request - return the request that caused this error
func (e *HTTPError) Request() *http.Request {
	return e.r
}

// Send - respond with this error, and return it for further use
func (e *HTTPError) Send(w http.ResponseWriter, r *http.Request) *HTTPError {
	if e.Code == 0 {
		e.Code = http.StatusInternalServerError
	}

	b := map[string][]errBody{
		"errors": {{Title: e.Error(), Code: "HTTP-" + strconv.Itoa(e.Code)}},
	}

	w.WriteHeader(e.Code)
	log := zerolog.Ctx(r.Context())
	if err := json.NewEncoder(w).Encode(b); err != nil {
		log.Error().Err(err).Send()
	}

	return e
}

type errBody struct {
	Code   string                 `json:"code,omitempty"`
	Title  string                 `json:"title,omitempty"`
	Detail string                 `json:"detail,omitempty"`
	Meta   map[string]interface{} `json:"meta,omitempty"`
}
