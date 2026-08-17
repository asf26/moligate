package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertTopUpQueryWindowRecord(t *testing.T, userID int, tradeNo string, createTime int64) {
	t.Helper()
	topUp := &TopUp{
		UserId:        userID,
		Amount:        100,
		Money:         30,
		TradeNo:       tradeNo,
		PaymentMethod: "wxpay",
		Status:        common.TopUpStatusSuccess,
		CreateTime:    createTime,
	}
	require.NoError(t, DB.Create(topUp).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Delete(&TopUp{}, topUp.Id).Error)
	})
}

func TestGetUserTopUpsUsesSixMonthWindow(t *testing.T) {
	userID := 9601
	now := time.Now()
	insertTopUpQueryWindowRecord(t, userID, "topup-window-in", now.AddDate(0, -5, 0).Unix())
	insertTopUpQueryWindowRecord(t, userID, "topup-window-out", now.AddDate(0, -7, 0).Unix())

	topUps, total, err := GetUserTopUps(userID, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, topUps, 1)
	assert.Equal(t, "topup-window-in", topUps[0].TradeNo)
}

func TestSearchUserTopUpsUsesSixMonthWindow(t *testing.T) {
	userID := 9602
	now := time.Now()
	insertTopUpQueryWindowRecord(t, userID, "topup-search-in", now.AddDate(0, -5, 0).Unix())
	insertTopUpQueryWindowRecord(t, userID, "topup-search-out", now.AddDate(0, -7, 0).Unix())

	withinWindow, total, err := SearchUserTopUps(userID, "topup-search-in", &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, withinWindow, 1)
	assert.Equal(t, "topup-search-in", withinWindow[0].TradeNo)

	outOfWindow, total, err := SearchUserTopUps(userID, "topup-search-out", &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, outOfWindow)
}
