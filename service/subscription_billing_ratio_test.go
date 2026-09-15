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

func TestSubscriptionPreConsumeQuotaUsesSnapshotAndWalletKeepsLiveQuota(t *testing.T) {
	truncate(t)
	now := time.Now().Unix()
	require.NoError(t, model.DB.Create(&model.UserSubscription{
		Id:               9881,
		UserId:           9882,
		ModelFamily:      "gpt",
		IncludedModels:   []string{"gpt-5.5"},
		ApplicableGroups: []string{"plus/team"},
		BillingRatio:     0.14,
		AmountTotal:      10_000,
		StartTime:        now - 60,
		EndTime:          now + 3600,
		Status:           "active",
	}).Error)

	info := &relaycommon.RelayInfo{
		UserId:          9882,
		OriginModelName: "gpt-5.5",
		UsingGroup:      "plus/team",
		UserSetting:     dto.UserSetting{BillingPreference: "subscription_first"},
		PriceData:       types.PriceData{GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1}, BaseQuotaBeforeGroup: 1_000},
	}
	quota, ratio, ok := subscriptionPreConsumeQuota(info, 1_000)
	assert.True(t, ok)
	assert.InDelta(t, 0.14, ratio, 0.000001)
	assert.Equal(t, 140, quota)

	info.UserSetting.BillingPreference = "subscription_first"
	info.PriceData.GroupRatioInfo.GroupRatio = 0
	quota, _, ok = subscriptionPreConsumeQuota(info, 0)
	assert.True(t, ok)
	assert.Equal(t, 140, quota)
}

func TestNewBillingSessionAllowsZeroCostWalletWithZeroBalance(t *testing.T) {
	truncate(t)
	const userID = 9890
	seedUser(t, userID, 0)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		UserId:          userID,
		OriginModelName: "gpt-5.5",
		UsingGroup:      "free",
		UserSetting:     dto.UserSetting{BillingPreference: "wallet_only"},
		PriceData: types.PriceData{
			GroupRatioInfo:       types.GroupRatioInfo{GroupRatio: 0},
			BaseQuotaBeforeGroup: 1_000,
		},
	}

	session, apiErr := NewBillingSession(ctx, info, 0)
	require.Nil(t, apiErr)
	require.NotNil(t, session)
	assert.Equal(t, BillingSourceWallet, session.funding.Source())
	assert.Zero(t, session.GetPreConsumedQuota())
}

func TestNewBillingSessionUsesActuallySelectedPackageRatio(t *testing.T) {
	truncate(t)
	now := time.Now().Unix()
	const userID = 9891
	seedUser(t, userID, 0)
	firstPlan := &model.SubscriptionPlan{
		Id: 9892, Title: "First package", DurationUnit: model.SubscriptionDurationMonth,
		DurationValue: 1, ModelFamily: "gpt", BillingRatio: 1, TotalAmount: 200,
	}
	secondPlan := &model.SubscriptionPlan{
		Id: 9893, Title: "Fallback package", DurationUnit: model.SubscriptionDurationMonth,
		DurationValue: 1, ModelFamily: "gpt", BillingRatio: 0.5, TotalAmount: 100,
	}
	require.NoError(t, model.DB.Create(firstPlan).Error)
	require.NoError(t, model.DB.Create(secondPlan).Error)
	require.NoError(t, model.DB.Create(&model.UserSubscription{
		Id: 9894, UserId: userID, PlanId: firstPlan.Id, ModelFamily: "gpt", BillingRatio: 1,
		AmountTotal: 200, AmountUsed: 100, Status: "active", StartTime: now - 60, EndTime: now + 1800,
	}).Error)
	require.NoError(t, model.DB.Create(&model.UserSubscription{
		Id: 9895, UserId: userID, PlanId: secondPlan.Id, ModelFamily: "gpt", BillingRatio: 0.5,
		AmountTotal: 100, Status: "active", StartTime: now - 60, EndTime: now + 3600,
	}).Error)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		RequestId:       "session-selected-ratio",
		UserId:          userID,
		OriginModelName: "gpt-5.5",
		UsingGroup:      "default",
		IsPlayground:    true,
		UserSetting:     dto.UserSetting{BillingPreference: "subscription_only"},
		PriceData: types.PriceData{
			GroupRatioInfo:       types.GroupRatioInfo{GroupRatio: 1},
			BaseQuotaBeforeGroup: 200,
		},
	}

	session, apiErr := NewBillingSession(ctx, info, 200)
	require.Nil(t, apiErr)
	require.NotNil(t, session)
	assert.Equal(t, 100, session.GetPreConsumedQuota())
	assert.Equal(t, 9895, info.SubscriptionId)
	assert.InDelta(t, 0.5, info.SubscriptionBillingRatio, 0.000001)

	var selected model.UserSubscription
	require.NoError(t, model.DB.First(&selected, 9895).Error)
	assert.EqualValues(t, 100, selected.AmountUsed)
}

func TestSubscriptionResourceRequestCountsNativeGeminiImageGeneration(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "gemini-3-pro-image-preview",
		Request:         &dto.GeminiChatRequest{},
	}

	resource := subscriptionResourceRequest(info, 1_000)
	require.NotNil(t, resource)
	assert.Equal(t, "nano-banana", resource.ResourceKey)
	assert.Equal(t, model.SubscriptionResourceTypeImageCount, resource.ResourceType)
	assert.EqualValues(t, 1, resource.Amount)
}

func TestSubscriptionResourceRequestRecognizesStableGeminiImageModels(t *testing.T) {
	for _, modelName := range []string{"gemini-2.0-flash-exp-image-generation", "gemini-2.0-flash-exp", "gemini-2.5-flash-image", "gemini-3-pro-image", "gemini-3.1-flash-image", "nano-banana-pro-preview"} {
		resource := subscriptionResourceRequest(&relaycommon.RelayInfo{OriginModelName: modelName}, 1)
		require.NotNil(t, resource)
		assert.Equal(t, model.SubscriptionResourceTypeImageCount, resource.ResourceType)
		assert.Equal(t, "nano-banana", resource.ResourceKey)
	}
}

func TestSubscriptionResourceRequestUsesGeminiCandidateCount(t *testing.T) {
	candidateCount := 3
	resource := subscriptionResourceRequest(&relaycommon.RelayInfo{
		OriginModelName: "gemini-3-pro-image",
		Request: &dto.GeminiChatRequest{GenerationConfig: dto.GeminiChatGenerationConfig{
			CandidateCount: &candidateCount,
		}},
	}, 1)
	require.NotNil(t, resource)
	assert.EqualValues(t, 3, resource.Amount)
}

func TestBillingSessionSettlesActualImageGenerationCount(t *testing.T) {
	for index, testCase := range []struct {
		name      string
		requested uint
		actual    int64
	}{
		{name: "refunds generations not returned", requested: 3, actual: 2},
		{name: "refunds when no image is returned", requested: 1, actual: 0},
		{name: "charges additional generations returned", requested: 1, actual: 3},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			truncate(t)
			now := time.Now().Unix()
			planID := 9910 + index
			subscriptionID := 9920 + index
			userID := 9930 + index
			seedUser(t, userID, 0)
			plan := &model.SubscriptionPlan{
				Id: planID, Title: "Image package", DurationUnit: model.SubscriptionDurationDay,
				DurationValue: 30, ModelFamily: "gpt-image", BillingRatio: 0.1, TotalAmount: 1,
			}
			require.NoError(t, model.DB.Create(plan).Error)
			require.NoError(t, model.DB.Create(&model.UserSubscription{
				Id: subscriptionID, UserId: userID, PlanId: planID,
				ModelFamily: "gpt-image", IncludedModels: []string{"gpt-image-2"}, BillingRatio: 0.1,
				AmountTotal: 1, Status: "active", StartTime: now - 60, EndTime: now + 3600,
				ResourceGrants: []model.SubscriptionResourceGrant{{
					ResourceKey: "gpt-image-2", ResourceType: model.SubscriptionResourceTypeImageCount,
					ModelName: "gpt-image-2", Amount: 10,
				}},
			}).Error)

			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			info := &relaycommon.RelayInfo{
				RequestId:       "settle-image-count-" + testCase.name,
				UserId:          userID,
				OriginModelName: "gpt-image-2",
				UsingGroup:      "gpt-image-2",
				IsPlayground:    true,
				Request:         &dto.ImageRequest{N: &testCase.requested},
				UserSetting:     dto.UserSetting{BillingPreference: "subscription_only"},
				PriceData: types.PriceData{
					UsePrice:             true,
					ModelPrice:           0.1,
					GroupRatioInfo:       types.GroupRatioInfo{GroupRatio: 1},
					BaseQuotaBeforeGroup: 100,
				},
			}
			session, apiErr := NewBillingSession(ctx, info, 100)
			require.Nil(t, apiErr)
			require.NotNil(t, session)
			info.PriceData.AddOtherRatio("n", float64(testCase.actual))
			info.ActualImageCount = testCase.actual
			info.ActualImageCountSet = true
			require.NoError(t, session.Settle(session.GetPreConsumedQuota()))

			var subscription model.UserSubscription
			require.NoError(t, model.DB.First(&subscription, subscriptionID).Error)
			require.Len(t, subscription.ResourceGrants, 1)
			assert.Equal(t, testCase.actual, subscription.ResourceGrants[0].Used)
			assert.Equal(t, testCase.actual, info.SubscriptionResourceAmount)
			assert.Zero(t, subscription.AmountUsed)
		})
	}
}

func TestBillingSessionChargesImageOverageWithPurchasedRatio(t *testing.T) {
	truncate(t)
	now := time.Now().Unix()
	const userID = 9940
	const planID = 9941
	const subscriptionID = 9942
	seedUser(t, userID, 0)
	plan := &model.SubscriptionPlan{
		Id: planID, Title: "Image package with overage", DurationUnit: model.SubscriptionDurationDay,
		DurationValue: 30, ModelFamily: "gpt-image", BillingRatio: 0.1, TotalAmount: 1_000,
	}
	require.NoError(t, model.DB.Create(plan).Error)
	require.NoError(t, model.DB.Create(&model.UserSubscription{
		Id: subscriptionID, UserId: userID, PlanId: planID, ModelFamily: "gpt-image",
		IncludedModels: []string{"gpt-image-2"}, BillingRatio: 0.1,
		AmountTotal: 1_000, Status: "active", StartTime: now - 60, EndTime: now + 3600,
		ResourceGrants: []model.SubscriptionResourceGrant{{
			ResourceKey: "gpt-image-2", ResourceType: model.SubscriptionResourceTypeImageCount,
			ModelName: "gpt-image-2", Amount: 1,
		}},
	}).Error)

	requested := uint(1)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		RequestId: "settle-image-overage", UserId: userID, OriginModelName: "gpt-image-2",
		UsingGroup: "default", IsPlayground: true, Request: &dto.ImageRequest{N: &requested},
		UserSetting: dto.UserSetting{BillingPreference: "subscription_only"},
		PriceData: types.PriceData{
			UsePrice: true, ModelPrice: 1, BaseQuotaBeforeGroup: 1_000,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 9},
		},
	}
	session, apiErr := NewBillingSession(ctx, info, 9_000)
	require.Nil(t, apiErr)
	// The grant covers one image, while the upstream returned three.  The two
	// image overage must be charged at one purchased-quota unit per image.
	info.ActualImageCount = 3
	info.ActualImageCountSet = true
	// Keep the actual quota equal to the pre-consumed amount. The session must
	// still charge one extra image using the purchased snapshot, not the live
	// group ratio (9).
	require.NoError(t, session.Settle(session.GetPreConsumedQuota()))

	var subscription model.UserSubscription
	require.NoError(t, model.DB.First(&subscription, subscriptionID).Error)
	assert.EqualValues(t, 200, subscription.AmountUsed)
	assert.EqualValues(t, 1, subscription.ResourceGrants[0].Used)
	assert.EqualValues(t, 200, info.SubscriptionPostDelta)
}
