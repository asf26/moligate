/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

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
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type enterpriseBillingAccountRequest struct {
	Name        string   `json:"name"`
	Code        string   `json:"code"`
	Source      string   `json:"source"`
	Usernames   []string `json:"usernames"`
	PricingRule string   `json:"pricing_rule"`
	Enabled     *bool    `json:"enabled"`
}

func (request enterpriseBillingAccountRequest) serviceInput() service.EnterpriseBillingAccountInput {
	return service.EnterpriseBillingAccountInput{
		Name:        request.Name,
		Code:        request.Code,
		Source:      request.Source,
		Usernames:   request.Usernames,
		PricingRule: request.PricingRule,
		Enabled:     request.Enabled,
	}
}

func AdminListEnterpriseBillingAccounts(c *gin.Context) {
	accounts, err := service.ListEnterpriseBillingAccounts()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, accounts)
}

func AdminCreateEnterpriseBillingAccount(c *gin.Context) {
	var request enterpriseBillingAccountRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	account, err := service.CreateEnterpriseBillingAccount(request.serviceInput())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, account)
}

func AdminUpdateEnterpriseBillingAccount(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "企业账户 ID 无效")
		return
	}
	var request enterpriseBillingAccountRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	account, err := service.UpdateEnterpriseBillingAccount(id, request.serviceInput())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, account)
}

func AdminDeleteEnterpriseBillingAccount(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "企业账户 ID 无效")
		return
	}
	if err := service.DeleteEnterpriseBillingAccount(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func parseEnterpriseBillingReportQuery(c *gin.Context) (int, string, int, int, error) {
	accountID, err := strconv.Atoi(c.Query("account_id"))
	if err != nil || accountID <= 0 {
		return 0, "", 0, 0, fmt.Errorf("企业账户 ID 无效")
	}
	pageInfo := common.GetPageQuery(c)
	month := strings.TrimSpace(c.Query("month"))
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	return accountID, month, pageInfo.Page, pageInfo.PageSize, nil
}

func AdminGetEnterpriseBillingReport(c *gin.Context) {
	accountID, month, page, pageSize, err := parseEnterpriseBillingReportQuery(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	report, err := service.ReportEnterpriseBilling(accountID, month, page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, report)
}

func AdminExportEnterpriseBillingRaw(c *gin.Context) {
	exportEnterpriseBilling(c, false)
}

func AdminExportEnterpriseBillingCustomer(c *gin.Context) {
	exportEnterpriseBilling(c, true)
}

func exportEnterpriseBilling(c *gin.Context, customer bool) {
	accountID, month, _, _, err := parseEnterpriseBillingReportQuery(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	report, err := service.ExportEnterpriseBillingRows(accountID, month)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	kind := "raw"
	if customer {
		kind = "customer"
	}
	filename := fmt.Sprintf("enterprise-billing-%s-%s-%s.csv", report.Account.Code, report.Month, kind)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()
	if customer {
		_ = writer.Write([]string{"订单ID", "时间", "类型", "模型名称", "耗时（s)", "Request ID", "价格"})
		for _, row := range report.CustomerRows {
			_ = writer.Write([]string{
				row.OrderID,
				row.Time,
				row.ModelType,
				row.ModelName,
				strconv.Itoa(row.UseTime),
				row.RequestID,
				strconv.FormatFloat(row.Price, 'f', 6, 64),
			})
		}
		return
	}
	_ = writer.Write([]string{
		"ID", "Time", "Type", "Username", "Token", "Model", "Quota",
		"Prompt Tokens", "Completion Tokens", "Use Time", "Channel ID",
		"Channel Name", "Group", "Request ID",
	})
	for _, row := range report.RawRows {
		_ = writer.Write([]string{
			strconv.Itoa(row.ID),
			strconv.FormatInt(row.Time, 10),
			strconv.Itoa(row.Type),
			row.Username,
			row.Token,
			row.Model,
			strconv.Itoa(row.Quota),
			strconv.Itoa(row.PromptTokens),
			strconv.Itoa(row.CompletionTokens),
			strconv.Itoa(row.UseTime),
			strconv.Itoa(row.ChannelID),
			row.ChannelName,
			row.Group,
			row.RequestID,
		})
	}
}
