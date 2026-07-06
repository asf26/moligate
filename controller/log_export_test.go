package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectedLogExportFieldsKeepsRequestedOrderAndDropsUnknownFields(t *testing.T) {
	fields := selectedLogExportFields("id,username,channel,token_name,ip,unknown", true)

	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		keys = append(keys, field.Key)
	}

	require.NotEmpty(t, keys)
	assert.Equal(t, []string{"id", "username", "channel", "token_name", "ip"}, keys)
}

func TestSelectedLogExportFieldsFallsBackToDefault(t *testing.T) {
	fields := selectedLogExportFields("unknown", true)

	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		keys = append(keys, field.Key)
	}

	require.NotEmpty(t, keys)
	assert.Contains(t, keys, "username")
	assert.Contains(t, keys, "channel")
	assert.Contains(t, keys, "channel_name")
}
