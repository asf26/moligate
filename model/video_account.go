package model

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// VideoAccountBaseURL is intentionally fixed. CTMOAI exposes one public
// OpenAI-compatible video gateway and allowing a browser/admin to override the
// URL would turn this feature into another generic channel configuration.
const VideoAccountBaseURL = "https://video.ctmoai.com"

const (
	VideoAccountStatusDisabled = 0
	VideoAccountStatusEnabled  = 1
)

// VideoModelPricing is a price snapshot. Pricing is the upstream reference
// price; BillingPricing, when present, is the administrator-defined price
// charged by this gateway for this account/model.
type VideoModelPricing struct {
	Mode     string  `json:"mode,omitempty"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency,omitempty"`
}

// VideoModel describes the capabilities of one model for one account. The
// upstream model catalog is key-scoped, so this must not be replaced with a
// global hard-coded model list.
type VideoModel struct {
	ID                     string   `json:"id"`
	DisplayName            string   `json:"display_name,omitempty"`
	ProductKey             string   `json:"product_key,omitempty"`
	Group                  string   `json:"group,omitempty"`
	PrivateGroupKey        string   `json:"private_group_key,omitempty"`
	Available              bool     `json:"available"`
	UnavailableReason      string   `json:"unavailable_reason,omitempty"`
	SupportedEndpointTypes []string `json:"supported_endpoint_types,omitempty"`
	Resolution             string   `json:"resolution,omitempty"`
	DurationsSeconds       []int    `json:"durations_seconds,omitempty"`
	Ratios                 []string `json:"ratios,omitempty"`
	Sizes                  []string `json:"sizes,omitempty"`
	MaxImages              int      `json:"max_images"`
	MaxVideos              int      `json:"max_videos"`
	MaxAudios              int      `json:"max_audios"`
	// MaxVideoDurationSeconds is the longest single clip the model accepts. It
	// travels with the catalog for completeness; DurationsSeconds is what bounds
	// a request, because it lists the only accepted values.
	MaxVideoDurationSeconds int                `json:"max_video_duration_seconds,omitempty"`
	AudioRequiresImage      bool               `json:"audio_requires_image,omitempty"`
	SupportsFirstLastFrame  bool               `json:"supports_first_last_frame,omitempty"`
	Pricing                 VideoModelPricing  `json:"pricing,omitempty"`
	BillingPricing          *VideoModelPricing `json:"billing_pricing,omitempty"`
	GroupRatio              float64            `json:"group_ratio,omitempty"`
}

// EffectivePricing returns the account-level gateway price when one has been
// configured, otherwise the price reported by CTMOAI.
func (item VideoModel) EffectivePricing() VideoModelPricing {
	if item.BillingPricing != nil {
		return *item.BillingPricing
	}
	return item.Pricing
}

// VideoAccount stores credentials separately from the generic channels table.
// ApiKey is never serialized to an API response; PublicKey is an opaque
// server-side account selector used by the video creation UI.
type VideoAccount struct {
	Id              int    `json:"id" gorm:"primaryKey"`
	Name            string `json:"name" gorm:"type:varchar(128);not null;index"`
	ApiKey          string `json:"-" gorm:"column:api_key;type:text;not null"`
	PublicKey       string `json:"-" gorm:"column:public_key;type:varchar(96);not null;uniqueIndex"`
	Status          int    `json:"status" gorm:"index"`
	Groups          string `json:"groups" gorm:"column:groups;type:varchar(255);not null"`
	ModelsJSON      string `json:"-" gorm:"column:models_json;type:text"`
	LastModelSyncAt int64  `json:"last_model_sync_at" gorm:"index"`
	Proxy           string `json:"proxy,omitempty" gorm:"type:text"`
	Remark          string `json:"remark,omitempty" gorm:"type:text"`
	CreatedAt       int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (account *VideoAccount) IsEnabled() bool {
	return account != nil && account.Status == VideoAccountStatusEnabled
}

// BaseURL returns the only upstream origin supported by this dedicated
// integration. It is kept as a method so relay/middleware code cannot
// accidentally reintroduce per-account arbitrary endpoints.
func (account *VideoAccount) BaseURL() string {
	return VideoAccountBaseURL
}

func (account *VideoAccount) GroupList() []string {
	if account == nil {
		return nil
	}
	groups := strings.FieldsFunc(account.Groups, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	})
	result := make([]string, 0, len(groups))
	seen := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		key := strings.ToLower(group)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, group)
	}
	if len(result) == 0 {
		return []string{"default"}
	}
	return result
}

// MatchesGroups reports whether a request routed under any of groups may use
// this account. Groups is an authorization list rather than an identity: the
// account is usable when the routing groups intersect it, or when it lists the
// all/* wildcard. An empty caller set matches nothing, so an unresolved caller
// never inherits every account.
func (account *VideoAccount) MatchesGroups(groups []string) bool {
	if account == nil {
		return false
	}
	candidates := account.GroupList()
	for _, candidate := range candidates {
		if isWildcardVideoAccountGroup(candidate) {
			return true
		}
	}
	for _, candidate := range candidates {
		for _, group := range groups {
			group = strings.TrimSpace(group)
			if group == "" {
				continue
			}
			if strings.EqualFold(candidate, group) {
				return true
			}
		}
	}
	return false
}

func isWildcardVideoAccountGroup(group string) bool {
	group = strings.TrimSpace(group)
	return strings.EqualFold(group, "all") || group == "*"
}

func (account *VideoAccount) ModelCatalog() []VideoModel {
	if account == nil || strings.TrimSpace(account.ModelsJSON) == "" {
		return []VideoModel{}
	}
	var models []VideoModel
	if err := common.Unmarshal([]byte(account.ModelsJSON), &models); err != nil {
		return []VideoModel{}
	}
	result := make([]VideoModel, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, item := range models {
		item.ID = strings.TrimSpace(item.ID)
		if item.ID == "" {
			continue
		}
		key := strings.ToLower(item.ID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if item.DisplayName == "" {
			item.DisplayName = item.ID
		}
		if item.GroupRatio <= 0 {
			item.GroupRatio = 1
		}
		result = append(result, item.WithDerivedCapabilities())
	}
	return result
}

func (account *VideoAccount) SetModelCatalog(models []VideoModel) error {
	if account == nil {
		return errors.New("video account is nil")
	}
	encoded, err := common.Marshal(models)
	if err != nil {
		return fmt.Errorf("marshal video model catalog: %w", err)
	}
	account.ModelsJSON = string(encoded)
	return nil
}

// SetModelBillingPrices updates only administrator-defined prices. A nil
// value clears an override and makes the model follow CTMOAI again.
func (account *VideoAccount) SetModelBillingPrices(prices map[string]*VideoModelPricing) error {
	if account == nil {
		return errors.New("video account is nil")
	}
	models := account.ModelCatalog()
	indexes := make(map[string]int, len(models))
	for index, item := range models {
		indexes[strings.ToLower(item.ID)] = index
	}
	for rawID, price := range prices {
		id := strings.TrimSpace(rawID)
		index, ok := indexes[strings.ToLower(id)]
		if !ok {
			return fmt.Errorf("video model %q is not available for this account", id)
		}
		if price == nil {
			models[index].BillingPricing = nil
			continue
		}
		if math.IsNaN(price.Amount) || math.IsInf(price.Amount, 0) || price.Amount < 0 || price.Amount > 1_000_000 {
			return fmt.Errorf("invalid billing price for video model %q", id)
		}
		normalized := *price
		normalized.Mode = strings.TrimSpace(normalized.Mode)
		normalized.Currency = strings.TrimSpace(normalized.Currency)
		if normalized.Currency == "" {
			normalized.Currency = strings.TrimSpace(models[index].Pricing.Currency)
		}
		if normalized.Currency == "" {
			normalized.Currency = "CNY"
		}
		if normalized.Mode == "" {
			normalized.Mode = strings.TrimSpace(models[index].Pricing.Mode)
		}
		models[index].BillingPricing = &normalized
	}
	return account.SetModelCatalog(models)
}

func (account *VideoAccount) FindModel(modelName string) (*VideoModel, bool) {
	modelName = strings.TrimSpace(modelName)
	for _, item := range account.ModelCatalog() {
		if strings.EqualFold(item.ID, modelName) {
			copy := item
			return &copy, true
		}
	}
	return nil, false
}

func (account *VideoAccount) OpaqueKey() string {
	if account == nil {
		return ""
	}
	if strings.TrimSpace(account.PublicKey) != "" {
		return account.PublicKey
	}
	return "video-account-" + strconv.Itoa(account.Id)
}

// PublicView is safe to return to the administrator UI. It deliberately
// returns only a masked key and the dynamic model snapshot.
func (account *VideoAccount) PublicView() map[string]any {
	if account == nil {
		return nil
	}
	masked := ""
	if len(account.ApiKey) > 8 {
		masked = account.ApiKey[:4] + "••••" + account.ApiKey[len(account.ApiKey)-4:]
	} else if account.ApiKey != "" {
		masked = "••••"
	}
	return map[string]any{
		"id":                 account.Id,
		"name":               account.Name,
		"base_url":           VideoAccountBaseURL,
		"status":             account.Status,
		"groups":             account.GroupList(),
		"models":             account.ModelCatalog(),
		"public_key":         account.OpaqueKey(),
		"token_id":           account.OpaqueKey(),
		"api_key_masked":     masked,
		"has_api_key":        account.ApiKey != "",
		"last_model_sync_at": account.LastModelSyncAt,
		"proxy":              account.Proxy,
		"remark":             account.Remark,
		"created_at":         account.CreatedAt,
		"updated_at":         account.UpdatedAt,
	}
}

func ListVideoAccounts() ([]*VideoAccount, error) {
	var accounts []*VideoAccount
	if err := DB.Order("id asc").Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func GetVideoAccountById(id int) (*VideoAccount, error) {
	if id <= 0 {
		return nil, errors.New("invalid video account id")
	}
	var account VideoAccount
	if err := DB.First(&account, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func GetVideoAccountByPublicKey(key string) (*VideoAccount, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("video account key is empty")
	}
	var account VideoAccount
	if err := DB.Where("public_key = ?", key).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

// ListUsableVideoAccounts returns every enabled account that a request routed
// under the given group set may use, ordered by id.
func ListUsableVideoAccounts(groups []string) ([]*VideoAccount, error) {
	var accounts []*VideoAccount
	if err := DB.Where("status = ?", VideoAccountStatusEnabled).Order("id asc").Find(&accounts).Error; err != nil {
		return nil, err
	}
	result := make([]*VideoAccount, 0, len(accounts))
	for _, account := range accounts {
		if account.MatchesGroups(groups) {
			result = append(result, account)
		}
	}
	return result, nil
}

func FindVideoAccountForModel(modelName string, groups []string, publicKey string) (*VideoAccount, error) {
	if strings.TrimSpace(publicKey) != "" {
		account, err := GetVideoAccountByPublicKey(publicKey)
		if err != nil {
			return nil, err
		}
		if !account.IsEnabled() || !account.MatchesGroups(groups) {
			return nil, errors.New("video account is unavailable for this group")
		}
		item, ok := account.FindModel(modelName)
		if !ok {
			return nil, fmt.Errorf("video model %q is not available in this account", modelName)
		}
		if !item.Available {
			return nil, fmt.Errorf("video model %q is currently unavailable", modelName)
		}
		return account, nil
	}
	accounts, err := ListUsableVideoAccounts(groups)
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		if item, ok := account.FindModel(modelName); ok && item.Available {
			return account, nil
		}
	}
	return nil, fmt.Errorf("no enabled video account supports model %q", modelName)
}

func FindVideoAccountForModelFromPublicKey(publicKey string, groups []string) (*VideoAccount, error) {
	account, err := GetVideoAccountByPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	if !account.IsEnabled() || !account.MatchesGroups(groups) {
		return nil, errors.New("video account is unavailable for this group")
	}
	return account, nil
}
