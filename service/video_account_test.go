package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestNormalizeAccountModelsAddsOpaqueAccountSelectors(t *testing.T) {
	account := &model.VideoAccount{Id: 7, PublicKey: "vca_account"}
	models := normalizeAccountModels([]model.VideoModel{{ID: "seedance-1", PrivateGroupKey: "upstream-private-key", ProductKey: "upstream-product"}}, account)
	require.Len(t, models, 1)
	assert.Equal(t, "vca_account", models[0].PrivateGroupKey)
	assert.Equal(t, "vca_account:seedance-1", models[0].ProductKey)
	assert.Equal(t, []string{"openai-video"}, models[0].SupportedEndpointTypes)
	assert.Equal(t, float64(1), models[0].GroupRatio)
}
