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
package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// AdminListOrders exposes every user's orders (package purchases and wallet
// recharges) to the console, with optional filters on time range, username,
// user id, trade number, kind and status.
func AdminListOrders(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId, _ := strconv.Atoi(strings.TrimSpace(c.Query("user_id")))
	orders, total, err := model.ListAdminOrders(model.AdminOrderQuery{
		UserId:    userId,
		Username:  c.Query("username"),
		TradeNo:   c.Query("trade_no"),
		Status:    c.Query("status"),
		Kind:      c.Query("kind"),
		StartTime: parseOrderFilterTime(c.Query("start_time")),
		EndTime:   parseOrderFilterTime(c.Query("end_time")),
	}, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(orders)
	common.ApiSuccess(c, pageInfo)
}

// parseOrderFilterTime reads a unix-seconds bound; anything else means "no bound".
func parseOrderFilterTime(raw string) int64 {
	seconds, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || seconds <= 0 {
		return 0
	}
	return seconds
}
