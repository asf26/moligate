/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package controller

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

const canvasTokenName = "infinite-canvas"

type canvasGroupResponse struct {
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	APIKey    string                `json:"api_key"`
	KeyID     int                   `json:"key_id"`
	KeyName   string                `json:"key_name"`
	GroupID   string                `json:"group_id"`
	GroupName string                `json:"group_name"`
	Models    []canvasModelResponse `json:"models"`
}

type canvasModelResponse struct {
	Name       string                    `json:"name"`
	Capability string                    `json:"capability"`
	Video      *canvasVideoModelMetadata `json:"video,omitempty"`
}

// canvasVideoModelMetadata carries a dedicated video account's catalog entry to
// the creation workspace. VideoAccountTokenID pins the exact account the model
// must be routed to, so the browser can submit the request with its own API key
// while the gateway keeps the upstream credential on the server.
type canvasVideoModelMetadata struct {
	VideoAccountTokenID string `json:"video_account_token_id"`
	Group               string `json:"group,omitempty"`
	// Family names the upstream integration. The two integrations share an
	// endpoint but not a request dialect, so the workspace needs it to pick the
	// right field spelling instead of guessing from the model id.
	Family           string   `json:"family,omitempty"`
	Resolution       string   `json:"resolution,omitempty"`
	DurationsSeconds []int    `json:"durations_seconds,omitempty"`
	Ratios           []string `json:"ratios,omitempty"`
	Sizes            []string `json:"sizes,omitempty"`
	// RatioSizes tells the workspace which size value to submit for each
	// supported ratio. It is the only way the browser can send a legal size,
	// because the models endpoint publishes ratios and resolution but not the
	// mapping between them.
	RatioSizes         map[string]string `json:"ratio_sizes,omitempty"`
	MaxImages          int               `json:"max_images"`
	MaxVideos          int               `json:"max_videos"`
	MaxAudios          int               `json:"max_audios"`
	AudioRequiresImage bool              `json:"audio_requires_image,omitempty"`
	// RequiresImage marks models that have no text-to-video workflow, so the
	// workspace can require a reference image instead of letting the request
	// fail after it reaches the gateway.
	RequiresImage          bool `json:"requires_image,omitempty"`
	SupportsFirstLastFrame bool `json:"supports_first_last_frame,omitempty"`
	// PricingMode plus the amount below let the workspace show what a generation
	// costs before the user commits to it. The amount is per second or per task,
	// following PricingMode.
	PricingMode     string  `json:"pricing_mode,omitempty"`
	PricingAmount   float64 `json:"pricing_amount,omitempty"`
	PricingCurrency string  `json:"pricing_currency,omitempty"`
}

type canvasConfigResponse struct {
	Groups []canvasGroupResponse `json:"groups"`
}

func GetCanvasConfig(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.ApiErrorMsg(c, "invalid user")
		return
	}

	user, err := model.GetUserById(userID, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	userGroup := user.Group

	// Refresh endpoint capabilities before deciding which models are suitable
	// for the creation workspace.
	model.GetPricing()
	usableGroups := service.GetUserUsableGroups(userGroup)

	var tokens []*model.Token
	if err := model.DB.Where("user_id = ? AND status = ?", userID, common.TokenStatusEnabled).
		Order("id asc").Find(&tokens).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	groups := make([]canvasGroupResponse, 0, len(tokens))
	for _, token := range tokens {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(token.Name)), canvasTokenName+":") {
			continue
		}
		tokenGroups, err := canvasTokenGroups(userGroup, token)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if len(tokenGroups) == 0 {
			continue
		}
		canvasModels, err := canvasCreationModelsForTokenGroups(tokenGroups, token)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		groupNames := make([]string, 0, len(tokenGroups))
		for _, groupID := range tokenGroups {
			groupName := strings.TrimSpace(usableGroups[groupID])
			if groupName == "" {
				groupName = groupID
			}
			groupNames = append(groupNames, groupName)
		}
		groupID := strings.Join(tokenGroups, ",")
		groupName := strings.Join(groupNames, " / ")
		if token.Group == "auto" {
			groupID = "auto"
			groupName = "Auto · " + groupName
		}
		keyName := strings.TrimSpace(token.Name)
		if keyName == "" {
			keyName = fmt.Sprintf("API Key #%d", token.Id)
		}
		groups = append(groups, canvasGroupResponse{
			ID:        fmt.Sprintf("token-%d", token.Id),
			Name:      fmt.Sprintf("%s · %s", keyName, groupName),
			APIKey:    "sk-" + token.Key,
			KeyID:     token.Id,
			KeyName:   keyName,
			GroupID:   groupID,
			GroupName: groupName,
			Models:    canvasModels,
		})
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": canvasConfigResponse{
			Groups: groups,
		},
	})
}

func canvasVideoMetadata(account *model.VideoAccount, item model.VideoModel) *canvasVideoModelMetadata {
	pricing := item.EffectivePricing()
	return &canvasVideoModelMetadata{
		VideoAccountTokenID:    account.OpaqueKey(),
		Group:                  item.Group,
		Family:                 item.Family(),
		Resolution:             item.Resolution,
		DurationsSeconds:       item.DurationsSeconds,
		Ratios:                 item.Ratios,
		Sizes:                  item.Sizes,
		RatioSizes:             item.RatioSizes(),
		MaxImages:              item.MaxImages,
		MaxVideos:              item.MaxVideos,
		MaxAudios:              item.MaxAudios,
		AudioRequiresImage:     item.AudioRequiresImage,
		RequiresImage:          item.RequiresReferenceImage(),
		SupportsFirstLastFrame: item.SupportsFirstLastFrame,
		PricingMode:            pricing.Mode,
		PricingAmount:          pricing.Amount,
		PricingCurrency:        pricing.Currency,
	}
}

// canvasCreationModelsForTokenGroups lists the creation models one API key may
// use. It merges both model sources the key can reach - the relay channels of
// its groups, plus the dedicated video accounts those groups authorize - and
// applies the key's own model limit to both. Merging them here (rather than
// exposing video accounts as a separate credential-free entry) is what makes a
// video request travel through the caller's own key, and sharing one limit
// check keeps the workspace from offering a model the relay would refuse.
func canvasCreationModelsForTokenGroups(groups []string, token *model.Token) ([]canvasModelResponse, error) {
	models := canvasModelsForTokenGroups(groups, token)
	result := make([]canvasModelResponse, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, modelName := range models {
		seen[strings.ToLower(modelName)] = struct{}{}
		result = append(result, canvasModelResponse{
			Name:       modelName,
			Capability: canvasModelCapability(modelName),
		})
	}

	accounts, err := model.ListUsableVideoAccounts(groups)
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		for _, item := range account.ModelCatalog() {
			name := strings.TrimSpace(item.ID)
			if !item.Available || name == "" || !canvasTokenAllowsModel(token, name) {
				continue
			}
			key := strings.ToLower(name)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, canvasModelResponse{
				Name:       name,
				Capability: "video",
				Video:      canvasVideoMetadata(account, item),
			})
		}
	}
	return result, nil
}

func canvasModelCapability(modelName string) string {
	for _, endpointType := range model.GetModelSupportEndpointTypes(modelName) {
		switch endpointType {
		case constant.EndpointTypeImageGeneration:
			return "image"
		case constant.EndpointTypeOpenAIVideo:
			return "video"
		}
	}
	if common.IsImageGenerationModel(modelName) {
		return "image"
	}
	if common.IsVideoGenerationModel(modelName) {
		return "video"
	}
	return "text"
}

func canvasModelsForGroup(group string) []string {
	models := model.GetGroupEnabledModels(group)
	result := make([]string, 0, len(models))
	for _, modelName := range models {
		isCreationModel := common.IsImageGenerationModel(modelName) || common.IsVideoGenerationModel(modelName)
		for _, endpointType := range model.GetModelSupportEndpointTypes(modelName) {
			if endpointType == constant.EndpointTypeImageGeneration || endpointType == constant.EndpointTypeOpenAIVideo {
				isCreationModel = true
				break
			}
		}
		if isCreationModel {
			result = append(result, modelName)
		}
	}
	sort.Strings(result)
	return result
}

func canvasModelsForTokenGroup(group string, token *model.Token) []string {
	models := canvasModelsForGroup(group)
	filtered := make([]string, 0, len(models))
	for _, modelName := range models {
		if canvasTokenAllowsModel(token, modelName) {
			filtered = append(filtered, modelName)
		}
	}
	return filtered
}

// canvasTokenAllowsModel reports whether the key's own model limit permits the
// model. A key without limits allows everything.
func canvasTokenAllowsModel(token *model.Token, modelName string) bool {
	if token == nil || !token.ModelLimitsEnabled {
		return true
	}
	for _, limit := range token.GetModelLimits() {
		if strings.EqualFold(strings.TrimSpace(limit), strings.TrimSpace(modelName)) {
			return true
		}
	}
	return false
}

func canvasModelsForTokenGroups(groups []string, token *model.Token) []string {
	models := make([]string, 0)
	seen := make(map[string]struct{})
	for _, group := range groups {
		for _, modelName := range canvasModelsForTokenGroup(group, token) {
			if _, ok := seen[modelName]; ok {
				continue
			}
			seen[modelName] = struct{}{}
			models = append(models, modelName)
		}
	}
	sort.Strings(models)
	return models
}

func canvasTokenGroups(userGroup string, token *model.Token) ([]string, error) {
	if token == nil {
		return nil, nil
	}
	if token.Group == "auto" {
		groups, err := token.GetAutoGroups()
		if err != nil {
			return nil, err
		}
		if len(groups) == 0 {
			return service.GetUserAutoGroup(userGroup), nil
		}
		return service.FilterUserTokenAutoGroups(userGroup, groups), nil
	}
	groupID := strings.TrimSpace(token.Group)
	if groupID == "" {
		groupID = userGroup
	}
	if !service.IsUserSelectableGroup(userGroup, groupID) {
		return nil, nil
	}
	return []string{groupID}, nil
}
