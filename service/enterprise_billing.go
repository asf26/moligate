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
package service

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
)

const (
	EnterpriseBillingCurrency        = "USD"
	EnterpriseBillingMaxRows         = 100000
	EnterpriseBillingMaxPageSize     = 100
	EnterpriseBillingDefaultPageSize = 50
)

var enterpriseBillingCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type EnterpriseBillingAccountInput struct {
	Name        string
	Code        string
	Source      string
	Usernames   []string
	PricingRule string
	Enabled     *bool
}

type EnterpriseBillingAccountView struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	Code             string   `json:"code"`
	Source           string   `json:"source"`
	Usernames        []string `json:"usernames"`
	PricingRule      string   `json:"pricing_rule"`
	Enabled          bool     `json:"enabled"`
	SourceConfigured bool     `json:"source_configured"`
	CreatedAt        int64    `json:"created_at"`
	UpdatedAt        int64    `json:"updated_at"`
}

type EnterpriseBillingRawRow struct {
	ID               int    `json:"id"`
	Time             int64  `json:"time"`
	Type             int    `json:"type"`
	Username         string `json:"username"`
	Token            string `json:"token"`
	Model            string `json:"model"`
	Quota            int    `json:"quota"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	UseTime          int    `json:"use_time"`
	ChannelID        int    `json:"channel_id"`
	ChannelName      string `json:"channel_name"`
	Group            string `json:"group"`
	RequestID        string `json:"request_id"`
}

type EnterpriseBillingCustomerRow struct {
	OrderID   string  `json:"order_id"`
	Time      string  `json:"time"`
	ModelType string  `json:"model_type"`
	ModelName string  `json:"model_name"`
	UseTime   int     `json:"use_time"`
	RequestID string  `json:"request_id"`
	Price     float64 `json:"price"`
}

type EnterpriseBillingSummary struct {
	RawCount         int     `json:"raw_count"`
	BilledCount      int     `json:"billed_count"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalQuota       int64   `json:"total_quota"`
	TotalPrice       float64 `json:"total_price"`
	Currency         string  `json:"currency"`
}

type EnterpriseBillingReport struct {
	Account      EnterpriseBillingAccountView   `json:"account"`
	Month        string                         `json:"month"`
	PeriodStart  int64                          `json:"period_start"`
	PeriodEnd    int64                          `json:"period_end"`
	Summary      EnterpriseBillingSummary       `json:"summary"`
	RawRows      []EnterpriseBillingRawRow      `json:"raw_rows"`
	CustomerRows []EnterpriseBillingCustomerRow `json:"customer_rows"`
	Page         int                            `json:"page"`
	PageSize     int                            `json:"page_size"`
	Total        int                            `json:"total"`
}

func ParseEnterpriseBillingMonth(month string) (time.Time, time.Time, error) {
	month = strings.TrimSpace(month)
	if len(month) != len("2006-01") {
		return time.Time{}, time.Time{}, errors.New("月份必须使用 YYYY-MM 格式")
	}
	parsed, err := time.ParseInLocation("2006-01", month, time.Local)
	if err != nil || parsed.Format("2006-01") != month {
		return time.Time{}, time.Time{}, errors.New("月份必须使用 YYYY-MM 格式")
	}
	return parsed, parsed.AddDate(0, 1, 0), nil
}

func validateEnterpriseBillingInput(input EnterpriseBillingAccountInput) (EnterpriseBillingAccountInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	input.Source = strings.ToLower(strings.TrimSpace(input.Source))
	input.PricingRule = strings.TrimSpace(input.PricingRule)
	if input.Name == "" || len(input.Name) > 128 {
		return input, errors.New("企业名称不能为空且不能超过 128 个字符")
	}
	if !enterpriseBillingCodePattern.MatchString(input.Code) {
		return input, errors.New("企业编码只能包含小写字母、数字、下划线和短横线")
	}
	if input.Source != model.EnterpriseBillingSourceLocal && input.Source != model.EnterpriseBillingSourceS10 && input.Source != model.EnterpriseBillingSourceMoligate {
		return input, errors.New("不支持的账单数据源")
	}
	if len(input.PricingRule) == 0 || len(input.PricingRule) > 4096 {
		return input, errors.New("合同算价规则不能为空且不能超过 4096 个字符")
	}
	value, _, err := billingexpr.RunExpr(input.PricingRule, billingexpr.TokenParams{P: 1, C: 1, Len: 1})
	if err != nil {
		return input, fmt.Errorf("合同算价规则无效: %w", err)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return input, errors.New("合同算价规则必须返回非负有限数值")
	}
	cleaned := make([]string, 0, len(input.Usernames))
	seen := make(map[string]struct{}, len(input.Usernames))
	for _, username := range input.Usernames {
		username = strings.TrimSpace(username)
		if username == "" {
			continue
		}
		if len(username) > 255 {
			return input, errors.New("使用账号不能超过 255 个字符")
		}
		if _, ok := seen[username]; ok {
			continue
		}
		seen[username] = struct{}{}
		cleaned = append(cleaned, username)
	}
	if len(cleaned) == 0 {
		return input, errors.New("至少需要配置一个使用账号")
	}
	sort.Strings(cleaned)
	input.Usernames = cleaned
	if input.Enabled == nil {
		enabled := true
		input.Enabled = &enabled
	}
	return input, nil
}

func accountView(account *model.EnterpriseBillingAccount) EnterpriseBillingAccountView {
	return EnterpriseBillingAccountView{
		ID:               account.Id,
		Name:             account.Name,
		Code:             account.Code,
		Source:           account.Source,
		Usernames:        account.UsernameList(),
		PricingRule:      account.PricingRule,
		Enabled:          account.Enabled,
		SourceConfigured: model.EnterpriseBillingSourceConfigured(account.Source),
		CreatedAt:        account.CreatedAt,
		UpdatedAt:        account.UpdatedAt,
	}
}

func ListEnterpriseBillingAccounts() ([]EnterpriseBillingAccountView, error) {
	accounts, err := model.ListEnterpriseBillingAccounts()
	if err != nil {
		return nil, err
	}
	result := make([]EnterpriseBillingAccountView, 0, len(accounts))
	for _, account := range accounts {
		result = append(result, accountView(account))
	}
	return result, nil
}

func CreateEnterpriseBillingAccount(input EnterpriseBillingAccountInput) (EnterpriseBillingAccountView, error) {
	input, err := validateEnterpriseBillingInput(input)
	if err != nil {
		return EnterpriseBillingAccountView{}, err
	}
	account := &model.EnterpriseBillingAccount{
		Name:        input.Name,
		Code:        input.Code,
		Source:      input.Source,
		PricingRule: input.PricingRule,
		Enabled:     *input.Enabled,
	}
	if err := account.SetUsernameList(input.Usernames); err != nil {
		return EnterpriseBillingAccountView{}, err
	}
	if err := model.CreateEnterpriseBillingAccount(account); err != nil {
		return EnterpriseBillingAccountView{}, err
	}
	return accountView(account), nil
}

func UpdateEnterpriseBillingAccount(id int, input EnterpriseBillingAccountInput) (EnterpriseBillingAccountView, error) {
	if id <= 0 {
		return EnterpriseBillingAccountView{}, errors.New("企业账户 ID 无效")
	}
	input, err := validateEnterpriseBillingInput(input)
	if err != nil {
		return EnterpriseBillingAccountView{}, err
	}
	account, err := model.GetEnterpriseBillingAccount(id)
	if err != nil {
		return EnterpriseBillingAccountView{}, err
	}
	account.Name = input.Name
	account.Code = input.Code
	account.Source = input.Source
	account.PricingRule = input.PricingRule
	account.Enabled = *input.Enabled
	if err := account.SetUsernameList(input.Usernames); err != nil {
		return EnterpriseBillingAccountView{}, err
	}
	if err := model.SaveEnterpriseBillingAccount(account); err != nil {
		return EnterpriseBillingAccountView{}, err
	}
	return accountView(account), nil
}

func DeleteEnterpriseBillingAccount(id int) error {
	if id <= 0 {
		return errors.New("企业账户 ID 无效")
	}
	return model.DeleteEnterpriseBillingAccount(id)
}

func EnterpriseBillingModelType(modelName string) string {
	name := strings.ToLower(strings.TrimSpace(modelName))
	for _, marker := range []string{
		"video", "t2v", "text-to-video", "video_generation", "video-generation",
		"sora", "kling", "seedance", "veo", "runway", "hailuo", "minimax-video",
		"wan2.1-vace", "hunyuan-video", "wanx", "vidu", "luma", "pika", "pixverse",
	} {
		if strings.Contains(name, marker) {
			return "文生视频"
		}
	}
	for _, marker := range []string{
		"image", "t2i", "text-to-image", "image_generation", "image-generation",
		"dall-e", "dalle", "imagen", "flux", "stable-diffusion", "midjourney",
		"ernie-vil", "kolors", "seedream", "qwen-image", "hunyuan-image",
		"cogview", "ideogram", "recraft", "jimeng",
	} {
		if strings.Contains(name, marker) {
			return "文生图"
		}
	}
	for _, marker := range []string{
		"audio", "tts", "speech", "whisper", "transcri", "music", "text-to-speech",
	} {
		if strings.Contains(name, marker) {
			return "音频"
		}
	}
	return "文生文"
}

func buildEnterpriseBillingRows(logs []*model.Log, pricingRule string) ([]EnterpriseBillingRawRow, []EnterpriseBillingCustomerRow, EnterpriseBillingSummary, error) {
	rawRows := make([]EnterpriseBillingRawRow, 0, len(logs))
	customerRows := make([]EnterpriseBillingCustomerRow, 0, len(logs))
	summary := EnterpriseBillingSummary{Currency: EnterpriseBillingCurrency}
	for _, log := range logs {
		if log == nil {
			continue
		}
		if log.PromptTokens < 0 || log.CompletionTokens < 0 {
			return nil, nil, summary, errors.New("账单记录包含负数 token，无法计算")
		}
		modelType := EnterpriseBillingModelType(log.ModelName)
		cost, _, err := billingexpr.RunExpr(pricingRule, billingexpr.TokenParams{
			P:   float64(log.PromptTokens),
			C:   float64(log.CompletionTokens),
			Len: float64(log.PromptTokens),
		})
		if err != nil {
			return nil, nil, summary, fmt.Errorf("记录 %d 算价失败: %w", log.Id, err)
		}
		if math.IsNaN(cost) || math.IsInf(cost, 0) || cost < 0 {
			return nil, nil, summary, fmt.Errorf("记录 %d 的算价结果无效", log.Id)
		}
		price := cost / 1_000_000
		if math.IsInf(price, 0) || math.IsNaN(price) || price < 0 || price > math.MaxFloat64/1_000_000 || summary.TotalPrice > math.MaxFloat64-price {
			return nil, nil, summary, errors.New("账单金额超出可计算范围")
		}
		price = math.Round(price*1_000_000) / 1_000_000
		orderID := log.RequestId
		if orderID == "" {
			orderID = strconv.Itoa(log.Id)
		}
		rawRows = append(rawRows, EnterpriseBillingRawRow{
			ID: log.Id, Time: log.CreatedAt, Type: log.Type, Username: log.Username,
			Token: log.TokenName, Model: log.ModelName, Quota: log.Quota,
			PromptTokens: log.PromptTokens, CompletionTokens: log.CompletionTokens,
			UseTime: log.UseTime, ChannelID: log.ChannelId, ChannelName: log.ChannelName,
			Group: log.Group, RequestID: log.RequestId,
		})
		customerRows = append(customerRows, EnterpriseBillingCustomerRow{
			OrderID:   orderID,
			Time:      time.Unix(log.CreatedAt, 0).In(time.Local).Format("2006-01-02 15:04:05"),
			ModelType: modelType, ModelName: log.ModelName,
			UseTime: log.UseTime, RequestID: log.RequestId, Price: price,
		})
		promptTokens := int64(log.PromptTokens)
		if summary.PromptTokens > math.MaxInt64-promptTokens {
			return nil, nil, summary, errors.New("账单输入 token 总量超出可计算范围")
		}
		completionTokens := int64(log.CompletionTokens)
		if summary.CompletionTokens > math.MaxInt64-completionTokens {
			return nil, nil, summary, errors.New("账单输出 token 总量超出可计算范围")
		}
		quota := int64(log.Quota)
		if (quota > 0 && summary.TotalQuota > math.MaxInt64-quota) ||
			(quota < 0 && summary.TotalQuota < math.MinInt64-quota) {
			return nil, nil, summary, errors.New("账单额度总量超出可计算范围")
		}
		summary.PromptTokens += promptTokens
		summary.CompletionTokens += completionTokens
		summary.TotalQuota += quota
		summary.TotalPrice += price
	}
	summary.RawCount = len(rawRows)
	summary.BilledCount = len(customerRows)
	if summary.TotalPrice > math.MaxFloat64/1_000_000 {
		return nil, nil, summary, errors.New("账单金额超出可计算范围")
	}
	summary.TotalPrice = math.Round(summary.TotalPrice*1_000_000) / 1_000_000
	if math.IsNaN(summary.TotalPrice) || math.IsInf(summary.TotalPrice, 0) {
		return nil, nil, summary, errors.New("账单金额超出可计算范围")
	}
	return rawRows, customerRows, summary, nil
}

func ReportEnterpriseBilling(accountID int, month string, page, pageSize int) (EnterpriseBillingReport, error) {
	if accountID <= 0 {
		return EnterpriseBillingReport{}, errors.New("企业账户 ID 无效")
	}
	start, end, err := ParseEnterpriseBillingMonth(month)
	if err != nil {
		return EnterpriseBillingReport{}, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = EnterpriseBillingDefaultPageSize
	}
	if pageSize > EnterpriseBillingMaxPageSize && pageSize != EnterpriseBillingMaxRows {
		pageSize = EnterpriseBillingMaxPageSize
	}
	account, err := model.GetEnterpriseBillingAccount(accountID)
	if err != nil {
		return EnterpriseBillingReport{}, err
	}
	logs, err := model.QueryEnterpriseBillingLogs(account.Source, start.Unix(), end.Unix(), account.UsernameList())
	if err != nil {
		return EnterpriseBillingReport{}, err
	}
	if len(logs) > EnterpriseBillingMaxRows {
		return EnterpriseBillingReport{}, fmt.Errorf("账单记录超过 %d 条，请缩小月份范围或拆分账号", EnterpriseBillingMaxRows)
	}
	rawRows, customerRows, summary, err := buildEnterpriseBillingRows(logs, account.PricingRule)
	if err != nil {
		return EnterpriseBillingReport{}, err
	}
	from := len(rawRows)
	if page <= (len(rawRows)/pageSize)+1 {
		from = (page - 1) * pageSize
	}
	if from > len(rawRows) {
		from = len(rawRows)
	}
	to := from + pageSize
	if to > len(rawRows) {
		to = len(rawRows)
	}
	return EnterpriseBillingReport{
		Account: accountView(account), Month: month,
		PeriodStart: start.Unix(), PeriodEnd: end.Unix(), Summary: summary,
		RawRows: rawRows[from:to], CustomerRows: customerRows[from:to],
		Page: page, PageSize: pageSize, Total: len(rawRows),
	}, nil
}

func ExportEnterpriseBillingRows(accountID int, month string) (EnterpriseBillingReport, error) {
	report, err := ReportEnterpriseBilling(accountID, month, 1, EnterpriseBillingMaxRows)
	if err != nil {
		return EnterpriseBillingReport{}, err
	}
	return report, nil
}
