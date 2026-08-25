package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTopUpPermissionMigrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&User{}, &Option{}))
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestMigrateEnableExistingUserTopUpIsOneTime(t *testing.T) {
	db := setupTopUpPermissionMigrationDB(t)
	legacy := User{Username: "legacy-top-up", Password: "password", AffCode: "legacy-top-up", Status: common.UserStatusEnabled}
	disabled := User{Username: "disabled-top-up", Password: "password", AffCode: "disabled-top-up", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&legacy).Error)
	require.NoError(t, db.Create(&disabled).Error)

	require.NoError(t, migrateEnableExistingUserTopUp())
	var users []User
	require.NoError(t, db.Order("id").Find(&users).Error)
	require.Len(t, users, 2)
	assert.True(t, users[0].TopUpEnabled)
	assert.True(t, users[1].TopUpEnabled)

	// An administrator can revoke the permission after migration. A rerun must
	// observe the marker and leave that decision untouched.
	require.NoError(t, db.Model(&User{}).Where("id = ?", disabled.Id).Update("top_up_enabled", false).Error)
	require.NoError(t, migrateEnableExistingUserTopUp())
	var reloaded User
	require.NoError(t, db.First(&reloaded, disabled.Id).Error)
	assert.False(t, reloaded.TopUpEnabled)

	var markerCount int64
	require.NoError(t, db.Model(&Option{}).Where("key = ?", enableExistingUserTopUpMigrationKey).Count(&markerCount).Error)
	assert.EqualValues(t, 1, markerCount)
}

func TestInsertCreatesUserWithTopUpDisabled(t *testing.T) {
	db := setupTopUpPermissionMigrationDB(t)
	// Mark the migration complete to model a normal post-upgrade startup.
	require.NoError(t, db.Create(&Option{Key: enableExistingUserTopUpMigrationKey, Value: "done"}).Error)

	user := &User{Username: "new-top-up", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled}
	require.NoError(t, user.Insert(0))
	var stored User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.False(t, stored.TopUpEnabled)
}
