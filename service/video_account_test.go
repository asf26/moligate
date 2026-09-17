package service

import (
	"context"
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

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestFetchVideoAccountModelsUsesBearerAndDefaultsMissingCapabilities(t *testing.T) {
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"minimax-h3-01","pricing":{"mode":"per_second","amount":"0.02"},"durations_seconds":[6]}]}`))
	}))
	defer server.Close()

	originalBaseURL := videoAccountBaseURL
	videoAccountBaseURL = server.URL
	t.Cleanup(func() { videoAccountBaseURL = originalBaseURL })

	models, err := FetchVideoAccountModels(context.Background(), "sk-secret", "")
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "Bearer sk-secret", gotAuthorization)
	assert.Equal(t, "minimax-h3-01", models[0].ID)
	assert.Equal(t, "minimax-h3", models[0].Group)
	assert.True(t, models[0].Available)
	assert.Equal(t, []string{"openai-video"}, models[0].SupportedEndpointTypes)
	assert.Equal(t, -1, models[0].MaxImages)
	assert.Equal(t, -1, models[0].MaxVideos)
	assert.Equal(t, -1, models[0].MaxAudios)
	assert.Equal(t, "per_second", models[0].Pricing.Mode)
	assert.InDelta(t, 0.02, models[0].Pricing.Amount, 0.000001)
	encoded, err := common.Marshal(models[0])
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"max_images":-1`)
}

func setupVideoCreationCatalogTest(t *testing.T) *gorm.DB {
	t.Helper()
	originalDB := model.DB
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.VideoAccount{}))
	model.DB = db
	t.Cleanup(func() {
		model.DB = originalDB
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// A dedicated account authorizes by group, and the group in force is the one
// the API key selected. A key bound to another group must not see the account,
// which is what keeps the catalog scoped to the key rather than to the user.
func TestBuildVideoCreationCatalogAuthorizesByRequestGroup(t *testing.T) {
	db := setupVideoCreationCatalogTest(t)
	account := &model.VideoAccount{Name: "h3", ApiKey: "secret", PublicKey: "vca_h3", Groups: "视频-h3", Status: model.VideoAccountStatusEnabled}
	require.NoError(t, account.SetModelCatalog([]model.VideoModel{{ID: "minimax-h3-01", Available: true}}))
	require.NoError(t, db.Create(account).Error)

	catalog, err := BuildVideoCreationCatalog([]string{"视频-h3"})
	require.NoError(t, err)
	require.Len(t, catalog.PrivateGroups, 1)
	assert.Equal(t, "h3", catalog.PrivateGroups[0].Name)
	assert.Equal(t, "vca_h3", catalog.PrivateGroups[0].Key)
	require.Len(t, catalog.Models, 1)
	assert.Equal(t, "minimax-h3-01", catalog.Models[0].ID)

	// A key that selected a different group must not inherit the account.
	catalog, err = BuildVideoCreationCatalog([]string{"default"})
	require.NoError(t, err)
	assert.Empty(t, catalog.PrivateGroups)
	assert.Empty(t, catalog.Models)
}

func TestGetVideoAccountModelsForGroupRejectsUnauthorizedGroups(t *testing.T) {
	db := setupVideoCreationCatalogTest(t)
	account := &model.VideoAccount{Name: "h3", ApiKey: "secret", PublicKey: "vca_h3", Groups: "视频-h3", Status: model.VideoAccountStatusEnabled}
	require.NoError(t, account.SetModelCatalog([]model.VideoModel{{ID: "minimax-h3-01", Available: true}}))
	require.NoError(t, db.Create(account).Error)

	models, err := GetVideoAccountModelsForGroup([]string{"视频-h3"}, "vca_h3")
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "minimax-h3-01", models[0].ID)

	// The account selector alone must not grant access: the requesting group
	// set has to cover a group the account authorizes.
	_, err = GetVideoAccountModelsForGroup([]string{"default"}, "vca_h3")
	require.Error(t, err)

	// A disabled account is never usable regardless of the group.
	account.Status = model.VideoAccountStatusDisabled
	require.NoError(t, db.Save(account).Error)
	_, err = GetVideoAccountModelsForGroup([]string{"视频-h3"}, "vca_h3")
	require.Error(t, err)
}

// The relay and the creation catalog must resolve the same group set, including
// the concrete groups an "auto" key expands to, otherwise the workspace would
// offer a model that the relay then refuses.
func TestVideoAccountRequestGroupsExpandsAutoKeys(t *testing.T) {
	configureRequestAutoGroupsTest(t)
	// The dedicated video group has to be selectable for an Auto key to expand
	// onto it, exactly as it would be in production.
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP","svip":"SVIP","视频-h3":"H3"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1,"svip":1,"视频-h3":1}`))

	concrete := newRequestAutoGroupsContext()
	common.SetContextKey(concrete, constant.ContextKeyUsingGroup, "视频-h3")
	common.SetContextKey(concrete, constant.ContextKeyUserGroup, "default")
	assert.Equal(t, []string{"视频-h3"}, VideoAccountRequestGroups(concrete))

	auto := newRequestAutoGroupsContext()
	common.SetContextKey(auto, constant.ContextKeyUsingGroup, "auto")
	common.SetContextKey(auto, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(auto, constant.ContextKeyTokenAutoGroups, []string{"vip", "视频-h3"})
	assert.Equal(t, []string{"vip", "视频-h3"}, VideoAccountRequestGroups(auto))

	// A dashboard session has no key, so the user's own group is used.
	session := newRequestAutoGroupsContext()
	common.SetContextKey(session, constant.ContextKeyUserGroup, "vip")
	assert.Equal(t, []string{"vip"}, VideoAccountRequestGroups(session))

	// An unresolved caller matches nothing rather than inheriting every account.
	assert.Empty(t, VideoAccountRequestGroups(newRequestAutoGroupsContext()))
}

func TestNormalizeAccountModelsAddsOpaqueAccountSelectors(t *testing.T) {
	account := &model.VideoAccount{Id: 7, PublicKey: "vca_account"}
	models := normalizeAccountModels([]model.VideoModel{{ID: "seedance-1", PrivateGroupKey: "upstream-private-key", ProductKey: "upstream-product"}}, account)
	require.Len(t, models, 1)
	assert.Equal(t, "vca_account", models[0].PrivateGroupKey)
	assert.Equal(t, "vca_account:seedance-1", models[0].ProductKey)
	assert.Equal(t, []string{"openai-video"}, models[0].SupportedEndpointTypes)
	assert.Equal(t, float64(1), models[0].GroupRatio)
}

func TestNormalizeAccountModelsPreservesBillingOverridesAcrossSync(t *testing.T) {
	account := &model.VideoAccount{Id: 7, PublicKey: "vca_account"}
	override := model.VideoModelPricing{Amount: 0.4, Currency: "USD", Mode: "per_second"}
	require.NoError(t, account.SetModelCatalog([]model.VideoModel{{
		ID: "seedance-1", BillingPricing: &override,
	}}))
	models := normalizeAccountModels([]model.VideoModel{{
		ID: "seedance-1", Pricing: model.VideoModelPricing{Amount: 0.2, Currency: "CNY", Mode: "per_second"},
	}}, account)
	require.Len(t, models, 1)
	require.NotNil(t, models[0].BillingPricing)
	assert.InDelta(t, 0.4, models[0].BillingPricing.Amount, 0.000001)
	assert.Equal(t, "USD", models[0].BillingPricing.Currency)
}
