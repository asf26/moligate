package ctmoai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ctmoaiTaskContext(body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	return ctx
}

func ctmoaiInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{ChannelBaseUrl: model.VideoAccountBaseURL, ApiKey: "secret"},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
		VideoAccount: &relaycommon.VideoAccountMeta{
			BaseURL: model.VideoAccountBaseURL,
			APIKey:  "secret",
			Models: map[string]relaycommon.VideoAccountModelMeta{
				"seedance-test": {
					ID: "seedance-test", Group: "stable", Available: true, DurationsSeconds: []int{4, 10}, Ratios: []string{"16:9"},
					MaxImages: 2, MaxVideos: 1, MaxAudios: 1, PricingMode: "per_second", PricingAmount: 0.5,
				},
			},
		},
	}
}

func TestBuildRequestBodyUsesStableArraysAndCatalogFields(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","images":["https://cdn.example/a.png"],"reference_videos":["https://cdn.example/a.mp4"],"reference_audios":["https://cdn.example/a.mp3"]}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","images":["https://cdn.example/a.png"],"videos":["https://cdn.example/a.mp4"],"audios":["https://cdn.example/a.mp3"]}`, string(data))
}

func TestParseUnknownRemainsQueued(t *testing.T) {
	result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{"id":"task_1","status":"unknown","progress":40}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusQueued, result.Status)
	assert.Equal(t, "40%", result.Progress)
}

func TestBuildRequestBodyDoesNotAllowMetadataToBypassValidatedFields(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","metadata":{"images":["https://evil.example/image.png"],"seconds":3600,"aspect_ratio":"1:1","safe_extension":"ok"}}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","safe_extension":"ok"}`, string(data))
}

func TestValidateRequestCountsSingularReferenceMediaAgainstCatalogLimits(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"reference_video":"https://cdn.example/a.mp4"}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	info.VideoAccount.Models["seedance-test"] = relaycommon.VideoAccountModelMeta{
		ID: "seedance-test", Available: true, DurationsSeconds: []int{10}, MaxVideos: 0,
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	taskErr := adaptor.ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "at most 0 videos")
}

func TestValidateRequestRequiresImageWhenVideoOrAudioReferencesArePresent(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","reference_videos":["https://cdn.example/a.mp4"]}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	taskErr := adaptor.ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "require at least one reference image")
}

func TestValidateRequestRequiresCatalogDurationAndSizeFields(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","aspect_ratio":"16:9","size":"1920x1080"}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	info.VideoAccount.Models["seedance-test"] = relaycommon.VideoAccountModelMeta{
		ID: "seedance-test", Available: true, DurationsSeconds: []int{10}, Ratios: []string{"16:9"}, Sizes: []string{"1280x720"},
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	taskErr := adaptor.ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "seconds")

	ctx = ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","size":"1920x1080"}`)
	defer common.CleanupBodyStorage(ctx)
	taskErr = adaptor.ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "size")
}
