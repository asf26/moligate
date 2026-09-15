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
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSubscriptionPlansReturnsEnabledPlansForAnonymousRequests(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)
	confirmPaymentComplianceForTest(t)

	hiddenPlan := &model.SubscriptionPlan{
		Title:         "Hidden Plan",
		PriceAmount:   10,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		SortOrder:     30,
		TotalAmount:   100,
	}
	require.NoError(t, db.Create(hiddenPlan).Error)
	require.NoError(t, db.Model(hiddenPlan).Update("enabled", false).Error)
	require.NoError(t, db.Create(&model.SubscriptionPlan{
		Title:         "Public Plan",
		PriceAmount:   20,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		SaleEnabled:   false,
		SortOrder:     20,
		TotalAmount:   200,
		ModelFamily:   "gpt",
		IncludedModels: []string{
			"gpt-5.5",
			"gpt-image-2",
		},
		BadgeText:     "Popular",
		IsRecommended: true,
		Benefits:      []string{"Priority capacity", "Extended context"},
	}).Error)

	c, w := subscriptionControllerTestContext(http.MethodGet, "/api/subscription/plans", nil)
	GetSubscriptionPlans(c)

	require.Equal(t, http.StatusOK, w.Code)
	var response struct {
		Success bool                  `json:"success"`
		Data    []SubscriptionPlanDTO `json:"data"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data, 1)
	assert.Equal(t, "Public Plan", response.Data[0].Plan.Title)
	assert.Equal(t, "gpt", response.Data[0].Plan.ModelFamily)
	assert.Equal(t, []string{"gpt-5.5", "gpt-image-2"}, response.Data[0].Plan.IncludedModels)
	assert.Equal(t, "Popular", response.Data[0].Plan.BadgeText)
	assert.False(t, response.Data[0].Plan.SaleEnabled, "preview plans remain visible in the catalog")
	assert.True(t, response.Data[0].Plan.IsRecommended)
	assert.Equal(t, []string{"Priority capacity", "Extended context"}, response.Data[0].Plan.Benefits)
}
