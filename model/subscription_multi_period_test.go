package model

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSubscriptionPeriodResetsStayAnchoredToStartTime(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	start := time.Date(2026, time.July, 20, 10, 30, 0, 0, location)
	end := time.Date(2027, time.July, 20, 10, 30, 0, 0, location).Unix()

	assert.Equal(t,
		time.Date(2026, time.July, 21, 10, 30, 0, 0, location).Unix(),
		calcNextAnchoredPeriodReset(start, start, SubscriptionResetDaily, end),
	)
	assert.Equal(t,
		time.Date(2026, time.July, 27, 10, 30, 0, 0, location).Unix(),
		calcNextAnchoredPeriodReset(start, start, SubscriptionResetWeekly, end),
	)
	assert.Equal(t,
		time.Date(2026, time.August, 20, 10, 30, 0, 0, location).Unix(),
		calcNextAnchoredPeriodReset(start, start, SubscriptionResetMonthly, end),
	)

	januaryEnd := time.Date(2026, time.January, 31, 9, 0, 0, 0, location)
	februaryEnd := time.Date(2026, time.February, 28, 9, 0, 0, 0, location)
	assert.Equal(t,
		februaryEnd.Unix(),
		calcNextAnchoredPeriodReset(januaryEnd, januaryEnd, SubscriptionResetMonthly, end),
	)
	assert.Equal(t,
		time.Date(2026, time.March, 31, 9, 0, 0, 0, location).Unix(),
		calcNextAnchoredPeriodReset(januaryEnd, februaryEnd, SubscriptionResetMonthly, end),
	)
	assert.Equal(t, februaryEnd.Unix(), anchoredPeriodWindowStart(januaryEnd, time.Date(2026, time.March, 15, 9, 0, 0, 0, location), SubscriptionResetMonthly))
}

func TestSubscriptionApplicableGroups(t *testing.T) {
	assert.True(t, subscriptionGroupMatches(nil, "vip"))
	assert.True(t, subscriptionGroupMatches([]string{"vip", "standard"}, "vip"))
	assert.False(t, subscriptionGroupMatches([]string{"vip"}, "standard"))
	assert.Equal(t, []string{"vip", "standard"}, normalizeSubscriptionGroups([]string{" vip ", "vip", "", "standard"}))
}

func TestAdminSubscriptionUsageImportFiltersGroupsAndRequiresOverLimitConfirmation(t *testing.T) {
	truncateTables(t)
	now := time.Unix(GetDBTimestamp(), 0)
	start := now.Add(-36 * time.Hour).Unix()
	plan := &SubscriptionPlan{
		Id: 9751, Title: "Imported Usage", PriceAmount: 10,
		DurationUnit: SubscriptionDurationMonth, DurationValue: 1,
		TotalAmount: 25, DailyAmount: 15, WeeklyAmount: 100, MonthlyAmount: 100,
		ApplicableGroups: []string{"vip"}, QuotaResetPeriod: SubscriptionResetNever,
	}
	require.NoError(t, DB.Create(plan).Error)
	require.NoError(t, DB.Create(&User{Id: 9752, Username: "import-user", Quota: 500}).Error)
	dailyStart := anchoredPeriodWindowStart(time.Unix(start, 0), now, SubscriptionResetDaily)
	logs := []Log{
		{UserId: 9752, CreatedAt: start + 60, Type: LogTypeConsume, Quota: 10, Group: "vip"},
		{UserId: 9752, CreatedAt: dailyStart + 60, Type: LogTypeConsume, Quota: 20, Group: "vip"},
		{UserId: 9752, CreatedAt: dailyStart + 120, Type: LogTypeConsume, Quota: 999, Group: "standard"},
		{UserId: 9752, CreatedAt: dailyStart + 180, Type: LogTypeConsume, Quota: 777, Group: "vip", Other: `{"billing_source":"subscription"}`},
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)

	preview, err := PreviewAdminSubscriptionUsage(9752, plan.Id, start)
	require.NoError(t, err)
	assert.EqualValues(t, 30, preview.AmountUsed)
	assert.EqualValues(t, 20, preview.DailyUsed)
	assert.True(t, preview.ExceedsAmountTotal)
	assert.True(t, preview.ExceedsDailyAmount)
	assert.True(t, preview.ExceedsAnyLimit)

	_, _, _, err = AdminBindSubscriptionWithOptions(9752, plan.Id, AdminSubscriptionBindOptions{
		EffectiveStartTime: start, ImportUsage: true,
	})
	assert.ErrorIs(t, err, ErrSubscriptionImportedUsageOverLimit)

	previousBatchUpdateEnabled := common.BatchUpdateEnabled
	common.BatchUpdateEnabled = true
	t.Cleanup(func() { common.BatchUpdateEnabled = previousBatchUpdateEnabled })
	addNewRecord(BatchUpdateTypeUserQuota, 9752, -100)
	subscription, _, _, err := AdminBindSubscriptionWithOptions(9752, plan.Id, AdminSubscriptionBindOptions{
		EffectiveStartTime: start, ImportUsage: true, ClearWallet: true, ConfirmOverLimit: true,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 30, subscription.AmountUsed)
	assert.EqualValues(t, 20, subscription.DailyUsed)
	assert.Equal(t, []string{"vip"}, subscription.ApplicableGroups)
	assert.Equal(t, start, subscription.StartTime)
	quota, err := GetUserQuota(9752, false)
	require.NoError(t, err)
	assert.Zero(t, quota)
	batchUpdateLocks[BatchUpdateTypeUserQuota].Lock()
	_, hasPendingWalletDelta := batchUpdateStores[BatchUpdateTypeUserQuota][9752]
	batchUpdateLocks[BatchUpdateTypeUserQuota].Unlock()
	assert.False(t, hasPendingWalletDelta)
}

func TestSubscriptionPreConsumeUsesEffectiveGroup(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	plan := &SubscriptionPlan{Id: 9761, Title: "Scoped", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 100}
	require.NoError(t, DB.Create(plan).Error)
	subscription := &UserSubscription{
		Id: 9762, UserId: 9763, PlanId: plan.Id, AmountTotal: 100,
		StartTime: now - 60, EndTime: now + 3600, Status: "active",
		ApplicableGroups: []string{"vip"}, AllowWalletOverflow: false,
	}
	require.NoError(t, DB.Create(subscription).Error)

	hasSubscription, err := HasActiveUserSubscription(subscription.UserId, "standard")
	require.NoError(t, err)
	assert.False(t, hasSubscription)
	allowOverflow, err := UserActiveSubscriptionsAllowWalletOverflow(subscription.UserId, "standard")
	require.NoError(t, err)
	assert.True(t, allowOverflow)
	_, err = PreConsumeUserSubscription("wrong-group", subscription.UserId, "gpt-test", 0, 10, "standard")
	assert.Error(t, err)

	result, err := PreConsumeUserSubscription("matching-group", subscription.UserId, "gpt-test", 0, 10, "vip")
	require.NoError(t, err)
	assert.Equal(t, subscription.Id, result.UserSubscriptionId)
	assert.EqualValues(t, 10, result.PreConsumed)

	require.NoError(t, AdminUpdateUserSubscriptionApplicableGroups(subscription.Id, []string{"standard"}))
	hasSubscription, err = HasActiveUserSubscription(subscription.UserId, "vip")
	require.NoError(t, err)
	assert.False(t, hasSubscription)
	result, err = PreConsumeUserSubscription("updated-group", subscription.UserId, "gpt-test", 0, 10, "standard")
	require.NoError(t, err)
	assert.Equal(t, subscription.Id, result.UserSubscriptionId)
}

func TestMigrateActiveSubscriptionResetAnchorsPreservesUsage(t *testing.T) {
	truncateTables(t)

	location := time.Local
	start := time.Date(2026, time.July, 20, 10, 30, 0, 0, location)
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, location)
	end := time.Date(2027, time.July, 20, 10, 30, 0, 0, location)
	plan := &SubscriptionPlan{
		Id:               9691,
		Title:            "Legacy Calendar Reset",
		PriceAmount:      100,
		DurationUnit:     SubscriptionDurationYear,
		DurationValue:    1,
		TotalAmount:      10_000,
		DailyAmount:      400,
		WeeklyAmount:     1_400,
		MonthlyAmount:    6_000,
		QuotaResetPeriod: SubscriptionResetMonthly,
	}
	require.NoError(t, DB.Create(plan).Error)
	subscription := &UserSubscription{
		Id:               9692,
		UserId:           491,
		PlanId:           plan.Id,
		AmountTotal:      plan.TotalAmount,
		AmountUsed:       800,
		DailyAmount:      plan.DailyAmount,
		DailyUsed:        80,
		DailyResetTime:   time.Date(2026, time.July, 26, 0, 0, 0, 0, location).Unix(),
		WeeklyAmount:     plan.WeeklyAmount,
		WeeklyUsed:       320,
		WeeklyResetTime:  time.Date(2026, time.July, 27, 0, 0, 0, 0, location).Unix(),
		MonthlyAmount:    plan.MonthlyAmount,
		MonthlyUsed:      700,
		MonthlyResetTime: time.Date(2026, time.August, 1, 0, 0, 0, 0, location).Unix(),
		StartTime:        start.Unix(),
		EndTime:          end.Unix(),
		Status:           "active",
		NextResetTime:    time.Date(2026, time.August, 1, 0, 0, 0, 0, location).Unix(),
	}
	require.NoError(t, DB.Create(subscription).Error)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return migrateUserSubscriptionResetAnchorTx(tx, subscription, plan, now.Unix())
	}))

	updated := getSubscriptionResetSub(t, subscription.Id)
	assert.EqualValues(t, 800, updated.AmountUsed)
	assert.EqualValues(t, 80, updated.DailyUsed)
	assert.EqualValues(t, 320, updated.WeeklyUsed)
	assert.EqualValues(t, 700, updated.MonthlyUsed)
	assert.Equal(t, time.Date(2026, time.July, 26, 10, 30, 0, 0, location).Unix(), updated.DailyResetTime)
	assert.Equal(t, time.Date(2026, time.July, 27, 10, 30, 0, 0, location).Unix(), updated.WeeklyResetTime)
	assert.Equal(t, time.Date(2026, time.August, 20, 10, 30, 0, 0, location).Unix(), updated.MonthlyResetTime)
	assert.Equal(t, updated.MonthlyResetTime, updated.NextResetTime)
	assert.Equal(t, subscriptionPeriodResetAnchorVersion, updated.PeriodResetAnchorVersion)
}

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
	assert.Equal(t, calcNextAnchoredPeriodReset(start, start, SubscriptionResetDaily, subscription.EndTime), subscription.DailyResetTime)
	assert.Equal(t, calcNextAnchoredPeriodReset(start, start, SubscriptionResetWeekly, subscription.EndTime), subscription.WeeklyResetTime)
	assert.Equal(t, calcNextAnchoredPeriodReset(start, start, SubscriptionResetMonthly, subscription.EndTime), subscription.MonthlyResetTime)
	assert.Equal(t, subscriptionPeriodResetAnchorVersion, subscription.PeriodResetAnchorVersion)
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
		Id:                       9712,
		UserId:                   511,
		PlanId:                   plan.Id,
		AmountTotal:              plan.TotalAmount,
		AmountUsed:               40,
		DailyAmount:              plan.DailyAmount,
		DailyUsed:                90,
		DailyResetTime:           now + 3600,
		WeeklyAmount:             plan.WeeklyAmount,
		WeeklyUsed:               100,
		WeeklyResetTime:          now + 7200,
		MonthlyAmount:            plan.MonthlyAmount,
		MonthlyUsed:              200,
		MonthlyResetTime:         now + 10800,
		StartTime:                now - 3600,
		EndTime:                  now + 30*24*3600,
		Status:                   "active",
		PeriodResetAnchorVersion: subscriptionPeriodResetAnchorVersion,
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
		Id:                       9722,
		UserId:                   521,
		PlanId:                   plan.Id,
		DailyAmount:              plan.DailyAmount,
		DailyUsed:                80,
		DailyResetTime:           now - 1,
		WeeklyAmount:             plan.WeeklyAmount,
		WeeklyUsed:               320,
		WeeklyResetTime:          now + 2*24*3600,
		MonthlyAmount:            plan.MonthlyAmount,
		MonthlyUsed:              700,
		MonthlyResetTime:         now + 10*24*3600,
		StartTime:                now - 2*24*3600,
		EndTime:                  now + 20*24*3600,
		Status:                   "active",
		PeriodResetAnchorVersion: subscriptionPeriodResetAnchorVersion,
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
		Id:                       9732,
		UserId:                   531,
		PlanId:                   plan.Id,
		AmountTotal:              plan.TotalAmount,
		DailyAmount:              plan.DailyAmount,
		DailyResetTime:           now + 3600,
		WeeklyAmount:             plan.WeeklyAmount,
		WeeklyResetTime:          now + 7200,
		MonthlyAmount:            plan.MonthlyAmount,
		MonthlyResetTime:         now + 10800,
		StartTime:                now - 3600,
		EndTime:                  now + 30*24*3600,
		Status:                   "active",
		PeriodResetAnchorVersion: subscriptionPeriodResetAnchorVersion,
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
		Id:                       9736,
		UserId:                   536,
		PlanId:                   plan.Id,
		AmountTotal:              plan.TotalAmount,
		AmountUsed:               80,
		DailyAmount:              plan.DailyAmount,
		DailyUsed:                70,
		DailyResetTime:           now - 1,
		WeeklyAmount:             plan.WeeklyAmount,
		WeeklyUsed:               100,
		WeeklyResetTime:          now + 2*24*3600,
		MonthlyAmount:            plan.MonthlyAmount,
		MonthlyUsed:              200,
		MonthlyResetTime:         now + 10*24*3600,
		StartTime:                now - 2*24*3600,
		EndTime:                  now + 20*24*3600,
		Status:                   "active",
		LastResetTime:            now - 24*3600,
		NextResetTime:            now - 1,
		PeriodResetAnchorVersion: subscriptionPeriodResetAnchorVersion,
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
		Id:                       9742,
		UserId:                   541,
		PlanId:                   plan.Id,
		AmountUsed:               math.MaxInt64,
		StartTime:                now - 3600,
		EndTime:                  now + 30*24*3600,
		Status:                   "active",
		AmountTotal:              0,
		PeriodResetAnchorVersion: subscriptionPeriodResetAnchorVersion,
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
		Id:                       9751,
		UserId:                   551,
		PlanId:                   9752,
		AmountUsed:               math.MaxInt64,
		DailyAmount:              math.MaxInt64,
		DailyUsed:                math.MaxInt64,
		WeeklyAmount:             math.MaxInt64,
		WeeklyUsed:               math.MaxInt64,
		MonthlyAmount:            math.MaxInt64,
		MonthlyUsed:              math.MaxInt64,
		StartTime:                now - 3600,
		EndTime:                  now + 30*24*3600,
		Status:                   "active",
		PeriodResetAnchorVersion: subscriptionPeriodResetAnchorVersion,
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
