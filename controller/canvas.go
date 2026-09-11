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
	Name       string `json:"name"`
	Capability string `json:"capability"`
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
		models := canvasModelsForTokenGroups(tokenGroups, token)
		canvasModels := make([]canvasModelResponse, 0, len(models))
		for _, modelName := range models {
			canvasModels = append(canvasModels, canvasModelResponse{
				Name:       modelName,
				Capability: canvasModelCapability(modelName),
			})
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
	if token == nil || !token.ModelLimitsEnabled {
		return models
	}

	allowed := make(map[string]struct{}, len(token.GetModelLimits()))
	for _, modelName := range token.GetModelLimits() {
		name := strings.TrimSpace(modelName)
		if name != "" {
			allowed[strings.ToLower(name)] = struct{}{}
		}
	}
	filtered := make([]string, 0, len(models))
	for _, modelName := range models {
		if _, ok := allowed[strings.ToLower(modelName)]; ok {
			filtered = append(filtered, modelName)
		}
	}
	return filtered
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
