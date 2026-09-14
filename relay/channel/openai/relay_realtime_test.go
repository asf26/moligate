package openai

import (
	"errors"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type realtimeBillingRecorder struct {
	reserved   int
	targets    []int
	reserveErr error
}

func (*realtimeBillingRecorder) Settle(int) error { return nil }

func (*realtimeBillingRecorder) Refund(*gin.Context) {}

func (*realtimeBillingRecorder) NeedsRefund() bool { return false }

func (r *realtimeBillingRecorder) GetPreConsumedQuota() int { return r.reserved }

func (r *realtimeBillingRecorder) Reserve(target int) error {
	r.targets = append(r.targets, target)
	if r.reserveErr != nil {
		return r.reserveErr
	}
	if target > r.reserved {
		r.reserved = target
	}
	return nil
}

func TestPreConsumeUsageDoesNotCommitRejectedChunk(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	billing := &realtimeBillingRecorder{reserved: 10, reserveErr: errors.New("quota exhausted")}
	info := &relaycommon.RelayInfo{
		OriginModelName: "realtime-test-model",
		Billing:         billing,
		PriceData: hosttypes.PriceData{
			ModelRatio:     1,
			GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1},
		},
	}
	total := &dto.RealtimeUsage{
		TotalTokens: 10,
		InputTokens: 10,
		InputTokenDetails: dto.InputTokenDetails{
			TextTokens: 10,
		},
	}

	err := preConsumeUsage(ctx, info, &dto.RealtimeUsage{
		TotalTokens: 20,
		InputTokens: 20,
		InputTokenDetails: dto.InputTokenDetails{
			TextTokens: 20,
		},
	}, total)

	require.ErrorContains(t, err, "quota exhausted")
	assert.Equal(t, 10, total.TotalTokens)
	assert.Equal(t, 10, total.InputTokens)
	assert.Equal(t, 10, total.InputTokenDetails.TextTokens)
	assert.Equal(t, []int{30}, billing.targets)
}

func TestReservePendingUsageRejectsEventBeforeItIsCommitted(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	billing := &realtimeBillingRecorder{reserved: 10, reserveErr: errors.New("quota exhausted")}
	info := &relaycommon.RelayInfo{
		OriginModelName: "realtime-test-model",
		Billing:         billing,
		PriceData: hosttypes.PriceData{
			ModelRatio:     1,
			GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1},
		},
	}
	committed := &dto.RealtimeUsage{TotalTokens: 5, OutputTokens: 5, OutputTokenDetails: dto.OutputTokenDetails{TextTokens: 5}}
	pending := &dto.RealtimeUsage{TotalTokens: 5, OutputTokens: 5, OutputTokenDetails: dto.OutputTokenDetails{TextTokens: 5}}
	event := &dto.RealtimeUsage{TotalTokens: 20, OutputTokens: 20, OutputTokenDetails: dto.OutputTokenDetails{TextTokens: 20}}

	err := reservePendingUsage(ctx, info, committed, pending, event)

	require.ErrorContains(t, err, "quota exhausted")
	assert.Equal(t, 5, pending.TotalTokens)
	assert.Equal(t, 5, pending.OutputTokenDetails.TextTokens)
	assert.Equal(t, []int{30}, billing.targets)
}

func TestPreConsumeUsageReservesCumulativeRealtimeQuota(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	billing := &realtimeBillingRecorder{}
	info := &relaycommon.RelayInfo{
		OriginModelName:          "realtime-test-model",
		BillingSource:            "subscription",
		SubscriptionBillingRatio: 0.25,
		Billing:                  billing,
		PriceData: hosttypes.PriceData{
			ModelRatio:     1,
			GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 2},
		},
	}
	total := &dto.RealtimeUsage{}

	require.NoError(t, preConsumeUsage(ctx, info, &dto.RealtimeUsage{
		TotalTokens: 40,
		InputTokens: 40,
		InputTokenDetails: dto.InputTokenDetails{
			TextTokens: 40,
		},
	}, total))
	require.NoError(t, preConsumeUsage(ctx, info, &dto.RealtimeUsage{
		TotalTokens: 60,
		InputTokens: 60,
		InputTokenDetails: dto.InputTokenDetails{
			TextTokens: 60,
		},
	}, total))

	assert.Equal(t, 100, total.TotalTokens)
	assert.Equal(t, 100, total.InputTokenDetails.TextTokens)
	assert.Equal(t, []int{10, 25}, billing.targets,
		"each usage event must grow the reservation to the cumulative charge")
}
