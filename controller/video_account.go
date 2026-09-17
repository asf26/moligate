package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// videoAccountCreateRequest is intentionally not model.VideoAccount: binding
// a request directly to the model would make it too easy to echo the API key
// back to the browser or to let a caller override CTMOAI's fixed base URL.
type videoAccountCreateRequest struct {
	Name    string   `json:"name"`
	APIKey  string   `json:"api_key"`
	Key     string   `json:"key"` // compatibility alias for older admin forms
	Status  *int     `json:"status"`
	Groups  []string `json:"groups"`
	Group   string   `json:"group"`
	Proxy   string   `json:"proxy"`
	Remark  string   `json:"remark"`
	BaseURL string   `json:"base_url"`
}

type videoAccountUpdateRequest struct {
	ID            int                                  `json:"id"`
	Name          *string                              `json:"name"`
	APIKey        *string                              `json:"api_key"`
	Key           *string                              `json:"key"`
	Status        *int                                 `json:"status"`
	Groups        *[]string                            `json:"groups"`
	Group         *string                              `json:"group"`
	Proxy         *string                              `json:"proxy"`
	Remark        *string                              `json:"remark"`
	BaseURL       *string                              `json:"base_url"`
	BillingPrices *map[string]*model.VideoModelPricing `json:"billing_prices"`
	// ModelCapabilities carries administrator decisions for capabilities the
	// upstream catalog does not publish (currently the first/last frame mode).
	ModelCapabilities *map[string]*model.VideoModelCapabilityOverride `json:"model_capabilities"`
}

func videoAccountGroups(groups []string, group string) []string {
	if len(groups) > 0 {
		return groups
	}
	if strings.TrimSpace(group) == "" {
		return nil
	}
	return strings.FieldsFunc(group, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	})
}

func validateVideoAccountBaseURL(baseURL string) error {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL != "" && strings.TrimRight(baseURL, "/") != model.VideoAccountBaseURL {
		return fmt.Errorf("video account base URL is fixed to %s", model.VideoAccountBaseURL)
	}
	return nil
}

func buildVideoAccountView(account *model.VideoAccount) map[string]any {
	view := account.PublicView()
	if view == nil {
		return nil
	}
	// `token_id` is the public opaque selector used by the video-creation API;
	// retain `public_key` as an alias for clients built during the migration.
	view["token_id"] = account.OpaqueKey()
	return view
}

func buildVideoAccountViews(accounts []*model.VideoAccount) []map[string]any {
	views := make([]map[string]any, 0, len(accounts))
	for _, account := range accounts {
		if account != nil {
			views = append(views, buildVideoAccountView(account))
		}
	}
	return views
}

func parseVideoAccountID(c *gin.Context, bodyID int) (int, bool) {
	if raw := strings.TrimSpace(c.Param("id")); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			common.ApiErrorMsg(c, "无效的视频账号 ID")
			return 0, false
		}
		return id, true
	}
	if bodyID <= 0 {
		common.ApiErrorMsg(c, "缺少视频账号 ID")
		return 0, false
	}
	return bodyID, true
}

// GetVideoAccounts lists the dedicated CTMOAI accounts for administrators.
// API keys are never part of the response; PublicView emits only a mask.
func GetVideoAccounts(c *gin.Context) {
	accounts, err := model.ListVideoAccounts()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	statusFilter := strings.TrimSpace(c.Query("status"))
	search := strings.ToLower(strings.TrimSpace(c.Query("search")))
	filtered := make([]*model.VideoAccount, 0, len(accounts))
	statusValue := -1
	if statusFilter != "" {
		switch strings.ToLower(statusFilter) {
		case "enabled", "active", "1":
			statusValue = model.VideoAccountStatusEnabled
		case "disabled", "inactive", "0":
			statusValue = model.VideoAccountStatusDisabled
		default:
			common.ApiErrorMsg(c, "无效的视频账号状态")
			return
		}
	}
	for _, account := range accounts {
		if account == nil {
			continue
		}
		if statusValue >= 0 {
			if account.Status != statusValue {
				continue
			}
		}
		if search != "" && !strings.Contains(strings.ToLower(account.Name), search) && !strings.Contains(strings.ToLower(account.Remark), search) {
			continue
		}
		filtered = append(filtered, account)
	}

	pageInfo := common.GetPageQuery(c)
	total := len(filtered)
	start := pageInfo.GetStartIdx()
	if start > total {
		start = total
	}
	end := start + pageInfo.GetPageSize()
	if end > total {
		end = total
	}
	pageInfo.SetTotal(total)
	pageInfo.SetItems(buildVideoAccountViews(filtered[start:end]))
	common.ApiSuccess(c, gin.H{
		"items":     pageInfo.Items,
		"total":     total,
		"page":      pageInfo.GetPage(),
		"page_size": pageInfo.GetPageSize(),
	})
}

func GetVideoAccount(c *gin.Context) {
	id, ok := parseVideoAccountID(c, 0)
	if !ok {
		return
	}
	account, err := model.GetVideoAccountById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "视频账号不存在")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, buildVideoAccountView(account))
}

func CreateVideoAccount(c *gin.Context) {
	var req videoAccountCreateRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := validateVideoAccountBaseURL(req.BaseURL); err != nil {
		common.ApiError(c, err)
		return
	}
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(req.Key)
	}
	account, err := service.CreateVideoAccount(c.Request.Context(), service.VideoAccountCreateInput{
		Name:   req.Name,
		APIKey: apiKey,
		Status: req.Status,
		Groups: videoAccountGroups(req.Groups, req.Group),
		Proxy:  req.Proxy,
		Remark: req.Remark,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, buildVideoAccountView(account))
}

func UpdateVideoAccount(c *gin.Context) {
	var req videoAccountUpdateRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.APIKey == nil && req.Key != nil {
		req.APIKey = req.Key
	}
	if req.BaseURL != nil {
		if err := validateVideoAccountBaseURL(*req.BaseURL); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	var groups *[]string
	if req.Groups != nil {
		groups = req.Groups
	} else if req.Group != nil {
		parsed := videoAccountGroups(nil, *req.Group)
		groups = &parsed
	}
	id, ok := parseVideoAccountID(c, req.ID)
	if !ok {
		return
	}
	account, err := service.UpdateVideoAccount(c.Request.Context(), id, service.VideoAccountUpdateInput{
		Name:              req.Name,
		APIKey:            req.APIKey,
		Status:            req.Status,
		Groups:            groups,
		Proxy:             req.Proxy,
		Remark:            req.Remark,
		BillingPrices:     req.BillingPrices,
		ModelCapabilities: req.ModelCapabilities,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "视频账号不存在")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, buildVideoAccountView(account))
}

func DeleteVideoAccount(c *gin.Context) {
	id, ok := parseVideoAccountID(c, 0)
	if !ok {
		return
	}
	if err := model.DeleteVideoAccount(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "视频账号不存在")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func SyncVideoAccount(c *gin.Context) {
	id, ok := parseVideoAccountID(c, 0)
	if !ok {
		return
	}
	account, err := service.SyncVideoAccountModels(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "视频账号不存在")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, buildVideoAccountView(account))
}

// GetVideoCreationCatalog mirrors CTMOAI's session-facing catalog shape while
// sourcing models from the account-key-scoped snapshots stored by admins.
// The response intentionally keeps `data` as the model array (rather than
// nesting it inside the usual ApiSuccess data object) because the upstream UI
// reads e.data.data and e.data.private_groups.
func GetVideoCreationCatalog(c *gin.Context) {
	catalog, err := service.BuildVideoCreationCatalog(service.VideoAccountRequestGroups(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	balanceQuota := common.GetContextKeyInt(c, constant.ContextKeyUserQuota)
	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"message":        "",
		"data":           catalog.Models,
		"models":         catalog.Models,
		"private_groups": catalog.PrivateGroups,
		"balance_quota":  balanceQuota,
	})
}

func GetVideoCreationPrivateGroupModels(c *gin.Context) {
	key := strings.TrimSpace(c.Param("key"))
	if key == "" {
		common.ApiErrorMsg(c, "缺少视频账号 token_id")
		return
	}
	models, err := service.GetVideoAccountModelsForGroup(service.VideoAccountRequestGroups(c), key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "视频账号不存在")
			return
		}
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    models,
		"models":  models,
	})
}
