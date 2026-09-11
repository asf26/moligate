package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCanvasControllerTest(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousMemoryCache := common.MemoryCacheEnabled
	previousRedis := common.RedisEnabled
	previousMainDB := common.MainDatabaseType()
	previousLogDBType := common.LogDatabaseType()
	previousGroups := setting.UserUsableGroups2JSONString()
	previousRatios := ratio_setting.GroupRatio2JSONString()
	initModelListColumnNames(t)
	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.MemoryCacheEnabled = previousMemoryCache
		common.RedisEnabled = previousRedis
		common.SetDatabaseTypes(previousMainDB, previousLogDBType)
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(previousGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
		model.InvalidatePricingCache()
	})

	gin.SetMode(gin.TestMode)
	common.MemoryCacheEnabled = false
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"默认分组","vip":"VIP 分组"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1}`))

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.Channel{}, &model.Ability{}, &model.Model{}, &model.Vendor{}))
	model.InvalidatePricingCache()
	return db
}

func TestGetCanvasConfigReturnsEligibleUserKeysWithoutBaseURL(t *testing.T) {
	db := setupCanvasControllerTest(t)
	require.NoError(t, db.Create(&model.User{Id: 71, Username: "canvas-user", Group: "default", Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&[]model.Channel{
		{Id: 81, Name: "image", Type: constant.ChannelTypeOpenAI, Key: "image-upstream", Status: common.ChannelStatusEnabled, Group: "default", Models: "gpt-image-2,gpt-5.4"},
		{Id: 82, Name: "video", Type: constant.ChannelTypeNewAPI, Key: "video-upstream", Status: common.ChannelStatusEnabled, Group: "default", Models: "seedance2.0-stable-full-720p"},
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "gpt-image-2", ChannelId: 81, Enabled: true},
		{Group: "default", Model: "gpt-5.4", ChannelId: 81, Enabled: true},
		{Group: "default", Model: "seedance2.0-stable-full-720p", ChannelId: 82, Enabled: true},
	}).Error)
	require.NoError(t, db.Create(&[]model.Token{
		{Id: 91, UserId: 71, Name: "image-key", Key: "actual-image-key", Status: common.TokenStatusEnabled, Group: "default", ModelLimitsEnabled: true, ModelLimits: "gpt-image-2"},
		{Id: 92, UserId: 71, Name: "text-only", Key: "text-only-key", Status: common.TokenStatusEnabled, Group: "default", ModelLimitsEnabled: true, ModelLimits: "gpt-5.4"},
		{Id: 93, UserId: 71, Name: "disabled-image", Key: "disabled-image-key", Status: common.TokenStatusDisabled, Group: "default", ModelLimitsEnabled: true, ModelLimits: "gpt-image-2"},
		{Id: 94, UserId: 71, Name: "infinite-canvas:default", Key: "legacy-internal-key", Status: common.TokenStatusEnabled, Group: "default", ModelLimitsEnabled: true, ModelLimits: "gpt-image-2"},
	}).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/canvas/config", nil)
	context.Set("id", 71)
	GetCanvasConfig(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "base_url")
	var response struct {
		Success bool                 `json:"success"`
		Data    canvasConfigResponse `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data.Groups, 2)
	assert.Equal(t, "token-91", response.Data.Groups[0].ID)
	assert.Equal(t, "image-key · 默认分组", response.Data.Groups[0].Name)
	assert.Equal(t, "sk-actual-image-key", response.Data.Groups[0].APIKey)
	assert.Equal(t, 91, response.Data.Groups[0].KeyID)
	assert.Equal(t, "image-key", response.Data.Groups[0].KeyName)
	assert.Equal(t, "default", response.Data.Groups[0].GroupID)
	assert.Equal(t, "默认分组", response.Data.Groups[0].GroupName)
	require.Len(t, response.Data.Groups[0].Models, 1)
	assert.Equal(t, "gpt-image-2", response.Data.Groups[0].Models[0].Name)
	assert.Equal(t, "image", response.Data.Groups[0].Models[0].Capability)
	assert.Equal(t, "token-92", response.Data.Groups[1].ID)
	assert.Equal(t, "text-only · 默认分组", response.Data.Groups[1].Name)
	assert.Empty(t, response.Data.Groups[1].Models)

	var tokenCount int64
	require.NoError(t, db.Model(&model.Token{}).Where("user_id = ?", 71).Count(&tokenCount).Error)
	assert.Equal(t, int64(4), tokenCount)
}
