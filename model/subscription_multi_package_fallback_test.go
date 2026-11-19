package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Two identical active packages: the older one sits at its weekly limit while
// the newer one is untouched. The pre-consume must move on to the newer package
// instead of failing, which would drop the request onto the wallet.
func TestPreConsumeFallsBackToTheSecondIdenticalPackage(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	groups := []string{"plus/team", "pro号池"}
	modelList := []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.5", "gpt-5.4", "gpt-5.4-mini",
		"gpt-5.3-codex-spark", "gpt-5.6", "gpt-6-astra", "gpt-6-luna", "gpt-6-sol", "gpt-6.1-sol"}

	plan := &SubscriptionPlan{
		Id: 9020, Title: "GPT 入门版", PriceAmount: 42, Currency: "CNY",
		DurationUnit: SubscriptionDurationMonth, DurationValue: 1,
		TotalAmount: 150000000, DailyAmount: 15000000, WeeklyAmount: 52500000, MonthlyAmount: 150000000,
		BillingRatio: 1, Enabled: true, SaleEnabled: true,
		ApplicableGroups: groups, ModelFamily: "gpt", IncludedModels: modelList,
	}
	require.NoError(t, DB.Create(plan).Error)

	first := &UserSubscription{
		Id: 9055, UserId: 9015, PlanId: plan.Id,
		AmountTotal: 150000000, AmountUsed: 104717358, NextResetTime: now + 20*86400,
		DailyAmount: 15000000, DailyUsed: 2467203, DailyResetTime: now + 12*3600,
		WeeklyAmount: 52500000, WeeklyUsed: 52268719, WeeklyResetTime: now + 2*86400,
		MonthlyAmount: 150000000, MonthlyUsed: 104717358, MonthlyResetTime: now + 20*86400,
		StartTime: now - 12*86400, EndTime: now + 18*86400, Status: "active",
		ApplicableGroups: groups, ModelFamily: "gpt", IncludedModels: modelList,
		AllowWalletOverflow: true, BillingRatio: 1,
	}
	require.NoError(t, DB.Create(first).Error)

	second := &UserSubscription{
		Id: 9058, UserId: 9015, PlanId: plan.Id,
		AmountTotal: 150000000, NextResetTime: now + 29*86400,
		DailyAmount: 15000000, DailyResetTime: now + 23*3600,
		WeeklyAmount: 52500000, WeeklyResetTime: now + 6*86400,
		MonthlyAmount: 150000000, MonthlyResetTime: now + 29*86400,
		StartTime: now - 3600, EndTime: now + 29*86400, Status: "active",
		ApplicableGroups: groups, ModelFamily: "gpt", IncludedModels: modelList,
		AllowWalletOverflow: true, BillingRatio: 1,
	}
	require.NoError(t, DB.Create(second).Error)

	res, err := PreConsumeUserSubscriptionWithResourceAndPricing("req-repro-1", 9015, "gpt-6-astra", 0, 500000, nil,
		&SubscriptionQuotaPricing{BaseQuotaBeforeGroup: 0, LiveQuota: 500000}, "pro号池")
	t.Logf("结果: %+v 错误: %v", res, err)
	require.NoError(t, err)
	assert.Equal(t, second.Id, res.UserSubscriptionId, "应该用未使用的新套餐")
}
