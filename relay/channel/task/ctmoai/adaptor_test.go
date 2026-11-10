package ctmoai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestBuildRequestBodyUsesTheDownstreamContract(t *testing.T) {
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
	// The upstream rejects images/videos/audios/size/second-field spellings with
	// 400 VIDEO_REQUEST_INVALID, so those must never reach it.
	assert.JSONEq(t, `{"model":"seedance-test","prompt":"waves","duration":10,"aspect_ratio":"16:9","guidances":{"image_reference":[{"image":{"url":"https://cdn.example/a.png","type":"UPLOADED"}}],"video_reference_base":[{"video":{"url":"https://cdn.example/a.mp4","type":"UPLOADED"}}],"audio_reference":[{"audio":{"url":"https://cdn.example/a.mp3","type":"UPLOADED"}}]}}`, string(data))
}

func TestParseUnknownRemainsQueued(t *testing.T) {
	result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{"id":"task_1","status":"unknown","progress":40}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusQueued, result.Status)
	assert.Equal(t, "40%", result.Progress)
}

func TestBuildRequestBodyDoesNotAllowMetadataToBypassValidatedFields(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","metadata":{"duration":3600,"image_url":"https://evil.example/x.png","start_frame_url":"https://evil.example/f.png","guidances":{},"images":["https://evil.example/image.png"],"safe_extension":"ok"}}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	data := readRequestBody(t, adaptor, ctx, info)
	assert.JSONEq(t, `{"model":"seedance-test","prompt":"waves","duration":10,"aspect_ratio":"16:9","safe_extension":"ok"}`, data)
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

// The two CTMOAI accounts label the same /v1/videos endpoint differently: the
// MiniMax H3 catalog reports "openai-video" while the Seedance catalog reports
// "openai". Accepting only one of them would lock out a whole account.
func TestValidateRequestAcceptsBothVideoEndpointLabels(t *testing.T) {
	for _, label := range []string{"openai-video", "openai", "OpenAI-Video"} {
		ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9"}`)
		info := ctmoaiInfo()
		info.VideoAccount.Models["seedance-test"] = relaycommon.VideoAccountModelMeta{
			ID: "seedance-test", Available: true, SupportedEndpointTypes: []string{label}, DurationsSeconds: []int{10}, Ratios: []string{"16:9"},
		}
		adaptor := &TaskAdaptor{}
		adaptor.Init(info)
		require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info), "endpoint label %q should be accepted", label)
		common.CleanupBodyStorage(ctx)
	}
}

func TestValidateRequestRejectsANonVideoEndpoint(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9"}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	info.VideoAccount.Models["seedance-test"] = relaycommon.VideoAccountModelMeta{
		ID: "seedance-test", Available: true, SupportedEndpointTypes: []string{"image-generation"}, DurationsSeconds: []int{10}, Ratios: []string{"16:9"},
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	taskErr := adaptor.ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "OpenAI video endpoint")
}

// CF super resolution models have no text-to-video workflow, and the flag now
// travels with the model instead of being re-derived from the model name.
func TestValidateRequestHonoursTheRequiresReferenceImageFlag(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9"}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	info.VideoAccount.Models["seedance-test"] = relaycommon.VideoAccountModelMeta{
		ID: "seedance-test", Available: true, DurationsSeconds: []int{10}, Ratios: []string{"16:9"}, MaxImages: 9, RequiresReferenceImage: true,
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	taskErr := adaptor.ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "at least one reference image")
}

// First/last frame is only reachable when the derived capability says so; the
// gateway must not reject a documented workflow_id=fl2v request.
func TestValidateRequestAllowsFirstLastFrameWhenDerivedCapabilityIsSet(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","workflow_id":"fl2v","images":["https://cdn.example/first.png","https://cdn.example/last.png"]}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	info.VideoAccount.Models["seedance-test"] = relaycommon.VideoAccountModelMeta{
		ID: "seedance-test", Available: true, DurationsSeconds: []int{10}, Ratios: []string{"16:9"}, MaxImages: 9, SupportsFirstLastFrame: true,
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))

	data := readRequestBody(t, adaptor, ctx, info)
	// Frames are positional and exclusive: no workflow_id, no image_url.
	assert.JSONEq(t, `{"model":"seedance-test","prompt":"waves","duration":10,"aspect_ratio":"16:9","start_frame_url":"https://cdn.example/first.png","end_frame_url":"https://cdn.example/last.png"}`, data)
}

func TestValidateRequestRejectsFirstLastFrameWithoutTheCapability(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","workflow_id":"fl2v","images":["https://cdn.example/first.png"]}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	taskErr := adaptor.ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "first/last frame")
}

// H3 expects reference_videos / reference_audios while Seedance expects
// videos / audios; the caller sends one canonical shape either way.
// size is not part of the upstream contract any more; the model id already pins
// the resolution.
func TestBuildRequestBodyDropsTheUnsupportedSizeField(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"minimax-h3-original-768p","prompt":"waves","seconds":10,"aspect_ratio":"16:9","size":"1376x768","images":["https://cdn.example/a.png"]}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	info.VideoAccount.Models["minimax-h3-original-768p"] = relaycommon.VideoAccountModelMeta{
		ID: "minimax-h3-original-768p", Group: "minimax-h3", Available: true, DurationsSeconds: []int{10},
		Ratios: []string{"16:9"}, Sizes: []string{"1376x768"}, MaxImages: 9,
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	data := readRequestBody(t, adaptor, ctx, info)
	assert.JSONEq(t, `{"model":"minimax-h3-original-768p","prompt":"waves","duration":10,"aspect_ratio":"16:9","image_url":"https://cdn.example/a.png"}`, data)
}

// CTMOAI's console keeps a lone reference image on the single-image field, which
// selects the single-reference workflow; a non-empty images array selects the
// multi-reference one. Rewriting the caller's choice would silently change which
// workflow runs.
func TestBuildRequestBodyUsesImageURLForASingleReference(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","input_reference":"https://cdn.example/only.png"}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	data := readRequestBody(t, adaptor, ctx, info)
	assert.JSONEq(t, `{"model":"seedance-test","prompt":"waves","duration":10,"aspect_ratio":"16:9","image_url":"https://cdn.example/only.png"}`, data)
}

func TestBuildRequestBodyKeepsMultipleImagesInGuidances(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","images":["https://cdn.example/one.png","https://cdn.example/two.png"]}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	data := readRequestBody(t, adaptor, ctx, info)
	assert.JSONEq(t, `{"model":"seedance-test","prompt":"waves","duration":10,"aspect_ratio":"16:9","guidances":{"image_reference":[{"image":{"url":"https://cdn.example/one.png","type":"UPLOADED"}},{"image":{"url":"https://cdn.example/two.png","type":"UPLOADED"}}]}}`, data)
}

// The frame workflow carries its frames as the documented array even when only
// one frame is supplied.
func TestBuildRequestBodyUsesOnlyStartFrameForASingleFrame(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","workflow_id":"fl2v","input_reference":"https://cdn.example/first.png"}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	info.VideoAccount.Models["seedance-test"] = relaycommon.VideoAccountModelMeta{
		ID: "seedance-test", Available: true, DurationsSeconds: []int{10}, Ratios: []string{"16:9"}, MaxImages: 9, SupportsFirstLastFrame: true,
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	data := readRequestBody(t, adaptor, ctx, info)
	assert.JSONEq(t, `{"model":"seedance-test","prompt":"waves","duration":10,"aspect_ratio":"16:9","start_frame_url":"https://cdn.example/first.png"}`, data)
}

func TestBuildRequestBodyForwardsPromptEnhanceAsAString(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","prompt_enhance":true}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	data := readRequestBody(t, adaptor, ctx, info)
	assert.JSONEq(t, `{"model":"seedance-test","prompt":"waves","duration":10,"aspect_ratio":"16:9","prompt_enhance":"true"}`, data)
}

// An explicit false must survive, and metadata must not be a second way to set it.
func TestBuildRequestBodyKeepsAnExplicitPromptEnhanceFalse(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","prompt":"waves","seconds":10,"aspect_ratio":"16:9","prompt_enhance":false,"metadata":{"prompt_enhance":true}}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(ctx, info))
	data := readRequestBody(t, adaptor, ctx, info)
	assert.JSONEq(t, `{"model":"seedance-test","prompt":"waves","duration":10,"aspect_ratio":"16:9","prompt_enhance":"false"}`, data)
}

func readRequestBody(t *testing.T, adaptor *TaskAdaptor, ctx *gin.Context, info *relaycommon.RelayInfo) string {
	t.Helper()
	body, err := adaptor.BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	return string(data)
}

func TestBuildRequestHeaderSendsTheIdempotencyKey(t *testing.T) {
	// Newer CTMOAI deployments answer 400 IDEMPOTENCY_KEY_INVALID without it.
	info := ctmoaiInfo()
	info.RequestId = "202610281-abcdef123456"
	req := httptest.NewRequest(http.MethodPost, "/v1/videos", nil)

	adaptor := &TaskAdaptor{apiKey: "secret"}
	require.NoError(t, adaptor.BuildRequestHeader(ctmoaiTaskContext(`{}`), req, info))

	assert.Equal(t, "Bearer secret", req.Header.Get("Authorization"))
	assert.Equal(t, info.RequestId, req.Header.Get("Idempotency-Key"))
}

func TestBuildRequestHeaderOmitsAnEmptyIdempotencyKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	adaptor := &TaskAdaptor{apiKey: "secret"}
	require.NoError(t, adaptor.BuildRequestHeader(ctmoaiTaskContext(`{}`), req, ctmoaiInfo()))
	assert.Empty(t, req.Header.Get("Idempotency-Key"))
}

func TestBuildRequestHeaderPrefersTheCallerIdempotencyKey(t *testing.T) {
	info := ctmoaiInfo()
	info.RequestId = "relay-request-id"
	req := httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	ctx := ctmoaiTaskContext(`{}`)
	ctx.Request.Header.Set("Idempotency-Key", "caller-key-1")

	adaptor := &TaskAdaptor{apiKey: "secret"}
	require.NoError(t, adaptor.BuildRequestHeader(ctx, req, info))
	assert.Equal(t, "caller-key-1", req.Header.Get("Idempotency-Key"))
}

func TestBuildRequestHeaderFallsBackWhenTheCallerKeyIsUnusable(t *testing.T) {
	info := ctmoaiInfo()
	info.RequestId = "relay-request-id"
	// Space is printable ASCII and therefore legal upstream; only length or
	// non-ASCII input makes a caller key unusable.
	for _, bad := range []string{strings.Repeat("a", 129), "中文", "tab\there"} {
		req := httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
		ctx := ctmoaiTaskContext(`{}`)
		ctx.Request.Header.Set("Idempotency-Key", bad)
		adaptor := &TaskAdaptor{apiKey: "secret"}
		require.NoError(t, adaptor.BuildRequestHeader(ctx, req, info))
		assert.Equal(t, "relay-request-id", req.Header.Get("Idempotency-Key"), "key %q must be ignored", bad)
	}
}

func TestValidateRequestRequiresAPromptEvenWithAnImage(t *testing.T) {
	ctx := ctmoaiTaskContext(`{"model":"seedance-test","seconds":10,"aspect_ratio":"16:9","input_reference":"https://cdn.example/only.png"}`)
	defer common.CleanupBodyStorage(ctx)
	info := ctmoaiInfo()
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	taskErr := adaptor.ValidateRequestAndSetAction(ctx, info)
	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "prompt is required")
}
