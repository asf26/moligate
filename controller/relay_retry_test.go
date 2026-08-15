package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldRetryBadRequestTransientUpstreamError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "model capacity is temporarily unavailable, please try again",
		Type:    "server_error",
		Code:    "server_error",
	}, http.StatusBadRequest)

	require.True(t, shouldRetry(ctx, err, 1))
}

func TestShouldRetryBadRequestClientError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "Invalid value for parameter size",
		Type:    "invalid_request_error",
		Code:    "invalid_request_error",
		Param:   "size",
	}, http.StatusBadRequest)

	require.False(t, shouldRetry(ctx, err, 1))
}

func TestShouldRetryBadRequestRespectsRetryGuards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "算力不足，请稍后再试",
		Type:    "upstream_error",
		Code:    "upstream_error",
	}, http.StatusBadRequest)

	require.False(t, shouldRetry(ctx, err, 0))

	ctx.Set("specific_channel_id", true)
	require.False(t, shouldRetry(ctx, err, 1))
}
