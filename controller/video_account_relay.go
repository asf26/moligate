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
	info.VideoAccount = service.VideoAccountRelayMeta(account)
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
	task.PrivateData.SubscriptionResourceKey = info.SubscriptionResourceKey
	task.PrivateData.SubscriptionResourceType = info.SubscriptionResourceType
	task.PrivateData.SubscriptionResourceAmount = info.SubscriptionResourceAmount
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
		FixedGroupRatio: info.BillingSource == service.BillingSourceSubscription && info.SubscriptionBillingRatio > 0,
	}
	task.Quota = result.Quota
	task.Data = result.TaskData
	task.Action = info.Action
	if insertErr := task.Insert(); insertErr != nil {
		common.SysError("insert video account task error: " + insertErr.Error())
	}
}
