package controller

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

const (
	videoImageUploadLimit = 10 << 20
	videoVideoUploadLimit = 50 << 20
	videoAudioUploadLimit = 30 << 20
)

// UploadVideoAccountMedia follows the upstream's /v1/videos/uploads contract. The
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

	request, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, strings.TrimRight(account.BaseURL(), "/")+"/v1/videos/uploads", &body)
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
	// The upstream stores uploads behind its own bearer auth, and its generation
	// workers fetch references without that header — so handing the upstream URL
	// straight back to a caller breaks image-to-video with 503. Publish a signed
	// URL on our own domain instead; the worker can read it anonymously.
	mediaID, _ := payload["media_id"].(string)
	if strings.TrimSpace(mediaID) == "" {
		videoProxyError(c, http.StatusBadGateway, "server_error", "upstream did not return a media id")
		return
	}
	publicURL := videoAccountMediaPublicURL(c, account, mediaID)
	payload["upstream_url"] = payload["url"]
	payload["url"] = publicURL
	payload[mediaType] = []string{publicURL}
	c.JSON(http.StatusOK, payload)
}

func resolveVideoUploadAccount(c *gin.Context, form *multipart.Form) (*model.VideoAccount, error) {
	publicKey := strings.TrimSpace(c.GetHeader("X-Video-Creation-Token-Id"))
	groups := service.VideoAccountRequestGroups(c)
	if publicKey != "" {
		return model.FindVideoAccountForModelFromPublicKey(publicKey, groups)
	}
	modelName := strings.TrimSpace(firstFormValue(form, "model"))
	if modelName != "" {
		return model.FindVideoAccountForModel(modelName, groups, "")
	}
	accounts, err := model.ListUsableVideoAccounts(groups)
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

// videoAccountMediaToken binds one upstream media id to the account that owns the
// key, so the public route can neither guess nor cross accounts.
func videoAccountMediaToken(account *model.VideoAccount, mediaID string) string {
	mac := hmac.New(sha256.New, []byte(account.ApiKey))
	mac.Write([]byte(mediaID))
	return fmt.Sprintf("%d.%s.%x", account.Id, mediaID, mac.Sum(nil))
}

func videoAccountMediaPublicURL(c *gin.Context, account *model.VideoAccount, mediaID string) string {
	scheme := "https"
	if proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); proto != "" {
		scheme = strings.ToLower(strings.Split(proto, ",")[0])
	} else if c.Request.TLS == nil {
		scheme = "http"
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}
	return fmt.Sprintf("%s://%s/v1/videos/media/%s", scheme, host, url.PathEscape(videoAccountMediaToken(account, mediaID)))
}

// verifyVideoAccountMediaToken returns the media id when the token was signed by
// this account, and rejects anything else.
func verifyVideoAccountMediaToken(account *model.VideoAccount, token string) (string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || !strings.HasPrefix(parts[1], "vidmedia_") {
		return "", false
	}
	expected := videoAccountMediaToken(account, parts[1])
	if subtle.ConstantTimeCompare([]byte(expected), []byte(token)) != 1 {
		return "", false
	}
	return parts[1], true
}

// ServeVideoAccountMedia streams an uploaded reference so the upstream can read
// it without credentials. The token is signed with the account key, which keeps
// the route public without exposing anything but media this gateway uploaded.
func ServeVideoAccountMedia(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		videoProxyError(c, http.StatusNotFound, "video_resource_not_found", "media not found")
		return
	}
	accountID, err := strconv.Atoi(parts[0])
	if err != nil || accountID <= 0 {
		videoProxyError(c, http.StatusNotFound, "video_resource_not_found", "media not found")
		return
	}
	account, err := model.GetVideoAccountById(accountID)
	if err != nil || !account.IsEnabled() {
		videoProxyError(c, http.StatusNotFound, "video_resource_not_found", "media not found")
		return
	}
	mediaID, ok := verifyVideoAccountMediaToken(account, token)
	if !ok {
		videoProxyError(c, http.StatusNotFound, "video_resource_not_found", "media not found")
		return
	}

	upstream := strings.TrimRight(account.BaseURL(), "/") + "/v1/videos/uploads/" + mediaID + "/content"
	request, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, upstream, nil)
	if err != nil {
		videoProxyError(c, http.StatusBadGateway, "server_error", "failed to prepare media request")
		return
	}
	request.Header.Set("Authorization", "Bearer "+account.ApiKey)
	if rng := strings.TrimSpace(c.GetHeader("Range")); rng != "" {
		request.Header.Set("Range", rng)
	}
	client, err := service.GetHttpClientWithProxy(account.Proxy)
	if err != nil {
		videoProxyError(c, http.StatusInternalServerError, "server_error", "failed to create media client")
		return
	}
	response, err := client.Do(request)
	if err != nil {
		videoProxyError(c, http.StatusBadGateway, "server_error", "failed to fetch media")
		return
	}
	defer response.Body.Close()
	for _, header := range []string{"Content-Type", "Content-Length", "Accept-Ranges", "Content-Range", "Last-Modified"} {
		if value := response.Header.Get(header); value != "" {
			c.Header(header, value)
		}
	}
	// Media ids expire upstream after 24h; let caches keep a short copy only.
	c.Header("Cache-Control", "public, max-age=900")
	c.Status(response.StatusCode)
	_, _ = io.Copy(c.Writer, io.LimitReader(response.Body, 64<<20))
}
