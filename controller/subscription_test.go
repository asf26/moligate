package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSubscriptionControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()
	t.Cleanup(func() {
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
	})

	gin.SetMode(gin.TestMode)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.SubscriptionPlan{}))

	return db
}

func subscriptionControllerTestContext(method string, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c, w
}

func subscriptionPlanPayload(t *testing.T, currency string) []byte {
	t.Helper()
	body, err := common.Marshal(map[string]any{
		"plan": map[string]any{
			"title":                 "Test Plan",
			"subtitle":              "Test",
			"price_amount":          10,
			"currency":              currency,
			"duration_unit":         model.SubscriptionDurationDay,
			"duration_value":        1,
			"enabled":               true,
			"sort_order":            0,
			"max_purchase_per_user": 0,
			"total_amount":          1000,
			"daily_amount":          100,
			"weekly_amount":         500,
			"monthly_amount":        2000,
			"quota_reset_period":    model.SubscriptionResetNever,
			"applicable_groups":     []string{"default"},
		},
	})
	require.NoError(t, err)
	return body
}

func TestAdminCreateSubscriptionPlanForcesCNY(t *testing.T) {
	setupSubscriptionControllerTestDB(t)
	confirmPaymentComplianceForTest(t)

	c, w := subscriptionControllerTestContext(http.MethodPost, "/api/subscription/admin/plans", subscriptionPlanPayload(t, "USD"))
	AdminCreateSubscriptionPlan(c)
	require.Equal(t, http.StatusOK, w.Code)

	var plan model.SubscriptionPlan
	require.NoError(t, model.DB.First(&plan).Error)
	assert.Equal(t, model.SubscriptionCurrencyCNY, plan.Currency)
	assert.EqualValues(t, 100, plan.DailyAmount)
	assert.EqualValues(t, 500, plan.WeeklyAmount)
	assert.EqualValues(t, 2000, plan.MonthlyAmount)
	assert.Equal(t, []string{"default"}, plan.ApplicableGroups)
}

func TestAdminUpdateSubscriptionPlanForcesCNY(t *testing.T) {
	setupSubscriptionControllerTestDB(t)
	confirmPaymentComplianceForTest(t)

	plan := &model.SubscriptionPlan{
		Title:         "Legacy USD Plan",
		PriceAmount:   10,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationDay,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   1000,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	c, w := subscriptionControllerTestContext(http.MethodPut, "/api/subscription/admin/plans/1", subscriptionPlanPayload(t, "USD"))
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	AdminUpdateSubscriptionPlan(c)
	require.Equal(t, http.StatusOK, w.Code)

	var updated model.SubscriptionPlan
	require.NoError(t, model.DB.First(&updated, plan.Id).Error)
	assert.Equal(t, model.SubscriptionCurrencyCNY, updated.Currency)
	assert.EqualValues(t, 100, updated.DailyAmount)
	assert.EqualValues(t, 500, updated.WeeklyAmount)
	assert.EqualValues(t, 2000, updated.MonthlyAmount)
	assert.Equal(t, []string{"default"}, updated.ApplicableGroups)
}

func TestAdminCreateSubscriptionPlanRejectsNegativePeriodQuota(t *testing.T) {
	setupSubscriptionControllerTestDB(t)
	confirmPaymentComplianceForTest(t)

	body, err := common.Marshal(map[string]any{
		"plan": map[string]any{
			"title":              "Invalid Plan",
			"price_amount":       10,
			"duration_unit":      model.SubscriptionDurationDay,
			"duration_value":     1,
			"daily_amount":       -1,
			"weekly_amount":      500,
			"monthly_amount":     2000,
			"quota_reset_period": model.SubscriptionResetNever,
		},
	})
	require.NoError(t, err)

	c, w := subscriptionControllerTestContext(http.MethodPost, "/api/subscription/admin/plans", body)
	AdminCreateSubscriptionPlan(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]any
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, false, response["success"])

	var count int64
	require.NoError(t, model.DB.Model(&model.SubscriptionPlan{}).Count(&count).Error)
	assert.Zero(t, count)
}
