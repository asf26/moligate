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
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// Order kinds. Money comes in through three tables — package purchases, wallet
// recharges and redemption codes — and the admin ledger lists all of them
// because an operator reconciles them against the same payment statements.
const (
	OrderKindSubscription = "subscription"
	OrderKindTopUp        = "topup"
	OrderKindRedemption   = "redemption"
)

// adminOrderUnion projects the three order tables onto one column set. It
// carries no user input: filters, ordering and paging are applied by the caller
// on top of this derived table, which keeps the statement portable across
// databases.
//
// Recharge rows whose trade_no matches a subscription order are the wallet
// mirror the payment callback writes for a package purchase — they are not
// recharges, and listing them would show every purchase twice.
const adminOrderUnion = `
SELECT 'subscription' AS kind, id, user_id, plan_id, money, 0 AS amount, trade_no,
       payment_method, payment_provider, status, create_time, complete_time, plan_snapshot, '' AS name
FROM subscription_orders
UNION ALL
SELECT 'topup' AS kind, id, user_id, 0 AS plan_id, money, amount, trade_no,
       payment_method, payment_provider, status, create_time, complete_time, '' AS plan_snapshot, '' AS name
FROM top_ups
WHERE NOT EXISTS (
    SELECT 1 FROM subscription_orders s WHERE s.trade_no = top_ups.trade_no
)
UNION ALL
SELECT 'redemption' AS kind, id, used_user_id AS user_id, 0 AS plan_id, 0 AS money, quota AS amount,
       '' AS trade_no, '' AS payment_method, '' AS payment_provider, 'success' AS status,
       redeemed_time AS create_time, redeemed_time AS complete_time, '' AS plan_snapshot, name
FROM redemptions
WHERE status = 3 AND used_user_id > 0 AND deleted_at IS NULL`

// AdminOrderQuery filters the admin order ledger. Empty values mean "no
// constraint"; the time bounds are inclusive unix seconds on create_time.
type AdminOrderQuery struct {
	UserId    int
	Username  string
	TradeNo   string
	Status    string
	Kind      string
	StartTime int64
	EndTime   int64
}

// AdminOrderRow is one row of the admin order ledger. For recharges, Amount is
// the credited face value in CNY (gift multipliers included) while Money is
// what was actually paid; for redemptions Amount is the credited API quota and
// Money is always 0. PlanSnapshot is read to resolve the purchased plan title
// and is never serialized.
type AdminOrderRow struct {
	Id              int     `json:"id"`
	Kind            string  `json:"kind"`
	UserId          int     `json:"user_id"`
	Username        string  `json:"username"`
	PlanId          int     `json:"plan_id"`
	PlanTitle       string  `json:"plan_title"`
	TradeNo         string  `json:"trade_no"`
	Money           float64 `json:"money"`
	Amount          int64   `json:"amount"`
	PaymentMethod   string  `json:"payment_method"`
	PaymentProvider string  `json:"payment_provider"`
	Status          string  `json:"status"`
	CreateTime      int64   `json:"create_time"`
	CompleteTime    int64   `json:"complete_time"`
	Name            string  `json:"name"`

	PlanSnapshot string `json:"-" gorm:"column:plan_snapshot"`
}

// ListAdminOrders returns one page of every user's orders across both order
// kinds, newest first, together with the buyer's username.
func ListAdminOrders(query AdminOrderQuery, pageInfo *common.PageInfo) ([]AdminOrderRow, int64, error) {
	if pageInfo == nil {
		pageInfo = &common.PageInfo{}
	}
	if pageInfo.Page < 1 {
		pageInfo.Page = 1
	}
	if pageInfo.PageSize <= 0 {
		pageInfo.PageSize = 20
	}
	if pageInfo.PageSize > 100 {
		pageInfo.PageSize = 100
	}

	// The buyer is joined so the ledger can be filtered by username. A missing
	// user must not hide the order, hence the LEFT JOIN.
	scoped := func() *gorm.DB {
		tx := DB.Table("(?) AS orders", DB.Raw(adminOrderUnion)).
			Joins("LEFT JOIN users AS u ON u.id = orders.user_id")
		if query.UserId > 0 {
			tx = tx.Where("orders.user_id = ?", query.UserId)
		}
		if username := strings.TrimSpace(query.Username); username != "" {
			tx = tx.Where("u.username LIKE ?", "%"+username+"%")
		}
		if tradeNo := strings.TrimSpace(query.TradeNo); tradeNo != "" {
			tx = tx.Where("orders.trade_no LIKE ?", "%"+tradeNo+"%")
		}
		if status := strings.TrimSpace(query.Status); status != "" {
			tx = tx.Where("orders.status = ?", status)
		}
		if kind := strings.TrimSpace(query.Kind); kind != "" {
			tx = tx.Where("orders.kind = ?", kind)
		}
		if query.StartTime > 0 {
			tx = tx.Where("orders.create_time >= ?", query.StartTime)
		}
		if query.EndTime > 0 {
			tx = tx.Where("orders.create_time <= ?", query.EndTime)
		}
		return tx
	}

	var total int64
	if err := scoped().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]AdminOrderRow, 0, pageInfo.PageSize)
	err := scoped().
		Select("orders.id, orders.kind, orders.user_id, orders.plan_id, orders.money, " +
			"orders.amount, orders.trade_no, orders.payment_method, orders.payment_provider, " +
			"orders.status, orders.create_time, orders.complete_time, orders.plan_snapshot, " +
			"orders.name, u.username").
		Order("orders.create_time DESC, orders.id DESC, orders.kind").
		Limit(pageInfo.PageSize).
		Offset(pageInfo.GetStartIdx()).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	// The title comes from the order's own snapshot: renaming or deleting a plan
	// must not rewrite what the customer bought.
	for i := range rows {
		if rows[i].Kind == OrderKindSubscription {
			rows[i].PlanTitle = subscriptionOrderSnapshotPlanTitle(rows[i].PlanSnapshot)
		}
	}
	return rows, total, nil
}

func subscriptionOrderSnapshotPlanTitle(snapshot string) string {
	if strings.TrimSpace(snapshot) == "" {
		return ""
	}
	var plan SubscriptionPlan
	if err := common.UnmarshalJsonStr(snapshot, &plan); err != nil {
		return ""
	}
	return plan.Title
}
