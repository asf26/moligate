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
		TotalAmount:    1000,
		BonusResources: []SubscriptionBonusResource{
			{ResourceKey: "grok", ResourceType: SubscriptionResourceTypeQuota, ModelName: "grok", Amount: 100},
			{ResourceKey: "gpt-image-2", ResourceType: SubscriptionResourceTypeImageCount, ModelName: "gpt-image-2", Amount: 3},
		},
	}
	require.NoError(t, DB.Create(plan).Error)
	subscription := &UserSubscription{
		Id: 9852, UserId: 9853, PlanId: plan.Id, AmountTotal: plan.TotalAmount,
		ModelFamily: "gpt", IncludedModels: []string{"gpt-5.5"},
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
