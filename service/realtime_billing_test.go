package service

import (
	"errors"
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

func TestReserveWssConsumeQuotaUsesSubscriptionAndCumulativeUsage(t *testing.T) {
	truncate(t)
	const (
		userID         = 9951
		subscriptionID = 9952
	)
	seedUser(t, userID, 0)
	require.NoError(t, model.DB.Create(&model.UserSubscription{
		Id:           subscriptionID,
		UserId:       userID,
		BillingRatio: 0.5,
		AmountTotal:  1_000_000,
		AmountUsed:   50,
		Status:       "active",
		StartTime:    time.Now().Add(-time.Minute).Unix(),
		EndTime:      time.Now().Add(time.Hour).Unix(),
	}).Error)

	info := &relaycommon.RelayInfo{
		UserId:                   userID,
		ChannelMeta:              &relaycommon.ChannelMeta{},
		OriginModelName:          "realtime-test-model",
		StartTime:                time.Now(),
		IsPlayground:             true,
		BillingSource:            BillingSourceSubscription,
		SubscriptionId:           subscriptionID,
		SubscriptionBillingRatio: 0.5,
		SubscriptionAmountTotal:  1_000_000,
		FinalPreConsumedQuota:    50,
		PriceData: types.PriceData{
			ModelRatio:     1,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 2},
		},
	}
	funding := &SubscriptionFunding{
		subscriptionId:  subscriptionID,
		preConsumed:     50,
		AmountTotal:     1_000_000,
		AmountUsedAfter: 50,
		BillingRatio:    0.5,
	}
	session := &BillingSession{
		relayInfo:        info,
		funding:          funding,
		preConsumedQuota: 50,
	}
	info.Billing = session

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	firstTotal := &dto.RealtimeUsage{
		TotalTokens: 200,
		InputTokens: 200,
		InputTokenDetails: dto.InputTokenDetails{
			TextTokens: 200,
		},
	}
	require.NoError(t, ReserveWssConsumeQuota(ctx, info, firstTotal))
	assert.Equal(t, 100, session.GetPreConsumedQuota())

	secondTotal := &dto.RealtimeUsage{
		TotalTokens: 500,
		InputTokens: 500,
		InputTokenDetails: dto.InputTokenDetails{
			TextTokens: 500,
		},
	}
	require.NoError(t, ReserveWssConsumeQuota(ctx, info, secondTotal))
	assert.Equal(t, 250, session.GetPreConsumedQuota())

	var subscription model.UserSubscription
	require.NoError(t, model.DB.First(&subscription, subscriptionID).Error)
	assert.EqualValues(t, 250, subscription.AmountUsed,
		"the second event must reserve only the cumulative delta")
	userQuota, err := model.GetUserQuota(userID, false)
	require.NoError(t, err)
	assert.Zero(t, userQuota, "subscription-funded realtime usage must not inspect or charge the wallet")

	// The websocket finalizer receives the same cumulative quota. It must not
	// charge either event again, and repeated settlement remains idempotent.
	require.NoError(t, PostWssConsumeQuota(ctx, info, secondTotal, ""))
	require.NoError(t, session.Settle(250))
	require.NoError(t, model.DB.First(&subscription, subscriptionID).Error)
	assert.EqualValues(t, 250, subscription.AmountUsed)
}

func TestCountTokenRealtimeCountsOutputTextBeforeForwarding(t *testing.T) {
	textTokens, audioTokens, err := CountTokenRealtime(
		&relaycommon.RelayInfo{},
		dto.RealtimeEvent{Type: dto.RealtimeEventResponseOutputTextDelta, Delta: "hello realtime world"},
		"gpt-4o-realtime-preview",
	)
	require.NoError(t, err)
	assert.Positive(t, textTokens)
	assert.Zero(t, audioTokens)
}

type failingRealtimeSettler struct {
	settleErr error
}

func (s *failingRealtimeSettler) Settle(int) error       { return s.settleErr }
func (*failingRealtimeSettler) Refund(*gin.Context)      {}
func (*failingRealtimeSettler) NeedsRefund() bool        { return true }
func (*failingRealtimeSettler) GetPreConsumedQuota() int { return 50 }
func (*failingRealtimeSettler) Reserve(int) error        { return nil }

func TestPostWssConsumeQuotaReturnsSettlementFailureBeforeAccounting(t *testing.T) {
	truncate(t)
	const (
		userID    = 9953
		channelID = 9954
	)
	seedUser(t, userID, 1_000)
	seedChannel(t, channelID)
	settleErr := errors.New("settlement failed")
	info := &relaycommon.RelayInfo{
		UserId:          userID,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: channelID},
		OriginModelName: "realtime-test-model",
		StartTime:       time.Now(),
		Billing:         &failingRealtimeSettler{settleErr: settleErr},
		PriceData: types.PriceData{
			ModelRatio:     1,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
	}
	usage := &dto.RealtimeUsage{
		TotalTokens: 100,
		InputTokens: 100,
		InputTokenDetails: dto.InputTokenDetails{
			TextTokens: 100,
		},
	}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	err := PostWssConsumeQuota(ctx, info, usage, "")
	require.ErrorIs(t, err, settleErr)

	var user model.User
	require.NoError(t, model.DB.First(&user, userID).Error)
	assert.Zero(t, user.UsedQuota)
	assert.Zero(t, user.RequestCount)
	var channel model.Channel
	require.NoError(t, model.DB.First(&channel, channelID).Error)
	assert.Zero(t, channel.UsedQuota)
	var logCount int64
	require.NoError(t, model.DB.Model(&model.Log{}).Where("user_id = ?", userID).Count(&logCount).Error)
	assert.Zero(t, logCount)
}
