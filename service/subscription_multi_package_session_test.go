package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A package at its weekly cap must not push the request onto the wallet while
// another active package still has room.
func TestBillingSessionSkipsTheExhaustedPackage(t *testing.T) {
	truncate(t)
	now := time.Now().Unix()
	const userID = 9915
	seedUser(t, userID, 100_000)

	groups := []string{"plus/team", "pro号池"}
	modelList := []string{"gpt-6-astra", "gpt-6-sol"}

	plan := &model.SubscriptionPlan{
		Id: 9920, Title: "GPT 入门版", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1,
		TotalAmount: 150_000_000, DailyAmount: 15_000_000, WeeklyAmount: 52_500_000, MonthlyAmount: 150_000_000,
		BillingRatio: 1, Enabled: true, ApplicableGroups: groups, ModelFamily: "gpt", IncludedModels: modelList,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	require.NoError(t, model.DB.Create(&model.UserSubscription{
		Id: 9955, UserId: userID, PlanId: plan.Id,
		AmountTotal: 150_000_000, AmountUsed: 104_717_358, NextResetTime: now + 20*86400,
		DailyAmount: 15_000_000, DailyUsed: 2_467_203, DailyResetTime: now + 12*3600,
		WeeklyAmount: 52_500_000, WeeklyUsed: 52_268_719, WeeklyResetTime: now + 2*86400,
		MonthlyAmount: 150_000_000, MonthlyUsed: 104_717_358, MonthlyResetTime: now + 20*86400,
		StartTime: now - 12*86400, EndTime: now + 18*86400, Status: "active",
		ApplicableGroups: groups, ModelFamily: "gpt", IncludedModels: modelList,
		AllowWalletOverflow: true, BillingRatio: 1,
	}).Error)

	require.NoError(t, model.DB.Create(&model.UserSubscription{
		Id: 9958, UserId: userID, PlanId: plan.Id,
		AmountTotal: 150_000_000, NextResetTime: now + 29*86400,
		DailyAmount: 15_000_000, DailyResetTime: now + 23*3600,
		WeeklyAmount: 52_500_000, WeeklyResetTime: now + 6*86400,
		MonthlyAmount: 150_000_000, MonthlyResetTime: now + 29*86400,
		StartTime: now - 3600, EndTime: now + 29*86400, Status: "active",
		ApplicableGroups: groups, ModelFamily: "gpt", IncludedModels: modelList,
		AllowWalletOverflow: true, BillingRatio: 1,
	}).Error)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		RequestId:       "repro-weekly-cap",
		UserId:          userID,
		OriginModelName: "gpt-6-astra",
		UsingGroup:      "pro号池",
		IsPlayground:    true,
		UserSetting:     dto.UserSetting{},
		PriceData: types.PriceData{
			GroupRatioInfo:       types.GroupRatioInfo{GroupRatio: 0.2},
			BaseQuotaBeforeGroup: 500_000,
		},
	}

	session, apiErr := NewBillingSession(ctx, info, 500_000)
	require.Nil(t, apiErr)
	require.NotNil(t, session)
	t.Logf("资金来源=%s 订阅=%d 预扣=%d", session.funding.Source(), info.SubscriptionId, session.GetPreConsumedQuota())
	assert.Equal(t, BillingSourceSubscription, session.funding.Source(), "不应落到钱包")
	assert.Equal(t, 9958, info.SubscriptionId, "应使用未使用的新套餐")
}
