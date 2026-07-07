package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreditedQuotaFromTopUpAddsActiveRechargeBonus(t *testing.T) {
	oldPayment := *operation_setting.GetPaymentSetting()
	t.Cleanup(func() {
		*operation_setting.GetPaymentSetting() = oldPayment
	})

	payment := operation_setting.GetPaymentSetting()
	payment.RechargeBonus = operation_setting.RechargeBonus{
		Enabled:   true,
		MinAmount: 100,
		BonusRate: 0.1,
		StartTime: common.GetTimestamp() - 60,
		EndTime:   common.GetTimestamp() + 60,
	}

	topUp := &TopUp{
		Amount:          100,
		PaymentProvider: PaymentProviderEpay,
	}

	assert.Equal(t, 55_000_000, creditedQuotaFromTopUp(topUp))
}

func TestGetInviteRankingUsesInviteeCreatedWindow(t *testing.T) {
	truncateTables(t)

	now := common.GetTimestamp()
	users := []User{
		{Id: 1, Username: "alice", DisplayName: "Alice", AffCode: "alice_aff", CreatedAt: now - 100},
		{Id: 2, Username: "bob", DisplayName: "Bob", AffCode: "bob_aff", CreatedAt: now - 100},
		{Id: 9, Username: "deleted-inviter", AffCode: "deleted_inviter_aff", CreatedAt: now - 100, DeletedAt: gorm.DeletedAt{Time: time.Unix(now-30, 0), Valid: true}},
		{Id: 3, Username: "a1", AffCode: "a1_aff", InviterId: 1, CreatedAt: now - 10},
		{Id: 4, Username: "a2", AffCode: "a2_aff", InviterId: 1, CreatedAt: now - 9},
		{Id: 5, Username: "b1", AffCode: "b1_aff", InviterId: 2, CreatedAt: now - 8},
		{Id: 6, Username: "old", AffCode: "old_aff", InviterId: 2, CreatedAt: now - 1000},
		{Id: 7, Username: "deleted-invitee", AffCode: "deleted_invitee_aff", InviterId: 2, CreatedAt: now - 7, DeletedAt: gorm.DeletedAt{Time: time.Unix(now-6, 0), Valid: true}},
		{Id: 8, Username: "deleted-inviter-child", AffCode: "deleted_inviter_child_aff", InviterId: 9, CreatedAt: now - 6},
	}
	for _, user := range users {
		require.NoError(t, DB.Create(&user).Error)
	}

	ranking, err := GetInviteRanking(now-60, now, 5)
	require.NoError(t, err)
	require.Len(t, ranking, 2)
	assert.Equal(t, 1, ranking[0].UserId)
	assert.Equal(t, int64(2), ranking[0].InviteCount)
	assert.Equal(t, 2, ranking[1].UserId)
	assert.Equal(t, int64(1), ranking[1].InviteCount)
}
