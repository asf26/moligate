package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionBonusResourcesPreConsumeAndRefund(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id:             9851,
		Title:          "GPT with extras",
		DurationUnit:   SubscriptionDurationMonth,
		DurationValue:  1,
		ModelFamily:    "gpt",
		IncludedModels: []string{"gpt-5.5"},
		BillingRatio:   0.14,
		TotalAmount:    1000,
		BonusResources: []SubscriptionBonusResource{
			{ResourceKey: "grok", ResourceType: SubscriptionResourceTypeQuota, ModelName: "grok", Amount: 100},
			{ResourceKey: "gpt-image-2", ResourceType: SubscriptionResourceTypeImageCount, ModelName: "gpt-image-2", Amount: 3},
		},
	}
	require.NoError(t, DB.Create(plan).Error)
	subscription := &UserSubscription{
		Id: 9852, UserId: 9853, PlanId: plan.Id, AmountTotal: plan.TotalAmount,
		ModelFamily: "gpt", IncludedModels: []string{"gpt-5.5"}, BillingRatio: 0.14,
		ResourceGrants: []SubscriptionResourceGrant{
			{ResourceKey: "grok", ResourceType: SubscriptionResourceTypeQuota, ModelName: "grok", Amount: 100},
			{ResourceKey: "gpt-image-2", ResourceType: SubscriptionResourceTypeImageCount, ModelName: "gpt-image-2", Amount: 3},
		},
		StartTime: now - 60, EndTime: now + 3600, Status: "active",
	}
	require.NoError(t, DB.Create(subscription).Error)

	result, err := PreConsumeUserSubscriptionWithResource(
		"bonus-grok", subscription.UserId, "grok-4", 0, 5,
		&SubscriptionResourceRequest{ResourceKey: "grok-4", ResourceType: SubscriptionResourceTypeQuota, Amount: 5},
	)
	require.NoError(t, err)
	assert.Equal(t, SubscriptionResourceTypeQuota, result.ResourceType)
	assert.EqualValues(t, 5, result.ResourcePreConsumed)
	updated := &UserSubscription{}
	require.NoError(t, DB.First(updated, subscription.Id).Error)
	assert.Zero(t, updated.AmountUsed)
	assert.EqualValues(t, 5, updated.ResourceGrants[0].Used)

	require.NoError(t, PostConsumeUserSubscriptionResourceDelta(subscription.Id, "grok", SubscriptionResourceTypeQuota, 2))
	require.NoError(t, RefundSubscriptionPreConsume("bonus-grok"))
	require.NoError(t, DB.First(updated, subscription.Id).Error)
	assert.EqualValues(t, 2, updated.ResourceGrants[0].Used)

	result, err = PreConsumeUserSubscriptionWithResource(
		"bonus-image", subscription.UserId, "gpt-image-2", 0, 1,
		&SubscriptionResourceRequest{ResourceKey: "gpt-image-2", ResourceType: SubscriptionResourceTypeImageCount, Amount: 2},
	)
	require.NoError(t, err)
	assert.Equal(t, SubscriptionResourceTypeImageCount, result.ResourceType)
	require.NoError(t, RefundSubscriptionPreConsume("bonus-image"))
	require.NoError(t, DB.First(updated, subscription.Id).Error)
	assert.EqualValues(t, 0, updated.ResourceGrants[1].Used)

	_, err = PreConsumeUserSubscriptionWithResource(
		"wrong-family", subscription.UserId, "claude-sonnet-5", 0, 1,
		&SubscriptionResourceRequest{ResourceKey: "claude-sonnet-5", ResourceType: SubscriptionResourceTypeQuota, Amount: 1},
	)
	assert.Error(t, err)
}

func TestSubscriptionImageOverageFallsBackToPrimaryQuotaAtomically(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id: 9861, Title: "Image overage", DurationUnit: SubscriptionDurationDay,
		DurationValue: 30, ModelFamily: "gpt-image", BillingRatio: 0.1,
		TotalAmount: 1000,
	}
	require.NoError(t, DB.Create(plan).Error)
	subscription := &UserSubscription{
		Id: 9862, UserId: 9863, PlanId: plan.Id, ModelFamily: "gpt-image", BillingRatio: 0.1,
		AmountTotal: 1000, StartTime: now - 60, EndTime: now + 3600, Status: "active",
		ResourceGrants: []SubscriptionResourceGrant{{
			ResourceKey: "gpt-image-2", ResourceType: SubscriptionResourceTypeImageCount,
			ModelName: "gpt-image-2", Amount: 1, Used: 1,
		}},
	}
	require.NoError(t, DB.Create(subscription).Error)

	applied, err := SettleUserSubscriptionImageResourceDeltaWithQuota(subscription.Id, "gpt-image-2", 1, 100)
	require.NoError(t, err)
	assert.EqualValues(t, 100, applied)
	var updated UserSubscription
	require.NoError(t, DB.First(&updated, subscription.Id).Error)
	assert.EqualValues(t, 1, updated.ResourceGrants[0].Used, "the included grant remains capped")
	assert.EqualValues(t, 100, updated.AmountUsed, "the extra image is charged to the primary quota")
}

func TestSubscriptionImageOverageRollsBackWhenPrimaryQuotaIsExhausted(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	plan := &SubscriptionPlan{
		Id: 9864, Title: "Exhausted image package", DurationUnit: SubscriptionDurationDay,
		DurationValue: 30, ModelFamily: "gpt-image", BillingRatio: 0.1,
		TotalAmount: 100,
	}
	require.NoError(t, DB.Create(plan).Error)
	subscription := &UserSubscription{
		Id: 9865, UserId: 9866, PlanId: plan.Id, ModelFamily: "gpt-image", BillingRatio: 0.1,
		AmountTotal: 100, AmountUsed: 100, StartTime: now - 60, EndTime: now + 3600, Status: "active",
		ResourceGrants: []SubscriptionResourceGrant{{
			ResourceKey: "gpt-image-2", ResourceType: SubscriptionResourceTypeImageCount,
			ModelName: "gpt-image-2", Amount: 1, Used: 1,
		}},
	}
	require.NoError(t, DB.Create(subscription).Error)

	_, err := SettleUserSubscriptionImageResourceDeltaWithQuota(subscription.Id, "gpt-image-2", 1, 100)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubscriptionImageOverageUnsettled)
	var unchanged UserSubscription
	require.NoError(t, DB.First(&unchanged, subscription.Id).Error)
	assert.EqualValues(t, 1, unchanged.ResourceGrants[0].Used)
	assert.EqualValues(t, 100, unchanged.AmountUsed)
}
