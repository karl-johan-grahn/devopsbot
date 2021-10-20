package bot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateIncidentChannelName(t *testing.T) {
	incidentChannelName := ""
	err := validateIncidentChannelName("empty_string", incidentChannelName)
	assert.Error(t, err)

	incidentChannelName = "UPPERCASE_INVALID"
	err = validateIncidentChannelName("uppercase_invalid", incidentChannelName)
	assert.Error(t, err)

	incidentChannelName = "?/*"
	err = validateIncidentChannelName("special_chars_invalid", incidentChannelName)
	assert.Error(t, err)
}
