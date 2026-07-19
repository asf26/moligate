package model

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreateUserSubscriptionSnapshotsAllPeriodQuotas(t *testing.T) {
	truncateTables(t)

	plan := &SubscriptionPlan{
		Id:               9701,
		Title:            "Three Periods",
		PriceAmount:      100,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    2,
		TotalAmount:      10_000,
		DailyAmount:      100,
		WeeklyAmount:     500,
		MonthlyAmount:    2_000,
		QuotaResetPeriod: SubscriptionResetNever,
	}
	require.NoError(t, DB.Create(plan).Error)

	var subscription *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		subscription, err = CreateUserSubscriptionFromPlanTx(tx, 501, plan, "admin")
		return err
	}))
	require.NotNil(t, subscription)

	assert.EqualValues(t, plan.TotalAmount, subscription.AmountTotal)
	assert.EqualValues(t, plan.DailyAmount, subscription.DailyAmount)
	assert.EqualValues(t, plan.WeeklyAmount, subscription.WeeklyAmount)
	assert.EqualValues(t, plan.MonthlyAmount, subscription.MonthlyAmount)
	assert.Zero(t, subscription.DailyUsed)
	assert.Zero(t, subscription.WeeklyUsed)
	assert.Zero(t, subscription.MonthlyUsed)

	start := time.Unix(subscription.StartTime, 0)
	assert.Equal(t, calcNextCalendarReset(start, SubscriptionResetDaily, subscription.EndTime), subscription.DailyResetTime)
	assert.Equal(t, calcNextCalendarReset(start, SubscriptionResetWeekly, subscription.EndTime), subscription.WeeklyResetTime)
	assert.Equal(t, calcNextCalendarReset(start, SubscriptionResetMonthly, subscription.EndTime), subscription.MonthlyResetTime)
}

func TestPreConsumeAppliesAllPeriodQuotasAndRejectsWhenAnyIsExhausted(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:               9711,
		Title:            "Concurrent Limits",
		PriceAmount:      100,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		TotalAmount:      1_000,
		DailyAmount:      100,
		WeeklyAmount:     500,
		MonthlyAmount:    900,
		QuotaResetPeriod: SubscriptionResetNever,
	}
	require.NoError(t, DB.Create(plan).Error)
	subscription := &UserSubscription{
		Id:               9712,
		UserId:           511,
		PlanId:           plan.Id,
		AmountTotal:      plan.TotalAmount,
		AmountUsed:       40,
		DailyAmount:      plan.DailyAmount,
		DailyUsed:        90,
		DailyResetTime:   now + 3600,
		WeeklyAmount:     plan.WeeklyAmount,
		WeeklyUsed:       100,
		WeeklyResetTime:  now + 7200,
		MonthlyAmount:    plan.MonthlyAmount,
		MonthlyUsed:      200,
		MonthlyResetTime: now + 10800,
		StartTime:        now - 3600,
		EndTime:          now + 30*24*3600,
		Status:           "active",
	}
	require.NoError(t, DB.Create(subscription).Error)

	result, err := PreConsumeUserSubscription("multi-period-success", subscription.UserId, "gpt-test", 0, 10)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, subscription.Id, result.UserSubscriptionId)

	updated := getSubscriptionResetSub(t, subscription.Id)
	assert.EqualValues(t, 50, updated.AmountUsed)
	assert.EqualValues(t, 100, updated.DailyUsed)
	assert.EqualValues(t, 110, updated.WeeklyUsed)
	assert.EqualValues(t, 210, updated.MonthlyUsed)

	result, err = PreConsumeUserSubscription("multi-period-rejected", subscription.UserId, "gpt-test", 0, 1)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, strings.Contains(err.Error(), "subscription quota insufficient"))

	unchanged := getSubscriptionResetSub(t, subscription.Id)
	assert.EqualValues(t, 50, unchanged.AmountUsed)
	assert.EqualValues(t, 100, unchanged.DailyUsed)
	assert.EqualValues(t, 110, unchanged.WeeklyUsed)
	assert.EqualValues(t, 210, unchanged.MonthlyUsed)
}

func TestDueDailyResetKeepsWeeklyAndMonthlyUsage(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:               9721,
		Title:            "Independent Resets",
		PriceAmount:      100,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		DailyAmount:      100,
		WeeklyAmount:     500,
		MonthlyAmount:    2_000,
		QuotaResetPeriod: SubscriptionResetNever,
	}
	require.NoError(t, DB.Create(plan).Error)
	subscription := &UserSubscription{
		Id:               9722,
		UserId:           521,
		PlanId:           plan.Id,
		DailyAmount:      plan.DailyAmount,
		DailyUsed:        80,
		DailyResetTime:   now - 1,
		WeeklyAmount:     plan.WeeklyAmount,
		WeeklyUsed:       320,
		WeeklyResetTime:  now + 2*24*3600,
		MonthlyAmount:    plan.MonthlyAmount,
		MonthlyUsed:      700,
		MonthlyResetTime: now + 10*24*3600,
		StartTime:        now - 2*24*3600,
		EndTime:          now + 20*24*3600,
		Status:           "active",
	}
	require.NoError(t, DB.Create(subscription).Error)

	resetCount, err := ResetDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, resetCount)

	updated := getSubscriptionResetSub(t, subscription.Id)
	assert.Zero(t, updated.DailyUsed)
	assert.Greater(t, updated.DailyResetTime, now)
	assert.EqualValues(t, 320, updated.WeeklyUsed)
	assert.EqualValues(t, subscription.WeeklyResetTime, updated.WeeklyResetTime)
	assert.EqualValues(t, 700, updated.MonthlyUsed)
	assert.EqualValues(t, subscription.MonthlyResetTime, updated.MonthlyResetTime)
}

func TestRefundSubscriptionPreConsumeRestoresAllPeriodUsage(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:               9731,
		Title:            "Refund Limits",
		PriceAmount:      100,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		TotalAmount:      1_000,
		DailyAmount:      100,
		WeeklyAmount:     500,
		MonthlyAmount:    900,
		QuotaResetPeriod: SubscriptionResetNever,
	}
	require.NoError(t, DB.Create(plan).Error)
	subscription := &UserSubscription{
		Id:               9732,
		UserId:           531,
		PlanId:           plan.Id,
		AmountTotal:      plan.TotalAmount,
		DailyAmount:      plan.DailyAmount,
		DailyResetTime:   now + 3600,
		WeeklyAmount:     plan.WeeklyAmount,
		WeeklyResetTime:  now + 7200,
		MonthlyAmount:    plan.MonthlyAmount,
		MonthlyResetTime: now + 10800,
		StartTime:        now - 3600,
		EndTime:          now + 30*24*3600,
		Status:           "active",
	}
	require.NoError(t, DB.Create(subscription).Error)

	_, err := PreConsumeUserSubscription("multi-period-refund", subscription.UserId, "gpt-test", 0, 25)
	require.NoError(t, err)
	require.NoError(t, RefundSubscriptionPreConsume("multi-period-refund"))

	updated := getSubscriptionResetSub(t, subscription.Id)
	assert.Zero(t, updated.AmountUsed)
	assert.Zero(t, updated.DailyUsed)
	assert.Zero(t, updated.WeeklyUsed)
	assert.Zero(t, updated.MonthlyUsed)

	require.NoError(t, RefundSubscriptionPreConsume("multi-period-refund"))
	idempotent := getSubscriptionResetSub(t, subscription.Id)
	assert.Zero(t, idempotent.AmountUsed)
	assert.Zero(t, idempotent.DailyUsed)
	assert.Zero(t, idempotent.WeeklyUsed)
	assert.Zero(t, idempotent.MonthlyUsed)
}

func TestPostConsumeDeltaResetsDuePeriodsBeforeSettlement(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:               9735,
		Title:            "Boundary Settlement",
		PriceAmount:      100,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		TotalAmount:      1_000,
		DailyAmount:      100,
		WeeklyAmount:     500,
		MonthlyAmount:    900,
		QuotaResetPeriod: SubscriptionResetDaily,
	}
	require.NoError(t, DB.Create(plan).Error)
	subscription := &UserSubscription{
		Id:               9736,
		UserId:           536,
		PlanId:           plan.Id,
		AmountTotal:      plan.TotalAmount,
		AmountUsed:       80,
		DailyAmount:      plan.DailyAmount,
		DailyUsed:        70,
		DailyResetTime:   now - 1,
		WeeklyAmount:     plan.WeeklyAmount,
		WeeklyUsed:       100,
		WeeklyResetTime:  now + 2*24*3600,
		MonthlyAmount:    plan.MonthlyAmount,
		MonthlyUsed:      200,
		MonthlyResetTime: now + 10*24*3600,
		StartTime:        now - 2*24*3600,
		EndTime:          now + 20*24*3600,
		Status:           "active",
		LastResetTime:    now - 24*3600,
		NextResetTime:    now - 1,
	}
	require.NoError(t, DB.Create(subscription).Error)

	require.NoError(t, PostConsumeUserSubscriptionDelta(subscription.Id, 10))

	updated := getSubscriptionResetSub(t, subscription.Id)
	assert.EqualValues(t, 10, updated.AmountUsed)
	assert.EqualValues(t, 10, updated.DailyUsed)
	assert.EqualValues(t, 110, updated.WeeklyUsed)
	assert.EqualValues(t, 210, updated.MonthlyUsed)
	assert.Greater(t, updated.NextResetTime, now)
	assert.Greater(t, updated.DailyResetTime, now)
}

func TestPreConsumeRejectsUsageOverflow(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:               9741,
		Title:            "Unlimited Overflow Guard",
		PriceAmount:      100,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		QuotaResetPeriod: SubscriptionResetNever,
	}
	require.NoError(t, DB.Create(plan).Error)
	subscription := &UserSubscription{
		Id:          9742,
		UserId:      541,
		PlanId:      plan.Id,
		AmountUsed:  math.MaxInt64,
		StartTime:   now - 3600,
		EndTime:     now + 30*24*3600,
		Status:      "active",
		AmountTotal: 0,
	}
	require.NoError(t, DB.Create(subscription).Error)

	result, err := PreConsumeUserSubscription("multi-period-overflow", subscription.UserId, "gpt-test", 0, 1)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "subscription quota insufficient")
	assert.EqualValues(t, math.MaxInt64, getSubscriptionResetSub(t, subscription.Id).AmountUsed)
}

func TestPostConsumeDeltaGuardsOverflowAndLargeRefund(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:               9752,
		Title:            "Overflow Guard",
		PriceAmount:      100,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		QuotaResetPeriod: SubscriptionResetNever,
	}
	require.NoError(t, DB.Create(plan).Error)
	subscription := &UserSubscription{
		Id:            9751,
		UserId:        551,
		PlanId:        9752,
		AmountUsed:    math.MaxInt64,
		DailyAmount:   math.MaxInt64,
		DailyUsed:     math.MaxInt64,
		WeeklyAmount:  math.MaxInt64,
		WeeklyUsed:    math.MaxInt64,
		MonthlyAmount: math.MaxInt64,
		MonthlyUsed:   math.MaxInt64,
		StartTime:     now - 3600,
		EndTime:       now + 30*24*3600,
		Status:        "active",
	}
	require.NoError(t, DB.Create(subscription).Error)

	require.Error(t, PostConsumeUserSubscriptionDelta(subscription.Id, 1))
	unchanged := getSubscriptionResetSub(t, subscription.Id)
	assert.EqualValues(t, math.MaxInt64, unchanged.AmountUsed)
	assert.EqualValues(t, math.MaxInt64, unchanged.DailyUsed)
	assert.EqualValues(t, math.MaxInt64, unchanged.WeeklyUsed)
	assert.EqualValues(t, math.MaxInt64, unchanged.MonthlyUsed)

	require.NoError(t, PostConsumeUserSubscriptionDelta(subscription.Id, math.MinInt64))
	refunded := getSubscriptionResetSub(t, subscription.Id)
	assert.Zero(t, refunded.AmountUsed)
	assert.Zero(t, refunded.DailyUsed)
	assert.Zero(t, refunded.WeeklyUsed)
	assert.Zero(t, refunded.MonthlyUsed)
}
