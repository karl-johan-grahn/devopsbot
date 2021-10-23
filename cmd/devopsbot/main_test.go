package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitFlags(t *testing.T) {
	cmd := newCmd()
	// Before initialization these flags should not have been set
	flags := []string{"addr", "tls.addr", "tls.cert", "tls.key", "slack.accessToken",
		"slack.signingSecret", "slack.adminGroupID", "slack.broadcastChannelID"}
	for _, f := range flags {
		_, err := cmd.Flags().GetString(f)
		assert.Error(t, err)
	}
	initFlags(cmd)
	// After initialization the flags should have been set
	for _, f := range flags {
		_, err := cmd.Flags().GetString(f)
		assert.NoError(t, err)
	}
}
