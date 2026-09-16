package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVideoAccountStoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))), &gorm.Config{})
	require.NoError(t, err)
	previous := DB
	DB = db
	require.NoError(t, db.AutoMigrate(&VideoAccount{}))
	t.Cleanup(func() {
		DB = previous
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestCreateVideoAccountGeneratesOpaqueTokenAndMasksSecret(t *testing.T) {
	db := setupVideoAccountStoreTestDB(t)
	account := &VideoAccount{Name: "ctmoai", ApiKey: "sk-video-secret", Groups: "vip,vip", Status: VideoAccountStatusEnabled}
	require.NoError(t, CreateVideoAccount(account))
	assert.NotZero(t, account.Id)
	assert.True(t, strings.HasPrefix(account.PublicKey, "vca_"))
	assert.Equal(t, "vip", account.Groups)
	view := account.PublicView()
	_, hasRawKey := view["api_key"]
	assert.False(t, hasRawKey)
	assert.Equal(t, "sk-v••••cret", view["api_key_masked"])

	var stored VideoAccount
	require.NoError(t, db.First(&stored, account.Id).Error)
	assert.Equal(t, "sk-video-secret", stored.ApiKey)
}

func TestFindVideoAccountForModelHonorsOpaqueKeyAndGroup(t *testing.T) {
	db := setupVideoAccountStoreTestDB(t)
	account := &VideoAccount{Name: "ctmoai", ApiKey: "secret", PublicKey: "vca_test", Groups: "vip", Status: VideoAccountStatusEnabled}
	require.NoError(t, account.SetModelCatalog([]VideoModel{{ID: "seedance-1", Available: true}}))
	require.NoError(t, db.Create(account).Error)

	selected, err := FindVideoAccountForModel("seedance-1", "vip", "vca_test")
	require.NoError(t, err)
	assert.Equal(t, account.Id, selected.Id)
	_, err = FindVideoAccountForModel("seedance-1", "default", "vca_test")
	assert.Error(t, err)
}

func TestFindVideoAccountForModelRejectsUnavailableSnapshot(t *testing.T) {
	db := setupVideoAccountStoreTestDB(t)
	account := &VideoAccount{Name: "ctmoai", ApiKey: "secret", PublicKey: "vca_unavailable", Groups: "default", Status: VideoAccountStatusEnabled}
	require.NoError(t, account.SetModelCatalog([]VideoModel{{ID: "seedance-1", Available: false}}))
	require.NoError(t, db.Create(account).Error)
	_, err := FindVideoAccountForModel("seedance-1", "default", "vca_unavailable")
	assert.ErrorContains(t, err, "currently unavailable")
}

func TestVideoModelEffectivePricingUsesBillingOverride(t *testing.T) {
	modelItem := VideoModel{
		ID:      "seedance-1",
		Pricing: VideoModelPricing{Amount: 0.2, Currency: "CNY", Mode: "per_second"},
	}
	assert.InDelta(t, 0.2, modelItem.EffectivePricing().Amount, 0.000001)
	modelItem.BillingPricing = &VideoModelPricing{Amount: 0.35, Currency: "USD", Mode: "per_second"}
	assert.InDelta(t, 0.35, modelItem.EffectivePricing().Amount, 0.000001)
}

func TestSetModelBillingPricesCanSetAndClearOverrides(t *testing.T) {
	account := &VideoAccount{}
	require.NoError(t, account.SetModelCatalog([]VideoModel{{
		ID:      "seedance-1",
		Pricing: VideoModelPricing{Amount: 0.2, Currency: "CNY", Mode: "per_second"},
	}}))
	price := VideoModelPricing{Amount: 0.35, Currency: "USD"}
	require.NoError(t, account.SetModelBillingPrices(map[string]*VideoModelPricing{"SEEDANCE-1": &price}))
	item, ok := account.FindModel("seedance-1")
	require.True(t, ok)
	require.NotNil(t, item.BillingPricing)
	assert.InDelta(t, 0.35, item.EffectivePricing().Amount, 0.000001)
	assert.Equal(t, "per_second", item.BillingPricing.Mode)

	require.NoError(t, account.SetModelBillingPrices(map[string]*VideoModelPricing{"seedance-1": nil}))
	item, ok = account.FindModel("seedance-1")
	require.True(t, ok)
	assert.Nil(t, item.BillingPricing)
}
