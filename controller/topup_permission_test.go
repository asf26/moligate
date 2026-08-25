package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTopUpPermissionControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalMainType, originalLogType := common.MainDatabaseType(), common.LogDatabaseType()
	originalCompliance := operation_setting.GetPaymentSetting().ComplianceConfirmed
	originalTermsVersion := operation_setting.GetPaymentSetting().ComplianceTermsVersion
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	operation_setting.GetPaymentSetting().ComplianceConfirmed = true
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Redemption{}, &model.Log{}))

	t.Cleanup(func() {
		model.DB, model.LOG_DB = originalDB, originalLogDB
		common.SetDatabaseTypes(originalMainType, originalLogType)
		operation_setting.GetPaymentSetting().ComplianceConfirmed = originalCompliance
		operation_setting.GetPaymentSetting().ComplianceTermsVersion = originalTermsVersion
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func topUpPermissionTestContext(method string, path string, body string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	return context, recorder
}

func decodeTopUpPermissionResponse(t *testing.T, recorder *httptest.ResponseRecorder) (bool, string) {
	t.Helper()
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response.Success, response.Message
}

func TestRequireTopUpEnabledTracksAdministratorChanges(t *testing.T) {
	db := setupTopUpPermissionControllerTestDB(t)
	user := &model.User{
		Id:           1001,
		Username:     "top-up-permission-user",
		DisplayName:  "top-up-permission-user",
		Role:         common.RoleCommonUser,
		Status:       common.UserStatusEnabled,
		TopUpEnabled: false,
	}
	require.NoError(t, db.Create(user).Error)

	context, recorder := topUpPermissionTestContext(http.MethodPost, "/api/user/pay", `{}`)
	context.Set("id", user.Id)
	assert.False(t, requireTopUpEnabled(context))
	success, message := decodeTopUpPermissionResponse(t, recorder)
	assert.False(t, success)
	assert.Equal(t, i18n.MsgUserTopUpDisabled, message)

	require.NoError(t, db.Model(&model.User{}).Where("id = ?", user.Id).Update("top_up_enabled", true).Error)
	context, recorder = topUpPermissionTestContext(http.MethodPost, "/api/user/pay", `{}`)
	context.Set("id", user.Id)
	assert.True(t, requireTopUpEnabled(context))
	assert.Empty(t, recorder.Body.String())
}

func TestTopUpAndSubscriptionPaymentRejectDisabledUser(t *testing.T) {
	db := setupTopUpPermissionControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:           1002,
		Username:     "disabled-top-up-user",
		DisplayName:  "disabled-top-up-user",
		Role:         common.RoleCommonUser,
		Status:       common.UserStatusEnabled,
		TopUpEnabled: false,
	}).Error)

	tests := []struct {
		name string
		call func(*gin.Context)
	}{
		{
			name: "redemption",
			call: TopUp,
		},
		{
			name: "balance subscription payment",
			call: SubscriptionRequestBalancePay,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context, recorder := topUpPermissionTestContext(http.MethodPost, "/api/test", `{}`)
			context.Set("id", 1002)
			test.call(context)
			success, message := decodeTopUpPermissionResponse(t, recorder)
			assert.False(t, success)
			assert.Equal(t, i18n.MsgUserTopUpDisabled, message)
		})
	}
}
