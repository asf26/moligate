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
	require.NoError(t, db.Exec(`INSERT INTO users (id, username) VALUES (1, 'alice'), (2, 'bob')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subscription_orders
		(id, user_id, plan_id, money, trade_no, payment_method, payment_provider, status, create_time, complete_time, plan_snapshot)
		VALUES
		(1, 1, 10, 270, 'SUBUSR1AAAA', 'wxpay', 'epay', 'success', 1000, 1100, '{"id":10,"title":"黄金套餐"}'),
		(2, 2, 11, 576, 'SUBUSR2BBBB', 'wxpay', 'epay', 'pending', 2000, 0, '{"id":11,"title":"铂金套餐"}'),
		(3, 1, 10, 270, 'SUBUSR1CCCC', 'alipay', 'epay', 'success', 3000, 3100, '')`).Error)
	return db
}

func TestListAdminSubscriptionOrdersReturnsEveryUsersOrdersNewestFirst(t *testing.T) {
	setupAdminOrderTestDB(t)

	rows, total, err := ListAdminSubscriptionOrders(AdminSubscriptionOrderQuery{}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	require.Len(t, rows, 3)

	assert.Equal(t, []int{3, 2, 1}, []int{rows[0].Id, rows[1].Id, rows[2].Id})
	assert.Equal(t, "alice", rows[0].Username)
	assert.Equal(t, "bob", rows[1].Username)
	assert.Equal(t, "黄金套餐", rows[2].PlanTitle)
	assert.Empty(t, rows[0].PlanTitle, "an order without a snapshot keeps an empty title")
	assert.Equal(t, 270.0, rows[2].Money)
	assert.Equal(t, "SUBUSR1AAAA", rows[2].TradeNo)
}

func TestListAdminSubscriptionOrdersFilters(t *testing.T) {
	setupAdminOrderTestDB(t)

	byUsername, total, err := ListAdminSubscriptionOrders(AdminSubscriptionOrderQuery{Username: "ali"}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, byUsername, 2)
	assert.Equal(t, "alice", byUsername[0].Username)

	byUserId, total, err := ListAdminSubscriptionOrders(AdminSubscriptionOrderQuery{UserId: 2}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, byUserId, 1)
	assert.Equal(t, "bob", byUserId[0].Username)

	byTradeNo, total, err := ListAdminSubscriptionOrders(AdminSubscriptionOrderQuery{TradeNo: "BBBB"}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, byTradeNo, 1)
	assert.Equal(t, "SUBUSR2BBBB", byTradeNo[0].TradeNo)

	byStatus, total, err := ListAdminSubscriptionOrders(AdminSubscriptionOrderQuery{Status: "success"}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, byStatus, 2)

	byTime, total, err := ListAdminSubscriptionOrders(AdminSubscriptionOrderQuery{StartTime: 1500, EndTime: 2500}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, byTime, 1)
	assert.Equal(t, 2, byTime[0].Id)

	combined, total, err := ListAdminSubscriptionOrders(
		AdminSubscriptionOrderQuery{Username: "alice", Status: "success", StartTime: 2500},
		&common.PageInfo{Page: 1, PageSize: 20},
	)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, combined, 1)
	assert.Equal(t, 3, combined[0].Id)
}

func TestListAdminSubscriptionOrdersPaginatesAndClampsPageSize(t *testing.T) {
	setupAdminOrderTestDB(t)

	// Enough extra orders to observe the default page size.
	values := make([]string, 0, 25)
	for i := 1; i <= 25; i++ {
		values = append(values, fmt.Sprintf(
			"(%d, 1, 10, 10, 'BULK%02d', 'wxpay', 'epay', 'success', %d, 0, '')",
			100+i, i, 1000+i,
		))
	}
	require.NoError(t, DB.Exec(
		"INSERT INTO subscription_orders (id, user_id, plan_id, money, trade_no, payment_method, payment_provider, status, create_time, complete_time, plan_snapshot) VALUES "+
			strings.Join(values, ", "),
	).Error)

	first, total, err := ListAdminSubscriptionOrders(AdminSubscriptionOrderQuery{}, &common.PageInfo{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.EqualValues(t, 28, total)
	assert.Len(t, first, 2)

	second, _, err := ListAdminSubscriptionOrders(AdminSubscriptionOrderQuery{}, &common.PageInfo{Page: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Len(t, second, 2)
	assert.Greater(t, first[1].CreateTime, second[0].CreateTime, "pages stay ordered by create time")

	// A zero page/size falls back to page 1 with the default 20 rows.
	defaulted, _, err := ListAdminSubscriptionOrders(AdminSubscriptionOrderQuery{}, &common.PageInfo{})
	require.NoError(t, err)
	assert.Len(t, defaulted, 20)

	// An oversized page size is clamped, so every remaining order still arrives.
	clamped, _, err := ListAdminSubscriptionOrders(AdminSubscriptionOrderQuery{}, &common.PageInfo{Page: 1, PageSize: 5000})
	require.NoError(t, err)
	assert.Len(t, clamped, 28)
}
