/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package minimax_h3

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
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

type requestPayload struct {
	Model           string                 `json:"model"`
	Prompt          string                 `json:"prompt"`
	Seconds         int                    `json:"seconds"`
	AspectRatio     string                 `json:"aspect_ratio,omitempty"`
	Size            string                 `json:"size,omitempty"`
	Images          []string               `json:"images,omitempty"`
	ReferenceVideos []string               `json:"reference_videos,omitempty"`
	ReferenceAudios []string               `json:"reference_audios,omitempty"`
	ReferenceVideo  string                 `json:"reference_video,omitempty"`
	ReferenceAudio  string                 `json:"reference_audio,omitempty"`
	WorkflowID      string                 `json:"workflow_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

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
	Seconds     interface{}    `json:"seconds,omitempty"`
	Size        string         `json:"size,omitempty"`
	Error       *responseError `json:"error,omitempty"`
}

type TaskAdaptor struct {
	taskcommon.BaseBilling
	baseURL string
	apiKey  string
}

const (
	maxReferenceImageCount = 16
	maxReferenceImageBytes = 20 << 20
)

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	if info == nil {
		return
	}
	a.baseURL = strings.TrimRight(info.ChannelBaseUrl, "/")
	a.apiKey = info.ApiKey
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
	if !IsModel(modelName) {
		return service.TaskErrorWrapperLocal(fmt.Errorf("unsupported MiniMax H3 model: %s", modelName), "invalid_model", http.StatusBadRequest)
	}

	seconds, err := resolveSeconds(req)
	if err != nil || seconds < 4 || seconds > 15 {
		return service.TaskErrorWrapperLocal(fmt.Errorf("seconds must be an integer between 4 and 15"), "invalid_seconds", http.StatusBadRequest)
	}
	req.Seconds = strconv.Itoa(seconds)
	req.Duration = seconds

	if req.AspectRatio == "" {
		req.AspectRatio = "16:9"
	}
	if !IsAspectRatio(req.AspectRatio) {
		return service.TaskErrorWrapperLocal(fmt.Errorf("unsupported aspect_ratio: %s", req.AspectRatio), "invalid_aspect_ratio", http.StatusBadRequest)
	}
	if req.InputReference != "" && len(req.Images) == 0 {
		req.Images = []string{strings.TrimSpace(req.InputReference)}
	}
	uploadedImages, err := extractMultipartReferenceImages(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_reference", http.StatusBadRequest)
	}
	req.Images = append(req.Images, uploadedImages...)
	if len(req.Images) > maxReferenceImageCount || len(req.ReferenceVideos) > maxReferenceImageCount || len(req.ReferenceAudios) > maxReferenceImageCount {
		return service.TaskErrorWrapperLocal(fmt.Errorf("reference media count must not exceed %d", maxReferenceImageCount), "invalid_reference", http.StatusBadRequest)
	}
	if strings.Contains(modelName, "-cf-") && len(req.Images) == 0 {
		return service.TaskErrorWrapperLocal(fmt.Errorf("this model requires at least one reference image"), "missing_reference", http.StatusBadRequest)
	}
	if req.WorkflowID == "fl2v" && (len(req.Images) < 1 || len(req.Images) > 2) {
		return service.TaskErrorWrapperLocal(fmt.Errorf("fl2v workflow requires one or two images"), "invalid_reference", http.StatusBadRequest)
	}
	if req.Size != "" && !validSizeForModel(modelName, req.AspectRatio, req.Size) {
		return service.TaskErrorWrapperLocal(fmt.Errorf("size is not supported by %s", modelName), "invalid_size", http.StatusBadRequest)
	}

	c.Set("task_request", req)
	return nil
}

func extractMultipartReferenceImages(c *gin.Context) ([]string, error) {
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		return nil, nil
	}

	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return nil, errors.Wrap(err, "parse reference image form")
	}
	defer form.RemoveAll()

	images := make([]string, 0)
	for _, field := range []string{"input_reference", "input_reference[]", "images", "images[]"} {
		for _, fileHeader := range form.File[field] {
			if len(images) >= maxReferenceImageCount {
				return nil, fmt.Errorf("reference media count must not exceed %d", maxReferenceImageCount)
			}
			if fileHeader.Size > maxReferenceImageBytes {
				return nil, fmt.Errorf("reference image %q exceeds %d MB", fileHeader.Filename, maxReferenceImageBytes>>20)
			}

			file, err := fileHeader.Open()
			if err != nil {
				return nil, errors.Wrap(err, "open reference image")
			}
			data, readErr := io.ReadAll(io.LimitReader(file, maxReferenceImageBytes+1))
			closeErr := file.Close()
			if readErr != nil {
				return nil, errors.Wrap(readErr, "read reference image")
			}
			if closeErr != nil {
				return nil, errors.Wrap(closeErr, "close reference image")
			}
			if len(data) > maxReferenceImageBytes {
				return nil, fmt.Errorf("reference image %q exceeds %d MB", fileHeader.Filename, maxReferenceImageBytes>>20)
			}

			mediaType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
			if parsedMediaType, _, parseErr := mime.ParseMediaType(mediaType); parseErr == nil {
				mediaType = parsedMediaType
			}
			if mediaType == "" || mediaType == "application/octet-stream" {
				mediaType = http.DetectContentType(data)
			}
			if !strings.HasPrefix(mediaType, "image/") {
				return nil, fmt.Errorf("reference file %q must be an image", fileHeader.Filename)
			}

			images = append(images, "data:"+mediaType+";base64,"+base64.StdEncoding.EncodeToString(data))
		}
	}
	return images, nil
}

func resolveSeconds(req relaycommon.TaskSubmitReq) (int, error) {
	if req.Duration > 0 {
		return req.Duration, nil
	}
	if strings.TrimSpace(req.Seconds) == "" {
		return 4, nil
	}
	seconds, err := strconv.Atoi(strings.TrimSpace(req.Seconds))
	if err != nil {
		return 0, err
	}
	return seconds, nil
}

func validSizeForModel(modelName, aspectRatio, size string) bool {
	if strings.Contains(modelName, "-cf-2k") {
		return size == "2K"
	}
	if strings.Contains(modelName, "-cf-4k") {
		return size == "4K"
	}
	return size == pixelSize(aspectRatio, strings.Contains(modelName, "-1080p"))
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	seconds, err := resolveSeconds(req)
	if err != nil || seconds < 4 {
		seconds = 4
	}
	if seconds > 15 {
		seconds = 15
	}
	return map[string]float64{"seconds": float64(seconds)}
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	if a.baseURL == "" {
		return "", errors.New("MiniMax H3 channel base URL is empty")
	}
	return a.baseURL + VideoEndpoint, nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
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
	images := append([]string(nil), req.Images...)
	if len(images) == 0 && req.InputReference != "" {
		images = []string{strings.TrimSpace(req.InputReference)}
	}
	size := req.Size
	if size == "" {
		size = defaultSize(modelName, req.AspectRatio)
	}
	payload := requestPayload{
		Model:           modelName,
		Prompt:          req.Prompt,
		Seconds:         seconds,
		AspectRatio:     req.AspectRatio,
		Size:            size,
		Images:          images,
		ReferenceVideos: req.ReferenceVideos,
		ReferenceAudios: req.ReferenceAudios,
		ReferenceVideo:  req.ReferenceVideo,
		ReferenceAudio:  req.ReferenceAudio,
		WorkflowID:      req.WorkflowID,
		Metadata:        req.Metadata,
	}
	body, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(body), nil
}

func defaultSize(modelName, ratio string) string {
	if strings.Contains(modelName, "-cf-2k") {
		return "2K"
	}
	if strings.Contains(modelName, "-cf-4k") {
		return "4K"
	}
	if ratio == "" {
		ratio = "16:9"
	}
	if strings.Contains(modelName, "-1080p") {
		return pixelSize(ratio, true)
	}
	return pixelSize(ratio, false)
}

func pixelSize(ratio string, high bool) string {
	sizes := map[string][2]string{
		"16:9": {"1376x768", "1920x1088"},
		"9:16": {"768x1376", "1088x1920"},
		"1:1":  {"1024x1024", "1440x1440"},
		"2:3":  {"832x1248", "1184x1760"},
		"3:2":  {"1248x832", "1760x1184"},
		"3:4":  {"896x1184", "1248x1664"},
		"4:3":  {"1184x896", "1664x1248"},
		"21:9": {"1568x672", "2208x960"},
	}
	size, ok := sizes[ratio]
	if !ok {
		size = sizes["16:9"]
	}
	if high {
		return size[1]
	}
	return size[0]
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *taskdto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()
	var upstream responseTask
	if err := common.Unmarshal(responseBody, &upstream); err != nil {
		return "", responseBody, service.TaskErrorWrapper(errors.Wrap(err, "decode MiniMax H3 response"), "invalid_response", http.StatusBadGateway)
	}
	upstreamID := upstream.ID
	if upstreamID == "" {
		upstreamID = upstream.TaskID
	}
	if upstreamID == "" {
		return "", responseBody, service.TaskErrorWrapper(errors.New("task_id is empty"), "invalid_response", http.StatusBadGateway)
	}
	video := a.toOpenAIVideo(upstream, info.PublicTaskID, info.OriginModelName)
	c.JSON(http.StatusOK, video)
	return upstreamID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, errors.New("invalid task_id")
	}
	requestURL := strings.TrimRight(baseURL, "/") + VideoEndpoint + "/" + url.PathEscape(taskID)
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

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var response responseTask
	if err := common.Unmarshal(respBody, &response); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result")
	}
	result := &relaycommon.TaskInfo{Code: 0}
	switch strings.ToLower(response.Status) {
	case "queued", "pending":
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
	case "unknown", "":
		// The H3 API uses unknown for an intermediate state. It must remain
		// pollable and must never trigger an automatic refund.
		result.Status = model.TaskStatusInProgress
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
			return nil, errors.Wrap(err, "unmarshal MiniMax H3 task data")
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
	body, err := common.Marshal(video)
	if err != nil {
		return nil, err
	}
	return body, nil
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
	switch strings.ToLower(status) {
	case "queued", "pending":
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

func stringifySeconds(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.Itoa(int(v))
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return ""
	}
}
