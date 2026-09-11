package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	relaydto "github.com/QuantumNous/new-api/relaykit/dto"
	relaytypes "github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// VideoAccountDistribute selects a dedicated VideoAccount for OpenAI Video
// requests. When no dedicated account exposes the requested model, it falls
// back to the legacy generic-channel distributor so existing Sora/OpenAI
// channel integrations keep working. An explicit token-id header is strict:
// an invalid or unavailable account must never silently fall through to a
// different credential.
func VideoAccountDistribute() func(c *gin.Context) {
	return func(c *gin.Context) {
		modelRequest, shouldSelect, err := getModelRequest(c)
		if err != nil {
			abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidRequest, map[string]any{"Error": err.Error()}))
			return
		}
		if !shouldSelect {
			c.Next()
			return
		}
		modelName := strings.TrimSpace(modelRequest.Model)
		if modelName == "" {
			abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorModelNameRequired))
			return
		}
		if !videoTokenModelAllowed(c, modelName) {
			abortWithOpenAiMessage(c, http.StatusForbidden, i18n.T(c, i18n.MsgDistributorTokenModelForbidden, map[string]any{"Model": modelName}))
			return
		}
		group := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
		if group == "" {
			group = common.GetContextKeyString(c, constant.ContextKeyUserGroup)
		}
		publicKey := strings.TrimSpace(c.GetHeader("X-Video-Creation-Token-Id"))
		account, selected, err := selectVideoAccount(modelName, group, publicKey)
		if err != nil {
			abortWithOpenAiMessage(c, http.StatusServiceUnavailable, err.Error(), relaytypes.ErrorCodeModelNotFound)
			return
		}
		if !selected {
			// Keep the old /v1/videos contract available for installations that
			// have not configured a dedicated CTMOAI account for this model.
			Distribute()(c)
			return
		}
		setVideoAccountContext(c, account)
		common.SetContextKey(c, constant.ContextKeyOriginalModel, modelName)
		c.Set("platform", string(constant.TaskPlatformVideoCTMoai))
		c.Next()
	}
}

// selectVideoAccount distinguishes "no dedicated account matches" from a
// lookup failure. That distinction is what lets the caller preserve legacy
// generic-channel behavior without hiding database or explicit-selector
// errors.
func selectVideoAccount(modelName, group, publicKey string) (*model.VideoAccount, bool, error) {
	if publicKey != "" {
		account, err := model.FindVideoAccountForModel(modelName, group, publicKey)
		return account, true, err
	}
	accounts, err := model.ListEnabledVideoAccounts(group)
	if err != nil {
		return nil, false, err
	}
	for _, account := range accounts {
		if item, ok := account.FindModel(modelName); ok && item.Available {
			return account, true, nil
		}
	}
	return nil, false, nil
}

func videoTokenModelAllowed(c *gin.Context, modelName string) bool {
	if !common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled) {
		return true
	}
	value, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
	if !ok {
		return false
	}
	limits, ok := value.(map[string]bool)
	if !ok {
		return false
	}
	_, ok = limits[ratio_setting.FormatMatchingModelName(modelName)]
	return ok
}

func setVideoAccountContext(c *gin.Context, account *model.VideoAccount) {
	if account == nil {
		return
	}
	common.SetContextKey(c, constant.ContextKeyVideoAccountId, account.Id)
	common.SetContextKey(c, constant.ContextKeyVideoAccountToken, account.OpaqueKey())
	common.SetContextKey(c, constant.ContextKeyChannelName, account.Name)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, account.BaseURL())
	common.SetContextKey(c, constant.ContextKeyChannelKey, account.ApiKey)
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeSora)
	common.SetContextKey(c, constant.ContextKeyChannelSetting, relaydto.ChannelSettings{Proxy: account.Proxy})
	common.SetContextKey(c, constant.ContextKeyChannelOtherSetting, relaydto.ChannelOtherSettings{})
}
