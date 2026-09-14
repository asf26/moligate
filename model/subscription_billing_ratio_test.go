package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreateUserSubscriptionSnapshotsFixedBillingRatio(t *testing.T) {
	truncateTables(t)

	plan := &SubscriptionPlan{
		Id:            9871,
		Title:         "Kiro fixed ratio",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		ModelFamily:   "kiro-claude",
		BillingRatio:  0,
		TotalAmount:   1000,
	}
	require.NoError(t, DB.Create(plan).Error)

	var sub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		sub, err = CreateUserSubscriptionFromPlanTx(tx, 9872, plan, "test")
		return err
	}))
	require.NotNil(t, sub)
	assert.InDelta(t, 0.17, sub.BillingRatio, 0.000001)

	plan.BillingRatio = 0.23
	assert.InDelta(t, 0.17, sub.BillingRatio, 0.000001,
		"editing a plan must not change a purchased subscription snapshot")
}

func TestCreateUserSubscriptionRejectsPlanWithoutFixedBillingRatio(t *testing.T) {
	truncateTables(t)
	plan := &SubscriptionPlan{
		Id: 9904, Title: "Invalid custom package", DurationUnit: SubscriptionDurationMonth,
		DurationValue: 1, ModelFamily: "custom", BillingRatio: 0, TotalAmount: 1_000,
	}
	require.NoError(t, DB.Create(plan).Error)

	err := DB.Transaction(func(tx *gorm.DB) error {
		_, createErr := CreateUserSubscriptionFromPlanTx(tx, 9905, plan, "test")
		return createErr
	})
	require.ErrorContains(t, err, "no valid fixed billing ratio")

	now := GetDBTimestamp()
	require.NoError(t, DB.Create(&UserSubscription{
		Id: 9906, UserId: 9905, PlanId: plan.Id, ModelFamily: "all",
		BillingRatio: 0, AmountTotal: 1_000, Status: "active",
		StartTime: now - 60, EndTime: now + 3600,
	}).Error)
	_, err = PreConsumeUserSubscriptionWithResourceAndPricing(
		"invalid-zero-ratio-history", 9905, "custom-model", 0, 10, nil,
		&SubscriptionQuotaPricing{LiveQuota: 10},
	)
	require.ErrorContains(t, err, "subscription quota insufficient")
}

func TestGetActiveSubscriptionBillingRatioMatchesSnapshotAndScope(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()

	plan := &SubscriptionPlan{
		Id:            9873,
		Title:         "Scoped fixed ratio",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		ModelFamily:   "gpt",
		BillingRatio:  0.14,
		TotalAmount:   1000,
	}
	require.NoError(t, DB.Create(plan).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:               9874,
		UserId:           9875,
		PlanId:           plan.Id,
		ModelFamily:      "gpt",
		IncludedModels:   []string{"gpt-5.5"},
		ApplicableGroups: []string{"plus/team"},
		BillingRatio:     0.14,
		AmountTotal:      1000,
		StartTime:        now - 60,
		EndTime:          now + 3600,
		Status:           "active",
	}).Error)

	ratio, ok, err := GetActiveSubscriptionBillingRatio(9875, "gpt-5.5", "plus/team")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.InDelta(t, 0.14, ratio, 0.000001)

	_, ok, err = GetActiveSubscriptionBillingRatio(9875, "gpt-5.5", "pro号池")
	require.NoError(t, err)
	assert.False(t, ok)
	_, ok, err = GetActiveSubscriptionBillingRatio(9875, "claude-sonnet-4-6", "plus/team")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestSubscriptionFamilyFallbackDoesNotCrossIntoAnotherProduct(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	for _, sub := range []UserSubscription{
		{Id: 9901, UserId: 9901, ModelFamily: "gpt", BillingRatio: 0.14, AmountTotal: 1_000, StartTime: now - 60, EndTime: now + 3600, Status: "active"},
		{Id: 9902, UserId: 9902, ModelFamily: "gemini", BillingRatio: 0.4, AmountTotal: 1_000, StartTime: now - 60, EndTime: now + 3600, Status: "active"},
		{Id: 9903, UserId: 9903, ModelFamily: "chinese", BillingRatio: 0.35, AmountTotal: 1_000, StartTime: now - 60, EndTime: now + 3600, Status: "active"},
	} {
		require.NoError(t, DB.Create(&sub).Error)
	}

	_, ok, err := GetActiveSubscriptionBillingRatio(9901, "gpt-image-2")
	require.NoError(t, err)
	assert.False(t, ok, "the GPT text package must not fund GPT Image")

	_, ok, err = GetActiveSubscriptionBillingRatio(9902, "gemini-3-pro-image-preview")
	require.NoError(t, err)
	assert.False(t, ok, "the Gemini text package must not fund Banana image generation")

	_, ok, err = GetActiveSubscriptionBillingRatio(9903, "qwen3.8-max")
	require.NoError(t, err)
	assert.False(t, ok, "the domestic package is limited to the advertised model families")
	ratio, ok, err := GetActiveSubscriptionBillingRatio(9903, "glm-5.3")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.InDelta(t, 0.35, ratio, 0.000001)
}

func TestPreConsumeReturnsRatioFromActuallySelectedSubscription(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	firstPlan := &SubscriptionPlan{
		Id:            9876,
		Title:         "Exhausted first package",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		ModelFamily:   "gpt",
		BillingRatio:  0.14,
		TotalAmount:   5,
	}
	secondPlan := &SubscriptionPlan{
		Id:            9877,
		Title:         "Fallback package",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		ModelFamily:   "gpt",
		BillingRatio:  0.4,
		TotalAmount:   1_000,
	}
	require.NoError(t, DB.Create(firstPlan).Error)
	require.NoError(t, DB.Create(secondPlan).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id: 9878, UserId: 9879, PlanId: firstPlan.Id,
		AmountTotal: 5, AmountUsed: 5,
		StartTime: now - 60, EndTime: now + 3600, Status: "active",
		ModelFamily: "gpt", BillingRatio: firstPlan.BillingRatio,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id: 9880, UserId: 9879, PlanId: secondPlan.Id,
		AmountTotal: 1_000, AmountUsed: 0,
		StartTime: now - 60, EndTime: now + 3600, Status: "active",
		ModelFamily: "gpt", BillingRatio: secondPlan.BillingRatio,
	}).Error)

	result, err := PreConsumeUserSubscription("selected-ratio", 9879, "gpt-5.5", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 9880, result.UserSubscriptionId)
	assert.InDelta(t, 0.4, result.BillingRatio, 0.000001)

	// The idempotent retry must return the same selected subscription and its
	// snapshotted ratio, rather than re-quoting the exhausted first package.
	retry, err := PreConsumeUserSubscription("selected-ratio", 9879, "gpt-5.5", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, result.UserSubscriptionId, retry.UserSubscriptionId)
	assert.InDelta(t, result.BillingRatio, retry.BillingRatio, 0.000001)
}

func TestPreConsumePricesEachSubscriptionCandidateWithItsSnapshot(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	firstPlan := &SubscriptionPlan{
		Id: 9885, Title: "Expensive exhausted package", DurationUnit: SubscriptionDurationMonth,
		DurationValue: 1, ModelFamily: "gpt", BillingRatio: 1, TotalAmount: 200,
	}
	secondPlan := &SubscriptionPlan{
		Id: 9886, Title: "Cheaper fallback package", DurationUnit: SubscriptionDurationMonth,
		DurationValue: 1, ModelFamily: "gpt", BillingRatio: 0.5, TotalAmount: 100,
	}
	require.NoError(t, DB.Create(firstPlan).Error)
	require.NoError(t, DB.Create(secondPlan).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id: 9887, UserId: 9888, PlanId: firstPlan.Id, AmountTotal: 200, AmountUsed: 100,
		StartTime: now - 60, EndTime: now + 1800, Status: "active", ModelFamily: "gpt", BillingRatio: 1,
	}).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id: 9889, UserId: 9888, PlanId: secondPlan.Id, AmountTotal: 100,
		StartTime: now - 60, EndTime: now + 3600, Status: "active", ModelFamily: "gpt", BillingRatio: 0.5,
	}).Error)

	result, err := PreConsumeUserSubscriptionWithResourceAndPricing(
		"candidate-specific-ratio", 9888, "gpt-5.5", 0, 200, nil,
		&SubscriptionQuotaPricing{BaseQuotaBeforeGroup: 200, LiveQuota: 200},
	)
	require.NoError(t, err)
	assert.Equal(t, 9889, result.UserSubscriptionId)
	assert.EqualValues(t, 100, result.PreConsumed)
	assert.InDelta(t, 0.5, result.BillingRatio, 0.000001)
}

func TestMigrateActiveSubscriptionBillingRatiosBackfillsLegacyRows(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:            9882,
		Title:         "Legacy Gemini package",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		ModelFamily:   "gemini",
		BillingRatio:  0,
		TotalAmount:   1000,
	}
	require.NoError(t, DB.Create(plan).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		Id:           9883,
		UserId:       9884,
		PlanId:       plan.Id,
		ModelFamily:  "gemini",
		BillingRatio: 0,
		AmountTotal:  1000,
		StartTime:    now - 60,
		EndTime:      now + 3600,
		Status:       "active",
	}).Error)

	require.NoError(t, migrateActiveSubscriptionBillingRatios())
	var migrated UserSubscription
	require.NoError(t, DB.First(&migrated, 9883).Error)
	assert.InDelta(t, 0.4, migrated.BillingRatio, 0.000001)
}
