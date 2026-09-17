package ctmoai

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relaydto "github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	channelName = "CTMOAI Video"
	videoPath   = "/v1/videos"
	maxRefs     = 50
)

type responseError struct {
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

type responseTask struct {
	ID          string         `json:"id"`
	TaskID      string         `json:"task_id"`
	Object      string         `json:"object"`
	Model       string         `json:"model"`
	Status      string         `json:"status"`
	Progress    int            `json:"progress"`
	CreatedAt   int64          `json:"created_at"`
	CompletedAt int64          `json:"completed_at"`
	ExpiresAt   int64          `json:"expires_at"`
	Seconds     any            `json:"seconds,omitempty"`
	Size        string         `json:"size,omitempty"`
	Error       *responseError `json:"error,omitempty"`
}

type TaskAdaptor struct {
	taskcommon.BaseBilling
	baseURL string
	apiKey  string
	account *relaycommon.VideoAccountMeta
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	if info == nil {
		return
	}
	a.baseURL = strings.TrimRight(info.ChannelBaseUrl, "/")
	a.apiKey = info.ApiKey
	a.account = info.VideoAccount
	if a.account != nil {
		if a.account.BaseURL != "" {
			a.baseURL = strings.TrimRight(a.account.BaseURL, "/")
		}
		a.apiKey = a.account.APIKey
	}
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *taskdto.TaskError {
	if taskErr := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionTextGenerate); taskErr != nil {
		return taskErr
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	modelName := strings.TrimSpace(req.Model)
	if modelName == "" {
		modelName = strings.TrimSpace(info.OriginModelName)
	}
	meta, ok := a.modelMeta(modelName)
	if !ok {
		return service.TaskErrorWrapperLocal(fmt.Errorf("video model %q is not available for this account", modelName), "invalid_model", http.StatusBadRequest)
	}
	if !meta.Available {
		return service.TaskErrorWrapperLocal(fmt.Errorf("video model %q is currently unavailable", modelName), "invalid_model", http.StatusBadRequest)
	}
	if len(meta.SupportedEndpointTypes) > 0 && !videoEndpointSupported(meta.SupportedEndpointTypes) {
		return service.TaskErrorWrapperLocal(fmt.Errorf("video model %q is not available on the OpenAI video endpoint", modelName), "invalid_model", http.StatusBadRequest)
	}
	seconds, err := resolveSeconds(req)
	if err != nil || seconds <= 0 || seconds > relaycommon.MaxTaskDurationSeconds {
		return service.TaskErrorWrapperLocal(fmt.Errorf("seconds must be a valid integer between 1 and %d", relaycommon.MaxTaskDurationSeconds), "invalid_seconds", http.StatusBadRequest)
	}
	if len(meta.DurationsSeconds) > 0 && !containsInt(meta.DurationsSeconds, seconds) {
		return service.TaskErrorWrapperLocal(fmt.Errorf("seconds %d is not supported by %s", seconds, modelName), "invalid_seconds", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.AspectRatio) == "" && len(meta.Ratios) > 0 {
		return service.TaskErrorWrapperLocal(errors.New("aspect_ratio is required for this model"), "invalid_aspect_ratio", http.StatusBadRequest)
	}
	if req.AspectRatio != "" && len(meta.Ratios) > 0 && !containsFold(meta.Ratios, req.AspectRatio) {
		return service.TaskErrorWrapperLocal(fmt.Errorf("aspect_ratio %s is not supported by %s", req.AspectRatio, modelName), "invalid_aspect_ratio", http.StatusBadRequest)
	}
	if req.Size != "" && len(meta.Sizes) > 0 && !containsFold(meta.Sizes, req.Size) {
		return service.TaskErrorWrapperLocal(fmt.Errorf("size %s is not supported by %s", req.Size, modelName), "invalid_size", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Prompt) == "" && len(req.Images) == 0 && strings.TrimSpace(req.InputReference) == "" {
		return service.TaskErrorWrapperLocal(errors.New("prompt is required when no reference image is supplied"), "invalid_request", http.StatusBadRequest)
	}
	if req.InputReference != "" && len(req.Images) == 0 {
		req.Images = []string{strings.TrimSpace(req.InputReference)}
	}
	if req.ReferenceVideo != "" && len(req.ReferenceVideos) == 0 {
		req.ReferenceVideos = []string{strings.TrimSpace(req.ReferenceVideo)}
	}
	if req.ReferenceAudio != "" && len(req.ReferenceAudios) == 0 {
		req.ReferenceAudios = []string{strings.TrimSpace(req.ReferenceAudio)}
	}
	if len(req.Images) > maxRefs || len(req.ReferenceVideos) > maxRefs || len(req.ReferenceAudios) > maxRefs {
		return service.TaskErrorWrapperLocal(fmt.Errorf("reference media count must not exceed %d", maxRefs), "invalid_reference", http.StatusBadRequest)
	}
	if meta.MaxImages >= 0 && len(req.Images) > meta.MaxImages {
		return service.TaskErrorWrapperLocal(fmt.Errorf("the model accepts at most %d images", meta.MaxImages), "invalid_reference", http.StatusBadRequest)
	}
	if meta.MaxVideos >= 0 && len(req.ReferenceVideos) > meta.MaxVideos {
		return service.TaskErrorWrapperLocal(fmt.Errorf("the model accepts at most %d videos", meta.MaxVideos), "invalid_reference", http.StatusBadRequest)
	}
	if meta.MaxAudios >= 0 && len(req.ReferenceAudios) > meta.MaxAudios {
		return service.TaskErrorWrapperLocal(fmt.Errorf("the model accepts at most %d audios", meta.MaxAudios), "invalid_reference", http.StatusBadRequest)
	}
	if (len(req.ReferenceVideos) > 0 || len(req.ReferenceAudios) > 0) && len(req.Images) == 0 {
		return service.TaskErrorWrapperLocal(errors.New("reference videos or audios require at least one reference image"), "missing_reference", http.StatusBadRequest)
	}
	if meta.RequiresReferenceImage && len(req.Images) == 0 {
		return service.TaskErrorWrapperLocal(errors.New("this model requires at least one reference image"), "missing_reference", http.StatusBadRequest)
	}
	if req.Mode == "first_last_frame" || strings.EqualFold(strings.TrimSpace(req.WorkflowID), "fl2v") {
		if !meta.SupportsFirstLastFrame {
			return service.TaskErrorWrapperLocal(errors.New("first/last frame mode is not supported by this model"), "invalid_mode", http.StatusBadRequest)
		}
		if len(req.Images) < 1 || len(req.Images) > 2 {
			return service.TaskErrorWrapperLocal(errors.New("first/last frame mode accepts one or two images"), "invalid_reference", http.StatusBadRequest)
		}
	}
	for _, item := range append(append([]string{}, req.Images...), append(req.ReferenceVideos, req.ReferenceAudios...)...) {
		if err := validatePublicMediaURL(item); err != nil {
			return service.TaskErrorWrapperLocal(err, "invalid_reference", http.StatusBadRequest)
		}
	}
	if info.OriginModelName == "" {
		info.OriginModelName = modelName
	}
	req.Model = modelName
	req.Duration = seconds
	req.Seconds = strconv.Itoa(seconds)
	c.Set("task_request", req)
	return nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	meta, ok := a.modelMeta(info.OriginModelName)
	if !ok || strings.ToLower(meta.PricingMode) != "per_second" {
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	seconds, err := resolveSeconds(req)
	if err != nil || seconds <= 0 {
		return nil
	}
	return map[string]float64{"seconds": float64(seconds)}
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	if a.baseURL == "" {
		return "", errors.New("CTMOAI video account base URL is empty")
	}
	return a.baseURL + videoPath, nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	if a.apiKey == "" {
		return errors.New("CTMOAI video account API key is empty")
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	seconds, err := resolveSeconds(req)
	if err != nil {
		return nil, errors.Wrap(err, "parse seconds")
	}
	modelName := info.UpstreamModelName
	if modelName == "" {
		modelName = req.Model
	}
	payload := map[string]any{
		"model":   modelName,
		"prompt":  req.Prompt,
		"seconds": seconds,
	}
	if req.AspectRatio != "" {
		payload["aspect_ratio"] = req.AspectRatio
	}
	if req.Size != "" {
		payload["size"] = req.Size
	}
	if len(req.Images) > 0 {
		payload["images"] = req.Images
	}
	if req.Mode != "" {
		payload["mode"] = req.Mode
	}
	if req.WorkflowID != "" {
		payload["workflow_id"] = req.WorkflowID
	}
	isH3 := model.IsMiniMaxH3VideoModel(a.modelGroup(modelName), modelName)
	if isH3 {
		if len(req.ReferenceVideos) > 0 {
			payload["reference_videos"] = req.ReferenceVideos
		}
		if len(req.ReferenceAudios) > 0 {
			payload["reference_audios"] = req.ReferenceAudios
		}
	} else {
		if len(req.ReferenceVideos) > 0 {
			payload["videos"] = req.ReferenceVideos
		}
		if len(req.ReferenceAudios) > 0 {
			payload["audios"] = req.ReferenceAudios
		}
	}
	if req.Metadata != nil {
		for key, value := range req.Metadata {
			// Metadata is an extension escape hatch, not a way to bypass the
			// validated request fields. In particular, allowing metadata to add
			// images/reference media or duration here would bypass the catalog
			// max_* and billing bounds checked above.
			switch key {
			case "model", "seconds", "prompt", "duration", "images", "videos", "audios",
				"input_reference", "reference_video", "reference_videos", "reference_audio", "reference_audios",
				"aspect_ratio", "size", "mode", "workflow_id":
				continue
			}
			if _, exists := payload[key]; !exists {
				payload[key] = value
			}
		}
	}
	body, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(body), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *taskdto.TaskError) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusBadGateway)
	}
	_ = resp.Body.Close()
	var task responseTask
	if err := common.Unmarshal(body, &task); err != nil {
		return "", body, service.TaskErrorWrapper(errors.Wrap(err, "decode CTMOAI video response"), "invalid_response", http.StatusBadGateway)
	}
	upstreamID := strings.TrimSpace(task.ID)
	if upstreamID == "" {
		upstreamID = strings.TrimSpace(task.TaskID)
	}
	if upstreamID == "" {
		return "", body, service.TaskErrorWrapper(errors.New("task_id is empty"), "invalid_response", http.StatusBadGateway)
	}
	video := a.toOpenAIVideo(task, info.PublicTaskID, info.OriginModelName)
	c.JSON(http.StatusOK, video)
	return upstreamID, body, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	if a.account == nil {
		return nil
	}
	result := make([]string, 0, len(a.account.Models))
	for name := range a.account.Models {
		result = append(result, name)
	}
	return result
}

func (a *TaskAdaptor) GetChannelName() string { return channelName }

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, errors.New("invalid task_id")
	}
	requestURL := strings.TrimRight(baseURL, "/") + videoPath + "/" + url.PathEscape(taskID)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, errors.Wrap(err, "new proxy http client")
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(body []byte) (*relaycommon.TaskInfo, error) {
	var response responseTask
	if err := common.Unmarshal(body, &response); err != nil {
		return nil, errors.Wrap(err, "unmarshal CTMOAI task result")
	}
	result := &relaycommon.TaskInfo{Code: 0}
	switch strings.ToLower(strings.TrimSpace(response.Status)) {
	case "queued", "pending", "unknown", "":
		result.Status = model.TaskStatusQueued
	case "in_progress", "processing":
		result.Status = model.TaskStatusInProgress
	case "completed", "success":
		result.Status = model.TaskStatusSuccess
	case "failed", "cancelled":
		result.Status = model.TaskStatusFailure
		if response.Error != nil {
			result.Reason = response.Error.Message
		}
		if result.Reason == "" {
			result.Reason = "task failed"
		}
	default:
		result.Status = model.TaskStatusInProgress
	}
	if response.Progress >= 0 && response.Progress <= 100 {
		result.Progress = fmt.Sprintf("%d%%", response.Progress)
	}
	return result, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var response responseTask
	if len(task.Data) > 0 {
		if err := common.Unmarshal(task.Data, &response); err != nil {
			return nil, errors.Wrap(err, "unmarshal CTMOAI task data")
		}
	}
	modelName := task.Properties.OriginModelName
	if modelName == "" {
		modelName = response.Model
	}
	video := a.toOpenAIVideo(response, task.TaskID, modelName)
	video.Status = task.Status.ToVideoStatus()
	video.SetProgressStr(task.Progress)
	video.CreatedAt = task.CreatedAt
	video.CompletedAt = task.FinishTime
	video.SetMetadata("url", task.GetResultURL())
	return common.Marshal(video)
}

func (a *TaskAdaptor) modelMeta(name string) (relaycommon.VideoAccountModelMeta, bool) {
	if a.account == nil {
		return relaycommon.VideoAccountModelMeta{}, false
	}
	for key, value := range a.account.Models {
		if strings.EqualFold(key, name) {
			return value, true
		}
	}
	return relaycommon.VideoAccountModelMeta{}, false
}

func (a *TaskAdaptor) modelGroup(name string) string {
	meta, ok := a.modelMeta(name)
	if !ok {
		return ""
	}
	return meta.Group
}

func (a *TaskAdaptor) toOpenAIVideo(response responseTask, publicID, modelName string) *relaydto.OpenAIVideo {
	video := relaydto.NewOpenAIVideo()
	video.ID = publicID
	video.TaskID = publicID
	video.Object = "video"
	video.Model = modelName
	video.Status = mapStatus(response.Status)
	video.Progress = response.Progress
	video.CreatedAt = response.CreatedAt
	video.CompletedAt = response.CompletedAt
	video.ExpiresAt = response.ExpiresAt
	video.Seconds = stringifySeconds(response.Seconds)
	video.Size = response.Size
	if response.Error != nil {
		video.Error = &relaydto.OpenAIVideoError{Message: response.Error.Message, Code: response.Error.Code}
	}
	if video.CreatedAt == 0 {
		video.CreatedAt = time.Now().Unix()
	}
	return video
}

func mapStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued", "pending", "unknown", "":
		return relaydto.VideoStatusQueued
	case "in_progress", "processing":
		return relaydto.VideoStatusInProgress
	case "completed", "success":
		return relaydto.VideoStatusCompleted
	case "failed", "cancelled":
		return relaydto.VideoStatusFailed
	default:
		return relaydto.VideoStatusUnknown
	}
}

func resolveSeconds(req relaycommon.TaskSubmitReq) (int, error) {
	if req.Duration > 0 {
		return req.Duration, nil
	}
	if strings.TrimSpace(req.Seconds) == "" {
		return 0, errors.New("seconds is required")
	}
	return strconv.Atoi(strings.TrimSpace(req.Seconds))
}

func validatePublicMediaURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return errors.New("reference media URL is empty")
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("reference media must be a public http(s) URL")
	}
	return nil
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

// videoEndpointSupported reports whether a model advertises the OpenAI video
// endpoint. CTMOAI labels the same /v1/videos endpoint differently per account:
// the MiniMax H3 catalog reports "openai-video" while the Seedance catalog
// reports "openai". Both are video models on a dedicated video account, so
// accepting only one label would lock out an entire account.
func videoEndpointSupported(values []string) bool {
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "openai-video", "openai":
			return true
		}
	}
	return false
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringifySeconds(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.Itoa(int(typed))
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return ""
	}
}

var _ channel.TaskAdaptor = (*TaskAdaptor)(nil)

// Keep the platform constant referenced in this package so accidental removal
// of the dedicated routing registration is caught by the compiler/tests.
var _ = constant.TaskPlatformVideoCTMoai
