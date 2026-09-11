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
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEnterpriseBillingMonthUsesLocalHalfOpenPeriod(t *testing.T) {
	start, end, err := ParseEnterpriseBillingMonth("2026-02")
	require.NoError(t, err)

	wantStart := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.Local)
	assert.Equal(t, wantStart, start)
	assert.Equal(t, wantStart.AddDate(0, 1, 0), end)

	for _, invalid := range []string{"", "2026-2", "2026-13", "2026-02-01", "2026/02"} {
		_, _, err := ParseEnterpriseBillingMonth(invalid)
		assert.Error(t, err, invalid)
	}
}

func TestEnterpriseBillingModelTypeMapsMediaModels(t *testing.T) {
	tests := map[string]string{
		"gpt-5.5":               "文生文",
		"gpt-image-2":           "文生图",
		"doubao-seedream-4-0":   "文生图",
		"qwen-image-plus":       "文生图",
		"sora-2-pro":            "文生视频",
		"text-to-video-preview": "文生视频",
		"kling-video-v3":        "文生视频",
		"whisper-1":             "音频",
		"imagen-4-preview":      "文生图",
	}
	for modelName, want := range tests {
		assert.Equal(t, want, EnterpriseBillingModelType(modelName), modelName)
	}
}

func TestBuildEnterpriseBillingRowsCalculatesCustomerInvoice(t *testing.T) {
	createdAt := time.Date(2026, time.March, 5, 14, 6, 7, 0, time.Local).Unix()
	logs := []*model.Log{
		{
			Id: 42, CreatedAt: createdAt, Type: model.LogTypeConsume,
			Username: "alice", TokenName: "team-key", ModelName: "gpt-5.5",
			Quota: 900, PromptTokens: 1000, CompletionTokens: 500, UseTime: 3,
			ChannelId: 7, ChannelName: "OpenAI", Group: "team", RequestId: "req-42",
		},
		{
			Id: 43, CreatedAt: createdAt + 1, Type: model.LogTypeConsume,
			Username: "bob", TokenName: "team-key", ModelName: "gpt-image-2",
			Quota: 1200, PromptTokens: 100, CompletionTokens: 50, UseTime: 8,
		},
	}

	rawRows, customerRows, summary, err := buildEnterpriseBillingRows(logs, "p * 2 + c * 10")
	require.NoError(t, err)
	require.Len(t, rawRows, 2)
	require.Len(t, customerRows, 2)

	assert.Equal(t, 42, rawRows[0].ID)
	assert.Equal(t, "req-42", customerRows[0].OrderID)
	assert.Equal(t, "2026-03-05 14:06:07", customerRows[0].Time)
	assert.Equal(t, "文生文", customerRows[0].ModelType)
	assert.InDelta(t, 0.007, customerRows[0].Price, 1e-9)
	assert.Equal(t, "43", customerRows[1].OrderID)
	assert.Equal(t, "文生图", customerRows[1].ModelType)
	assert.InDelta(t, 0.0007, customerRows[1].Price, 1e-9)
	assert.Equal(t, 2, summary.RawCount)
	assert.Equal(t, int64(1100), summary.PromptTokens)
	assert.Equal(t, int64(550), summary.CompletionTokens)
	assert.Equal(t, int64(2100), summary.TotalQuota)
	assert.InDelta(t, 0.0077, summary.TotalPrice, 1e-9)
}

func TestBuildEnterpriseBillingRowsRejectsNegativeTokens(t *testing.T) {
	_, _, _, err := buildEnterpriseBillingRows([]*model.Log{{
		Id: 1, PromptTokens: -1, CompletionTokens: 0,
	}}, "p + c")
	require.Error(t, err)
}

func TestValidateEnterpriseBillingInputRequiresUsageAccounts(t *testing.T) {
	_, err := validateEnterpriseBillingInput(EnterpriseBillingAccountInput{
		Name: "Acme", Code: "acme", Source: model.EnterpriseBillingSourceLocal,
		PricingRule: "p + c", Usernames: nil,
	})
	require.Error(t, err)
}
