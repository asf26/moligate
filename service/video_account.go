package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const (
	videoAccountModelsTimeout = 30 * time.Second
	videoAccountModelsBodyMax = 8 << 20
)

// videoAccountBaseURL is a variable rather than a second configuration knob:
// it is only a test seam. Production always uses model.VideoAccountBaseURL.
var videoAccountBaseURL = model.VideoAccountBaseURL

// VideoAccountCreateInput and VideoAccountUpdateInput are deliberately kept
// separate from model.VideoAccount so an API request can distinguish an
// omitted key from an explicit replacement without ever binding a database
// record (and its secret) directly from JSON.
type VideoAccountCreateInput struct {
	Name   string
	APIKey string
	Status *int
	Groups []string
	Proxy  string
	Remark string
}

type VideoAccountUpdateInput struct {
	Name   *string
	APIKey *string
	Status *int
	Groups *[]string
	Proxy  *string
	Remark *string
	// BillingPrices is keyed by model ID. A nil value clears that model's
	// override; a nil map means the caller did not change pricing.
	BillingPrices *map[string]*model.VideoModelPricing
}

type VideoPrivateGroup struct {
	Key        string `json:"key"`
	Name       string `json:"name"`
	Group      string `json:"group,omitempty"`
	ModelCount int    `json:"model_count"`
}

type VideoCreationCatalog struct {
	Models        []model.VideoModel  `json:"models"`
	PrivateGroups []VideoPrivateGroup `json:"private_groups"`
}

// upstreamVideoModel uses pointer fields for values whose omission has a
// meaningful default. In particular, older CTMOAI responses omitted
// available/supported_endpoint_types; those models should remain usable.
type upstreamVideoModel struct {
	ID                      string               `json:"id"`
	DisplayName             string               `json:"display_name"`
	ProductKey              string               `json:"product_key"`
	Group                   string               `json:"group"`
	Available               *bool                `json:"available"`
	UnavailableReason       string               `json:"unavailable_reason"`
	SupportedEndpointTypes  []string             `json:"supported_endpoint_types"`
	Resolution              string               `json:"resolution"`
	DurationsSeconds        []int                `json:"durations_seconds"`
	Ratios                  []string             `json:"ratios"`
	Sizes                   []string             `json:"sizes"`
	MaxImages               *int                 `json:"max_images"`
	MaxVideos               *int                 `json:"max_videos"`
	MaxAudios               *int                 `json:"max_audios"`
	MaxVideoDurationSeconds *int                 `json:"max_video_duration_seconds"`
	AudioRequiresImage      bool                 `json:"audio_requires_image"`
	SupportsFirstLastFrame  bool                 `json:"supports_first_last_frame"`
	Pricing                 upstreamVideoPricing `json:"pricing"`
	GroupRatio              float64              `json:"group_ratio"`
}

type upstreamVideoPricing struct {
	Mode     string `json:"mode"`
	Amount   any    `json:"amount"`
	Currency string `json:"currency"`
}

type upstreamVideoModelsResponse struct {
	Data    []upstreamVideoModel `json:"data"`
	Models  []upstreamVideoModel `json:"models"`
	Message string               `json:"message"`
	Error   any                  `json:"error"`
}

// FetchVideoAccountModels reads the key-scoped catalog from CTMOAI. It is the
// only source of model capabilities and pricing for a dedicated account;
// model names and prices must not be hard-coded in the gateway.
func FetchVideoAccountModels(ctx context.Context, apiKey, proxyURL string) ([]model.VideoModel, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("video account api key is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	requestContext, cancel := context.WithTimeout(ctx, videoAccountModelsTimeout)
	defer cancel()

	endpoint := strings.TrimRight(videoAccountBaseURL, "/") + "/v1/models"
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create video model request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+apiKey)

	client, err := GetHttpClientWithProxy(strings.TrimSpace(proxyURL))
	if err != nil {
		return nil, fmt.Errorf("create video model http client: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch video model catalog: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, videoAccountModelsBodyMax))
	if err != nil {
		return nil, fmt.Errorf("read video model catalog: %w", err)
	}

	var payload upstreamVideoModelsResponse
	if len(body) > 0 {
		if err := common.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("decode video model catalog: %w", err)
		}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(payload.Message)
		if errorMessage := upstreamVideoErrorMessage(payload.Error); errorMessage != "" {
			message = errorMessage
		}
		if message == "" {
			message = strings.TrimSpace(string(body))
		}
		if len(message) > 512 {
			message = message[:512]
		}
		if message == "" {
			message = response.Status
		}
		message = strings.ReplaceAll(message, apiKey, "[redacted]")
		return nil, fmt.Errorf("video model catalog request failed (%d): %s", response.StatusCode, message)
	}

	items := payload.Data
	if len(items) == 0 {
		items = payload.Models
	}
	models := make([]model.VideoModel, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		converted := convertUpstreamVideoModel(item)
		if converted.ID == "" {
			continue
		}
		key := strings.ToLower(converted.ID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		models = append(models, converted)
	}
	return models, nil
}

func upstreamVideoErrorMessage(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if message, ok := typed["message"].(string); ok {
			return strings.TrimSpace(message)
		}
	}
	return ""
}

func convertUpstreamVideoModel(item upstreamVideoModel) model.VideoModel {
	available := true
	if item.Available != nil {
		available = *item.Available
	}
	group := strings.TrimSpace(item.Group)
	if group == "" {
		group = inferVideoModelGroup(item.ID)
	}
	endpoints := normalizeVideoEndpointTypes(item.SupportedEndpointTypes)
	if len(endpoints) == 0 {
		endpoints = []string{"openai-video"}
	}
	groupRatio := item.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1
	}
	displayName := strings.TrimSpace(item.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(item.ID)
	}
	pricingAmount, pricingValid := parseVideoPriceAmount(item.Pricing.Amount)
	if !pricingValid {
		available = false
		if strings.TrimSpace(item.UnavailableReason) == "" {
			item.UnavailableReason = "invalid pricing amount"
		}
	}
	maxImages, maxVideos, maxAudios, maxVideoDuration := -1, -1, -1, -1
	if item.MaxImages != nil {
		maxImages = *item.MaxImages
	}
	if item.MaxVideos != nil {
		maxVideos = *item.MaxVideos
	}
	if item.MaxAudios != nil {
		maxAudios = *item.MaxAudios
	}
	if item.MaxVideoDurationSeconds != nil {
		maxVideoDuration = *item.MaxVideoDurationSeconds
	}
	return model.VideoModel{
		ID:                      strings.TrimSpace(item.ID),
		DisplayName:             displayName,
		ProductKey:              strings.TrimSpace(item.ProductKey),
		Group:                   group,
		Available:               available,
		UnavailableReason:       strings.TrimSpace(item.UnavailableReason),
		SupportedEndpointTypes:  endpoints,
		Resolution:              strings.TrimSpace(item.Resolution),
		DurationsSeconds:        append([]int(nil), item.DurationsSeconds...),
		Ratios:                  append([]string(nil), item.Ratios...),
		Sizes:                   append([]string(nil), item.Sizes...),
		MaxImages:               maxImages,
		MaxVideos:               maxVideos,
		MaxAudios:               maxAudios,
		MaxVideoDurationSeconds: maxVideoDuration,
		AudioRequiresImage:      item.AudioRequiresImage,
		SupportsFirstLastFrame:  item.SupportsFirstLastFrame,
		Pricing: model.VideoModelPricing{
			Mode:     strings.TrimSpace(item.Pricing.Mode),
			Amount:   pricingAmount,
			Currency: strings.TrimSpace(item.Pricing.Currency),
		},
		GroupRatio: groupRatio,
	}
}

func parseVideoPriceAmount(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, typed >= 0 && !math.IsNaN(typed) && !math.IsInf(typed, 0)
	case float32:
		amount := float64(typed)
		return amount, amount >= 0 && !math.IsNaN(amount) && !math.IsInf(amount, 0)
	case int:
		return float64(typed), typed >= 0
	case int64:
		return float64(typed), typed >= 0
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return parsed, parsed >= 0 && !math.IsNaN(parsed) && !math.IsInf(parsed, 0)
		}
	}
	return 0, false
}

func normalizeVideoEndpointTypes(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func inferVideoModelGroup(modelID string) string {
	modelID = strings.ToLower(strings.TrimSpace(modelID))
	switch {
	case strings.Contains(modelID, "minimax-h3"):
		return "minimax-h3"
	case strings.Contains(modelID, "seedance"):
		return "seedance"
	default:
		return "video"
	}
}

func normalizeAccountModels(models []model.VideoModel, account *model.VideoAccount) []model.VideoModel {
	if account == nil {
		return models
	}
	existingPrices := make(map[string]*model.VideoModelPricing)
	for _, existing := range account.ModelCatalog() {
		if existing.BillingPricing == nil {
			continue
		}
		price := *existing.BillingPricing
		existingPrices[strings.ToLower(existing.ID)] = &price
	}
	result := make([]model.VideoModel, 0, len(models))
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
		// The upstream private_group_key is an upstream-internal selector. Never
		// expose it to the browser or use it for local routing: the gateway must
		// route back through this account row using our own opaque token_id.
		item.PrivateGroupKey = account.OpaqueKey()
		item.ProductKey = item.PrivateGroupKey + ":" + item.ID
		if item.GroupRatio <= 0 {
			item.GroupRatio = 1
		}
		if len(item.SupportedEndpointTypes) == 0 {
			item.SupportedEndpointTypes = []string{"openai-video"}
		}
		if price, ok := existingPrices[key]; ok {
			item.BillingPricing = price
		}
		result = append(result, item)
	}
	return result
}

func CreateVideoAccount(ctx context.Context, input VideoAccountCreateInput) (*model.VideoAccount, error) {
	account := &model.VideoAccount{
		Name:   strings.TrimSpace(input.Name),
		ApiKey: strings.TrimSpace(input.APIKey),
		Groups: model.NormalizeVideoAccountGroups(input.Groups),
		Proxy:  strings.TrimSpace(input.Proxy),
		Remark: strings.TrimSpace(input.Remark),
		Status: model.VideoAccountStatusEnabled,
	}
	if input.Status != nil {
		account.Status = *input.Status
	}
	if err := model.ValidateVideoAccount(account, true); err != nil {
		return nil, err
	}
	publicKey, err := model.GenerateVideoAccountTokenID()
	if err != nil {
		return nil, err
	}
	account.PublicKey = publicKey
	models, err := FetchVideoAccountModels(ctx, account.ApiKey, account.Proxy)
	if err != nil {
		return nil, err
	}
	if err := account.SetModelCatalog(normalizeAccountModels(models, account)); err != nil {
		return nil, err
	}
	account.LastModelSyncAt = time.Now().Unix()
	if err := model.CreateVideoAccount(account); err != nil {
		return nil, err
	}
	return account, nil
}

func UpdateVideoAccount(ctx context.Context, id int, input VideoAccountUpdateInput) (*model.VideoAccount, error) {
	account, err := model.GetVideoAccountById(id)
	if err != nil {
		return nil, err
	}
	keyChanged := false
	proxyChanged := false
	if input.Name != nil {
		account.Name = strings.TrimSpace(*input.Name)
	}
	if input.APIKey != nil && strings.TrimSpace(*input.APIKey) != "" && !isMaskedVideoAPIKey(*input.APIKey) {
		newKey := strings.TrimSpace(*input.APIKey)
		keyChanged = newKey != account.ApiKey
		account.ApiKey = newKey
	}
	if input.Status != nil {
		account.Status = *input.Status
	}
	if input.Groups != nil {
		account.Groups = model.NormalizeVideoAccountGroups(*input.Groups)
	}
	if input.Proxy != nil {
		newProxy := strings.TrimSpace(*input.Proxy)
		proxyChanged = newProxy != account.Proxy
		account.Proxy = newProxy
	}
	if input.Remark != nil {
		account.Remark = strings.TrimSpace(*input.Remark)
	}
	if err := model.ValidateVideoAccount(account, true); err != nil {
		return nil, err
	}
	if keyChanged || proxyChanged || len(account.ModelCatalog()) == 0 {
		models, fetchErr := FetchVideoAccountModels(ctx, account.ApiKey, account.Proxy)
		if fetchErr != nil {
			return nil, fetchErr
		}
		if err := account.SetModelCatalog(normalizeAccountModels(models, account)); err != nil {
			return nil, err
		}
		account.LastModelSyncAt = time.Now().Unix()
	}
	if input.BillingPrices != nil {
		if err := account.SetModelBillingPrices(*input.BillingPrices); err != nil {
			return nil, err
		}
	}
	if err := model.UpdateVideoAccount(account); err != nil {
		return nil, err
	}
	return account, nil
}

func isMaskedVideoAPIKey(value string) bool {
	value = strings.TrimSpace(value)
	return value == "***" || value == "••••" || strings.Contains(value, "••••")
}

func SyncVideoAccountModels(ctx context.Context, id int) (*model.VideoAccount, error) {
	account, err := model.GetVideoAccountById(id)
	if err != nil {
		return nil, err
	}
	models, err := FetchVideoAccountModels(ctx, account.ApiKey, account.Proxy)
	if err != nil {
		return nil, err
	}
	if err := account.SetModelCatalog(normalizeAccountModels(models, account)); err != nil {
		return nil, err
	}
	account.LastModelSyncAt = time.Now().Unix()
	if err := model.UpdateVideoAccount(account); err != nil {
		return nil, err
	}
	return account, nil
}

// VideoAccountRequestGroups resolves the group set a dedicated video account is
// authorized against for the current request. It is the API key's group when
// the caller presents a key - including the concrete groups a key set to "auto"
// expands to - and the user's own group for a dashboard session.
//
// Catalog and relay must both go through this helper: if they disagreed, the
// workspace would offer a model that the relay then refuses.
func VideoAccountRequestGroups(c *gin.Context) []string {
	usingGroup := strings.TrimSpace(common.GetContextKeyString(c, constant.ContextKeyUsingGroup))
	if usingGroup == "auto" {
		return GetRequestAutoGroups(c, common.GetContextKeyString(c, constant.ContextKeyUserGroup))
	}
	if usingGroup != "" {
		return []string{usingGroup}
	}
	if userGroup := strings.TrimSpace(common.GetContextKeyString(c, constant.ContextKeyUserGroup)); userGroup != "" {
		return []string{userGroup}
	}
	return nil
}

// BuildVideoCreationCatalog lists the dedicated accounts the caller may route
// to. The groups come from the request context (the key's selected groups), so
// a key bound to a video group sees exactly the accounts authorized for it.
func BuildVideoCreationCatalog(groups []string) (*VideoCreationCatalog, error) {
	accounts, err := model.ListUsableVideoAccounts(groups)
	if err != nil {
		return nil, err
	}
	catalog := &VideoCreationCatalog{
		Models:        make([]model.VideoModel, 0),
		PrivateGroups: make([]VideoPrivateGroup, 0, len(accounts)),
	}
	for _, account := range accounts {
		models := videoCreationModels(account)
		catalog.Models = append(catalog.Models, models...)
		authorized := account.GroupList()
		group := ""
		if len(authorized) > 0 {
			group = authorized[0]
		}
		catalog.PrivateGroups = append(catalog.PrivateGroups, VideoPrivateGroup{
			Key:        account.OpaqueKey(),
			Name:       account.Name,
			Group:      group,
			ModelCount: len(models),
		})
	}
	return catalog, nil
}

// GetVideoAccountModelsForGroup returns the models of one dedicated account,
// but only when the requesting key's group set covers the account. The set must
// come from the request context, never from the caller's body, so a key cannot
// widen its own access.
func GetVideoAccountModelsForGroup(groups []string, key string) ([]model.VideoModel, error) {
	account, err := model.GetVideoAccountByPublicKey(key)
	if err != nil {
		return nil, err
	}
	if !account.IsEnabled() || !account.MatchesGroups(groups) {
		return nil, errors.New("video account is unavailable for this group")
	}
	return videoCreationModels(account), nil
}

func videoCreationModels(account *model.VideoAccount) []model.VideoModel {
	if account == nil {
		return []model.VideoModel{}
	}
	models := normalizeAccountModels(account.ModelCatalog(), account)
	for index := range models {
		models[index].Pricing = models[index].EffectivePricing()
		models[index].BillingPricing = nil
	}
	return models
}
