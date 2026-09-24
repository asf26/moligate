/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAdminOrderTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
	})

	require.NoError(t, db.Exec(`CREATE TABLE users (
		id integer PRIMARY KEY,
		username varchar(64)
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE subscription_orders (
		id integer PRIMARY KEY,
		user_id integer,
		plan_id integer,
		money real,
		trade_no varchar(255),
		payment_method varchar(50),
		payment_provider varchar(50),
		status text,
		create_time bigint,
		complete_time bigint,
		provider_payload text,
		plan_snapshot text
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE top_ups (
		id integer PRIMARY KEY,
		user_id integer,
		amount bigint,
		money real,
		trade_no varchar(255),
		payment_method varchar(50),
		payment_provider varchar(50),
		create_time bigint,
		complete_time bigint,
		status text
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO users (id, username) VALUES (1, 'alice'), (2, 'bob')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subscription_orders
		(id, user_id, plan_id, money, trade_no, payment_method, payment_provider, status, create_time, complete_time, plan_snapshot)
		VALUES
		(1, 1, 10, 270, 'SUBUSR1AAAA', 'wxpay', 'epay', 'success', 1000, 1100, '{"id":10,"title":"黄金套餐"}'),
		(2, 2, 11, 576, 'SUBUSR2BBBB', 'wxpay', 'epay', 'pending', 2000, 0, '{"id":11,"title":"铂金套餐"}'),
		(3, 1, 10, 270, 'SUBUSR1CCCC', 'alipay', 'epay', 'success', 3000, 3100, '')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO top_ups
		(id, user_id, amount, money, trade_no, payment_method, payment_provider, status, create_time, complete_time)
		VALUES
		(11, 1, 5000000, 10, 'TOPUP1AAAA', 'alipay', 'epay', 'success', 4000, 4100),
		(12, 2, 25000000, 50, 'TOPUP2BBBB', 'stripe', 'stripe', 'success', 5000, 5100),
		(13, 1, 0, 270, 'SUBUSR1AAAA', 'wxpay', 'epay', 'success', 1000, 1100)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE redemptions (
		id integer PRIMARY KEY,
		user_id integer,
		name varchar(128),
		quota integer,
		status integer,
		source_type varchar(32),
		used_user_id integer,
		created_time bigint,
		redeemed_time bigint,
		deleted_at datetime
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO redemptions
		(id, user_id, name, quota, status, source_type, used_user_id, created_time, redeemed_time, deleted_at)
		VALUES
		(21, 1, '活动码A', 10000000, 3, '', 2, 1, 6000, NULL),
		(22, 1, '未用码', 5000000, 1, '', 0, 1, 0, NULL),
		(23, 1, '禁用码', 5000000, 2, '', 0, 1, 0, NULL),
		(24, 1, '软删已用', 5000000, 3, '', 2, 1, 7000, '2026-09-24 00:00:00')`).Error)
	return db
}

func TestListAdminOrdersMergesBothOrderKinds(t *testing.T) {
	setupAdminOrderTestDB(t)

	rows, total, err := ListAdminOrders(AdminOrderQuery{}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 6, total, "the wallet mirror of a purchase must not appear as a second order, and only used redemption codes count")
	require.Len(t, rows, 6)

	// Newest first, whatever the source.
	assert.Equal(t, []int{21, 12, 11, 3, 2, 1}, []int{rows[0].Id, rows[1].Id, rows[2].Id, rows[3].Id, rows[4].Id, rows[5].Id})
	assert.Equal(t, OrderKindRedemption, rows[0].Kind)
	assert.Equal(t, OrderKindTopUp, rows[1].Kind)
	assert.Equal(t, OrderKindTopUp, rows[2].Kind)
	assert.Equal(t, OrderKindSubscription, rows[3].Kind)

	// Redemption codes carry the credited quota, no money and no trade number.
	assert.EqualValues(t, 10000000, rows[0].Amount)
	assert.Equal(t, 0.0, rows[0].Money)
	assert.Empty(t, rows[0].TradeNo)
	assert.Equal(t, "活动码A", rows[0].Name)
	assert.Equal(t, "bob", rows[0].Username, "used_user_id is the credited user")

	// Recharges carry the credited face value and no plan title.
	assert.EqualValues(t, 25000000, rows[1].Amount)
	assert.Equal(t, 50.0, rows[1].Money)
	assert.Empty(t, rows[1].PlanTitle)
	assert.Equal(t, "bob", rows[1].Username)

	// Package purchases resolve their title from the order snapshot.
	assert.Equal(t, "黄金套餐", rows[5].PlanTitle)
	assert.EqualValues(t, 0, rows[5].Amount)
	assert.Empty(t, rows[3].PlanTitle, "an order without a snapshot keeps an empty title")
	assert.Equal(t, "alice", rows[3].Username)
}

func TestListAdminOrdersFiltersBothKinds(t *testing.T) {
	setupAdminOrderTestDB(t)

	subscriptions, total, err := ListAdminOrders(AdminOrderQuery{Kind: OrderKindSubscription}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	assert.Len(t, subscriptions, 3)

	topUps, total, err := ListAdminOrders(AdminOrderQuery{Kind: OrderKindTopUp}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, topUps, 2)

	redemptions, total, err := ListAdminOrders(AdminOrderQuery{Kind: OrderKindRedemption}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total, "only used codes with a credited user are listed")
	assert.Len(t, redemptions, 1)
	assert.Equal(t, 21, redemptions[0].Id)

	// A username filter applies to both sources.
	byUsername, total, err := ListAdminOrders(AdminOrderQuery{Username: "alice"}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 3, total, "alice has two purchases and one recharge")
	assert.Len(t, byUsername, 3)

	byUserId, total, err := ListAdminOrders(AdminOrderQuery{UserId: 2}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 3, total, "bob has a purchase, a recharge and a redeemed code")
	assert.Len(t, byUserId, 3)
	assert.Equal(t, "bob", byUserId[0].Username)

	byTradeNo, total, err := ListAdminOrders(AdminOrderQuery{TradeNo: "TOPUP"}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, byTradeNo, 2)
	assert.Equal(t, OrderKindTopUp, byTradeNo[0].Kind)

	byStatus, total, err := ListAdminOrders(AdminOrderQuery{Status: "pending"}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Len(t, byStatus, 1)
	assert.Equal(t, "SUBUSR2BBBB", byStatus[0].TradeNo)

	byTime, total, err := ListAdminOrders(AdminOrderQuery{StartTime: 4500, EndTime: 5500}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	assert.Equal(t, 12, byTime[0].Id)

	combined, total, err := ListAdminOrders(
		AdminOrderQuery{Kind: OrderKindTopUp, Username: "alice", Status: "success", StartTime: 3500},
		&common.PageInfo{Page: 1, PageSize: 20},
	)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, combined, 1)
	assert.Equal(t, 11, combined[0].Id)
}

func TestListAdminOrdersPaginatesAndClampsPageSize(t *testing.T) {
	setupAdminOrderTestDB(t)

	// Enough extra recharges to observe the default page size.
	values := make([]string, 0, 25)
	for i := 1; i <= 25; i++ {
		values = append(values, fmt.Sprintf(
			"(%d, 1, 1000, 1, 'BULK%02d', 'alipay', 'epay', 'success', %d, 0)",
			100+i, i, 1000+i,
		))
	}
	require.NoError(t, DB.Exec(
		"INSERT INTO top_ups (id, user_id, amount, money, trade_no, payment_method, payment_provider, status, create_time, complete_time) VALUES "+
			strings.Join(values, ", "),
	).Error)

	first, total, err := ListAdminOrders(AdminOrderQuery{}, &common.PageInfo{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.EqualValues(t, 31, total)
	assert.Len(t, first, 2)

	second, _, err := ListAdminOrders(AdminOrderQuery{}, &common.PageInfo{Page: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Len(t, second, 2)
	assert.Greater(t, first[0].CreateTime, second[0].CreateTime, "pages stay ordered by create time")

	// A zero page/size falls back to page 1 with the default 20 rows.
	defaulted, _, err := ListAdminOrders(AdminOrderQuery{}, &common.PageInfo{})
	require.NoError(t, err)
	assert.Len(t, defaulted, 20)

	// An oversized page size is clamped, so every remaining order still arrives.
	clamped, _, err := ListAdminOrders(AdminOrderQuery{}, &common.PageInfo{Page: 1, PageSize: 5000})
	require.NoError(t, err)
	assert.Len(t, clamped, 31)
}
