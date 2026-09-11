package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func proxyVideoAccountContent(c *gin.Context, task *model.Task) {
	account, err := model.GetVideoAccountById(task.VideoAccountId)
	if err != nil || account == nil {
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to retrieve video account")
		return
	}
	baseURL := account.BaseURL()
	contentURL := fmt.Sprintf("%s/v1/videos/%s/content", baseURL, url.PathEscape(task.GetUpstreamTaskID()))
	if err := service.ValidateSSRFProtectedFetchURL(contentURL); err != nil {
		videoProxyError(c, http.StatusForbidden, "server_error", fmt.Sprintf("request blocked: %v", err))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 600*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, contentURL, nil)
	if err != nil {
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to create proxy request")
		return
	}
	apiKey := account.ApiKey
	if task.PrivateData.Key != "" {
		apiKey = task.PrivateData.Key
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := service.GetSSRFProtectedHTTPClient()
	var clientErr error
	if account.Proxy != "" {
		client, clientErr = service.GetHttpClientWithProxy(account.Proxy)
	}
	if clientErr != nil {
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to create proxy client")
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch video content")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		videoProxyError(c, http.StatusBadGateway, "server_error", fmt.Sprintf("Upstream service returned status %d", resp.StatusCode))
		return
	}
	for key, values := range resp.Header {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}
	c.Writer.Header().Set("Cache-Control", "public, max-age=86400")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}
