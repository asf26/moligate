package controller

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

const (
	videoImageUploadLimit = 10 << 20
	videoVideoUploadLimit = 50 << 20
	videoAudioUploadLimit = 30 << 20
)

// UploadVideoAccountMedia follows CTMOAI's /api/sd-media/upload contract. The
// browser sends an opaque account selector; the upstream credential is read
// only from the server-side VideoAccount row.
func UploadVideoAccountMedia(c *gin.Context) {
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", "invalid multipart request")
		return
	}
	defer form.RemoveAll()

	mediaType := strings.ToLower(strings.TrimSpace(firstFormValue(form, "type")))
	if mediaType != "images" && mediaType != "videos" && mediaType != "audios" {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", "type must be images, videos, or audios")
		return
	}
	files := form.File["file"]
	if len(files) != 1 {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", "exactly one file is required")
		return
	}
	fileHeader := files[0]
	limit := videoImageUploadLimit
	if mediaType == "videos" {
		limit = videoVideoUploadLimit
	} else if mediaType == "audios" {
		limit = videoAudioUploadLimit
	}
	if fileHeader.Size <= 0 || fileHeader.Size > int64(limit) {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", fmt.Sprintf("file exceeds the %s upload limit", mediaType))
		return
	}

	account, err := resolveVideoUploadAccount(c, form)
	if err != nil {
		videoProxyError(c, http.StatusServiceUnavailable, "video_account_unavailable", err.Error())
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", "failed to open upload")
		return
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("type", mediaType); err != nil {
		videoProxyError(c, http.StatusInternalServerError, "server_error", "failed to prepare upload")
		return
	}
	part, err := writer.CreateFormFile("file", fileHeader.Filename)
	if err != nil {
		videoProxyError(c, http.StatusInternalServerError, "server_error", "failed to prepare upload")
		return
	}
	if _, err := io.CopyN(part, file, fileHeader.Size); err != nil {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", "failed to read upload")
		return
	}
	if err := writer.Close(); err != nil {
		videoProxyError(c, http.StatusInternalServerError, "server_error", "failed to prepare upload")
		return
	}

	request, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, strings.TrimRight(account.BaseURL(), "/")+"/api/sd-media/upload", &body)
	if err != nil {
		videoProxyError(c, http.StatusInternalServerError, "server_error", "failed to prepare upstream upload")
		return
	}
	request.Header.Set("Authorization", "Bearer "+account.ApiKey)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	client, err := service.GetHttpClientWithProxy(account.Proxy)
	if err != nil {
		videoProxyError(c, http.StatusInternalServerError, "server_error", "failed to create upstream client")
		return
	}
	response, err := client.Do(request)
	if err != nil {
		videoProxyError(c, http.StatusBadGateway, "server_error", "failed to upload reference media")
		return
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		videoProxyError(c, http.StatusBadGateway, "server_error", "failed to read upload response")
		return
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		c.Data(response.StatusCode, "application/json; charset=utf-8", responseBody)
		return
	}
	var payload map[string]any
	if err := common.Unmarshal(responseBody, &payload); err != nil {
		videoProxyError(c, http.StatusBadGateway, "server_error", "invalid upload response")
		return
	}
	if urlValue, ok := payload["url"].(string); ok && strings.TrimSpace(urlValue) != "" {
		payload[mediaType] = []string{urlValue}
	}
	c.JSON(http.StatusOK, payload)
}

func resolveVideoUploadAccount(c *gin.Context, form *multipart.Form) (*model.VideoAccount, error) {
	publicKey := strings.TrimSpace(c.GetHeader("X-Video-Creation-Token-Id"))
	group := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	if group == "" {
		group = common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	}
	if publicKey != "" {
		return model.FindVideoAccountForModelFromPublicKey(publicKey, group)
	}
	modelName := strings.TrimSpace(firstFormValue(form, "model"))
	if modelName != "" {
		return model.FindVideoAccountForModel(modelName, group, "")
	}
	accounts, err := model.ListEnabledVideoAccounts(group)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no enabled video account is configured")
	}
	return accounts[0], nil
}

func firstFormValue(form *multipart.Form, key string) string {
	if form == nil || len(form.Value[key]) == 0 {
		return ""
	}
	return form.Value[key][0]
}
