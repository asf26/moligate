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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionPlanAddedModelsKeepsOnlyAdditions(t *testing.T) {
	added := SubscriptionPlanAddedModels(
		[]string{"gpt-5.4", "gpt-5.5", " gpt-5.6 "},
		[]string{"gpt-5.4", "gpt-6-sol", "", "gpt-6-sol", " gpt-5.6 ", "gpt-6.1-sol"},
	)

	// gpt-5.5 was removed from the plan, which must not remove it from buyers.
	assert.Equal(t, []string{"gpt-6-sol", "gpt-6.1-sol"}, added)
}

func TestExtendActiveSubscriptionsWithPlanModels(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	const planId = 9301
	otherPlanId := 9302

	seed := func(sub *UserSubscription) {
		t.Helper()
		sub.Status = "active"
		require.NoError(t, DB.Create(sub).Error)
	}
	// Active subscription bought before the new models existed.
	seed(&UserSubscription{Id: 9201, UserId: 1, PlanId: planId, EndTime: now + 3600, AmountTotal: 100,
		IncludedModels: []string{"gpt-5.4", "gpt-5.5"}})
	// Active subscription without a snapshot: the family matcher already covers additions.
	seed(&UserSubscription{Id: 9202, UserId: 2, PlanId: planId, EndTime: now + 3600, AmountTotal: 100,
		IncludedModels: nil, ModelFamily: "gpt"})
	// Expired and cancelled subscriptions must stay frozen.
	seed(&UserSubscription{Id: 9203, UserId: 3, PlanId: planId, EndTime: now - 3600, AmountTotal: 100,
		IncludedModels: []string{"gpt-5.4"}})
	seed(&UserSubscription{Id: 9204, UserId: 4, PlanId: planId, EndTime: now + 3600, AmountTotal: 100,
		IncludedModels: []string{"gpt-5.4"}})
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", 9204).Update("status", "cancelled").Error)
	// Another plan's subscription must not be touched.
	seed(&UserSubscription{Id: 9205, UserId: 5, PlanId: otherPlanId, EndTime: now + 3600, AmountTotal: 100,
		IncludedModels: []string{"gpt-5.4"}})

	added := []string{"gpt-6-sol", "gpt-6.1-sol"}
	extended, err := ExtendActiveSubscriptionsWithPlanModels(planId, added)
	require.NoError(t, err)
	assert.Equal(t, 1, extended)

	var extended9001 UserSubscription
	require.NoError(t, DB.Where("id = ?", 9201).First(&extended9001).Error)
	assert.Equal(t, []string{"gpt-5.4", "gpt-5.5", "gpt-6-sol", "gpt-6.1-sol"}, extended9001.IncludedModels)

	// The family subscription, the expired one and the other plan stay as they were.
	var familySub UserSubscription
	require.NoError(t, DB.Where("id = ?", 9202).First(&familySub).Error)
	assert.Empty(t, familySub.IncludedModels)
	var expired UserSubscription
	require.NoError(t, DB.Where("id = ?", 9203).First(&expired).Error)
	assert.Equal(t, []string{"gpt-5.4"}, expired.IncludedModels)
	var foreign UserSubscription
	require.NoError(t, DB.Where("id = ?", 9205).First(&foreign).Error)
	assert.Equal(t, []string{"gpt-5.4"}, foreign.IncludedModels)

	// Running the same update again changes nothing.
	again, err := ExtendActiveSubscriptionsWithPlanModels(planId, added)
	require.NoError(t, err)
	assert.Zero(t, again)
}
