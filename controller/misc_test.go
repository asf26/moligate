package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStatusIncludesAssistantVersionConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalAssistantVersion := common.AssistantVersion
	originalAssistantForceUpdate := common.AssistantForceUpdate
	originalAssistantReleaseNotes := common.AssistantReleaseNotes
	originalAssistantMacDownloadURL := common.AssistantMacDownloadURL
	originalAssistantMacSignature := common.AssistantMacSignature
	originalAssistantWinDownloadURL := common.AssistantWinDownloadURL
	originalAssistantWinSignature := common.AssistantWinSignature
	originalAssistantPublishedAt := common.AssistantPublishedAt
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMap = map[string]string{
		"HeaderNavModules":    "",
		"SidebarModulesAdmin": "",
		"NoticeForcePopup":    "false",
	}
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.AssistantVersion = originalAssistantVersion
		common.AssistantForceUpdate = originalAssistantForceUpdate
		common.AssistantReleaseNotes = originalAssistantReleaseNotes
		common.AssistantMacDownloadURL = originalAssistantMacDownloadURL
		common.AssistantMacSignature = originalAssistantMacSignature
		common.AssistantWinDownloadURL = originalAssistantWinDownloadURL
		common.AssistantWinSignature = originalAssistantWinSignature
		common.AssistantPublishedAt = originalAssistantPublishedAt
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	common.AssistantVersion = "1.2.3"
	common.AssistantForceUpdate = true
	common.AssistantReleaseNotes = "修复钱包汇率和版本检查"
	common.AssistantMacDownloadURL = "https://example.com/assistant.app.tar.gz"
	common.AssistantMacSignature = "mac-signature"
	common.AssistantWinDownloadURL = "https://example.com/assistant-setup.exe"
	common.AssistantWinSignature = "win-signature"
	common.AssistantPublishedAt = 1710000000

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	GetStatus(c)

	require.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, "1.2.3", response.Data["assistant_version"])
	assert.Equal(t, true, response.Data["assistant_force_update"])
	assert.Equal(t, "修复钱包汇率和版本检查", response.Data["assistant_release_notes"])
	assert.Equal(
		t,
		"https://example.com/assistant.app.tar.gz",
		response.Data["assistant_mac_download_url"],
	)
	assert.Equal(
		t,
		"https://example.com/assistant-setup.exe",
		response.Data["assistant_win_download_url"],
	)
	assert.Equal(t, float64(1710000000), response.Data["assistant_published_at"])
}

func TestGetAssistantVersionReturnsVersionCheckPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalAssistantVersion := common.AssistantVersion
	originalAssistantForceUpdate := common.AssistantForceUpdate
	originalAssistantReleaseNotes := common.AssistantReleaseNotes
	originalAssistantMacDownloadURL := common.AssistantMacDownloadURL
	originalAssistantMacSignature := common.AssistantMacSignature
	originalAssistantWinDownloadURL := common.AssistantWinDownloadURL
	originalAssistantWinSignature := common.AssistantWinSignature
	originalAssistantPublishedAt := common.AssistantPublishedAt

	t.Cleanup(func() {
		common.AssistantVersion = originalAssistantVersion
		common.AssistantForceUpdate = originalAssistantForceUpdate
		common.AssistantReleaseNotes = originalAssistantReleaseNotes
		common.AssistantMacDownloadURL = originalAssistantMacDownloadURL
		common.AssistantMacSignature = originalAssistantMacSignature
		common.AssistantWinDownloadURL = originalAssistantWinDownloadURL
		common.AssistantWinSignature = originalAssistantWinSignature
		common.AssistantPublishedAt = originalAssistantPublishedAt
	})

	common.AssistantVersion = "0.1.1"
	common.AssistantForceUpdate = false
	common.AssistantReleaseNotes = "修复钱包汇率和版本检查"
	common.AssistantMacDownloadURL = "https://xxx/魔力门助手_0.1.1.app.tar.gz"
	common.AssistantMacSignature = "mac-signature"
	common.AssistantWinDownloadURL = "https://xxx/魔力门助手_0.1.1_x64-setup.exe"
	common.AssistantWinSignature = "win-signature"
	common.AssistantPublishedAt = 1710000000

	req := httptest.NewRequest(http.MethodGet, "/api/assistant/version", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	GetAssistantVersion(c)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	assert.Len(t, response, 8)
	assert.Equal(t, "0.1.1", response["version"])
	assert.Equal(t, false, response["force_update"])
	assert.Equal(t, "修复钱包汇率和版本检查", response["release_notes"])
	assert.Equal(
		t,
		"https://xxx/魔力门助手_0.1.1.app.tar.gz",
		response["mac_download_url"],
	)
	assert.Equal(
		t,
		"https://xxx/魔力门助手_0.1.1_x64-setup.exe",
		response["win_download_url"],
	)
	assert.Equal(t, float64(1710000000), response["published_at"])
}

func TestGetAssistantVersionReturnsTauriManifestForUpdaterQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalAssistantVersion := common.AssistantVersion
	originalAssistantReleaseNotes := common.AssistantReleaseNotes
	originalAssistantMacDownloadURL := common.AssistantMacDownloadURL
	originalAssistantMacSignature := common.AssistantMacSignature
	originalAssistantWinDownloadURL := common.AssistantWinDownloadURL
	originalAssistantWinSignature := common.AssistantWinSignature
	originalAssistantPublishedAt := common.AssistantPublishedAt

	t.Cleanup(func() {
		common.AssistantVersion = originalAssistantVersion
		common.AssistantReleaseNotes = originalAssistantReleaseNotes
		common.AssistantMacDownloadURL = originalAssistantMacDownloadURL
		common.AssistantMacSignature = originalAssistantMacSignature
		common.AssistantWinDownloadURL = originalAssistantWinDownloadURL
		common.AssistantWinSignature = originalAssistantWinSignature
		common.AssistantPublishedAt = originalAssistantPublishedAt
	})

	common.AssistantVersion = "0.2.0"
	common.AssistantReleaseNotes = "自动更新"
	common.AssistantMacDownloadURL = "https://xxx/assistant.app.tar.gz"
	common.AssistantMacSignature = "mac-signature"
	common.AssistantWinDownloadURL = "https://xxx/assistant-setup.exe"
	common.AssistantWinSignature = "win-signature"
	common.AssistantPublishedAt = 1710000000

	req := httptest.NewRequest(http.MethodGet, "/api/assistant/version?target=darwin&arch=aarch64&current_version=0.1.0", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	GetAssistantVersion(c)

	require.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Version   string `json:"version"`
		URL       string `json:"url"`
		Signature string `json:"signature"`
		Notes     string `json:"notes"`
		PubDate   string `json:"pub_date"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, "0.2.0", response.Version)
	assert.Equal(t, "https://xxx/assistant.app.tar.gz", response.URL)
	assert.Equal(t, "mac-signature", response.Signature)
	assert.Equal(t, "自动更新", response.Notes)
	assert.Equal(t, "2024-03-09T16:00:00Z", response.PubDate)
}

func TestGetAssistantVersionReturnsNoContentWhenUpdaterHasNoNewVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalAssistantVersion := common.AssistantVersion
	originalAssistantMacDownloadURL := common.AssistantMacDownloadURL
	originalAssistantMacSignature := common.AssistantMacSignature

	t.Cleanup(func() {
		common.AssistantVersion = originalAssistantVersion
		common.AssistantMacDownloadURL = originalAssistantMacDownloadURL
		common.AssistantMacSignature = originalAssistantMacSignature
	})

	common.AssistantVersion = "0.1.0"
	common.AssistantMacDownloadURL = "https://xxx/assistant.app.tar.gz"
	common.AssistantMacSignature = "mac-signature"

	req := httptest.NewRequest(http.MethodGet, "/api/assistant/version?target=darwin&arch=x86_64&current_version=0.1.0", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	GetAssistantVersion(c)

	require.Equal(t, http.StatusNoContent, c.Writer.Status())
}
