package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// RelayVideoAccountTask submits a task through the dedicated CTMOAI video
// account path. It intentionally has no generic channel selection/retry loop:
// the account was selected by VideoAccountDistribute and its credentials are
// kept on the server.
func RelayVideoAccountTask(c *gin.Context, info *relaycommon.RelayInfo) {
	accountID := common.GetContextKeyInt(c, constant.ContextKeyVideoAccountId)
	account, err := model.GetVideoAccountById(accountID)
	if err != nil || account == nil || !account.IsEnabled() {
		respondTaskError(c, service.TaskErrorWrapperLocal(fmt.Errorf("video account is unavailable"), "video_account_unavailable", http.StatusServiceUnavailable))
		return
	}
	info.VideoAccount = videoAccountRelayMeta(account)
	if info.Action == "" {
		info.Action = constant.TaskActionTextGenerate
	}

	var result *relay.TaskSubmitResult
	var taskErr *dto.TaskError
	defer func() {
		if taskErr != nil && info.Billing != nil {
			info.Billing.Refund(c)
		}
	}()

	result, taskErr = relay.RelayTaskSubmit(c, info)
	if taskErr != nil {
		respondTaskError(c, taskErr)
		return
	}
	if settleErr := service.SettleBilling(c, info, result.Quota); settleErr != nil {
		common.SysError("settle video account task billing error: " + settleErr.Error())
	}
	service.LogTaskConsumption(c, info)

	task := model.InitTask(result.Platform, info)
	task.VideoAccountId = account.Id
	task.PrivateData.VideoAccountId = account.Id
	task.PrivateData.UpstreamTaskID = result.UpstreamTaskID
	task.PrivateData.BillingSource = info.BillingSource
	task.PrivateData.SubscriptionId = info.SubscriptionId
	task.PrivateData.TokenId = info.TokenId
	task.PrivateData.NodeName = common.NodeName
	pricing := info.VideoAccount.Models[info.OriginModelName]
	task.PrivateData.BillingContext = &model.TaskBillingContext{
		ModelPrice:      info.PriceData.ModelPrice,
		GroupRatio:      info.PriceData.GroupRatioInfo.GroupRatio,
		ModelRatio:      info.PriceData.ModelRatio,
		OtherRatios:     info.PriceData.OtherRatios(),
		OriginModelName: info.OriginModelName,
		PerCallBilling:  !strings.EqualFold(strings.TrimSpace(pricing.PricingMode), "per_second"),
		PricingMode:     pricing.PricingMode,
		PricingCurrency: pricing.PricingCurrency,
	}
	task.Quota = result.Quota
	task.Data = result.TaskData
	task.Action = info.Action
	if insertErr := task.Insert(); insertErr != nil {
		common.SysError("insert video account task error: " + insertErr.Error())
	}
}

func videoAccountRelayMeta(account *model.VideoAccount) *relaycommon.VideoAccountMeta {
	models := make(map[string]relaycommon.VideoAccountModelMeta)
	for _, item := range account.ModelCatalog() {
		models[item.ID] = relaycommon.VideoAccountModelMeta{
			ID:                     item.ID,
			DisplayName:            item.DisplayName,
			Group:                  item.Group,
			Available:              item.Available,
			SupportedEndpointTypes: item.SupportedEndpointTypes,
			Resolution:             item.Resolution,
			DurationsSeconds:       item.DurationsSeconds,
			Ratios:                 item.Ratios,
			Sizes:                  item.Sizes,
			MaxImages:              item.MaxImages,
			MaxVideos:              item.MaxVideos,
			MaxAudios:              item.MaxAudios,
			AudioRequiresImage:     item.AudioRequiresImage,
			SupportsFirstLastFrame: item.SupportsFirstLastFrame,
			PricingMode:            item.Pricing.Mode,
			PricingAmount:          item.Pricing.Amount,
			PricingCurrency:        item.Pricing.Currency,
			GroupRatio:             item.GroupRatio,
		}
	}
	return &relaycommon.VideoAccountMeta{
		ID:      account.Id,
		Name:    account.Name,
		Token:   account.OpaqueKey(),
		BaseURL: account.BaseURL(),
		APIKey:  account.ApiKey,
		Proxy:   account.Proxy,
		Models:  models,
	}
}
