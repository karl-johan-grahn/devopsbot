package bot

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMWVerify(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	secret := "signingsecret123"
	h := mwVerify(secret, next)
	const exampleURL = "http://example.com"

	// missing signature
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, exampleURL, nil)
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)

	// valid signature
	w = httptest.NewRecorder()
	body := "test message"
	r = httptest.NewRequest(http.MethodPost, exampleURL, bytes.NewBufferString(body))
	hash := hmac.New(sha256.New, []byte(secret))
	ts := fmt.Sprintf("%d", time.Now().Unix())
	hash.Write([]byte(fmt.Sprintf("v0:%s:%s", ts, body)))

	r.Header.Set("X-Slack-Request-Timestamp", ts)
	r.Header.Set("X-Slack-Signature", "v0="+hex.EncodeToString(hash.Sum(nil)))
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	// invalid signature
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, exampleURL, bytes.NewBufferString(body))
	hash = hmac.New(sha256.New, []byte(secret))
	hash.Write([]byte(fmt.Sprintf("v0:%s:%s", fmt.Sprintf("%d", time.Now().Unix()), body)))
	// add a second to the timestamp to invalidate it
	r.Header.Set("X-Slack-Request-Timestamp", fmt.Sprintf("%d", time.Now().Add(1*time.Second).Unix()))
	r.Header.Set("X-Slack-Signature", "v0="+hex.EncodeToString(hash.Sum(nil)))
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
