package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var h3Ratios = []string{"16:9", "9:16", "1:1", "2:3", "3:2", "3:4", "4:3", "21:9"}

func TestMiniMaxH3SizesFollowTheDocumentedRatioTable(t *testing.T) {
	item := VideoModel{ID: "minimax-h3-original-768p", Group: "minimax-h3", Resolution: "768p", Ratios: h3Ratios}

	require.Equal(t, map[string]string{
		"16:9": "1376x768",
		"9:16": "768x1376",
		"1:1":  "1024x1024",
		"2:3":  "832x1248",
		"3:2":  "1248x832",
		"3:4":  "896x1184",
		"4:3":  "1184x896",
		"21:9": "1568x672",
	}, item.RatioSizes())

	// Sizes must stay usable by the relay's size validation and stay aligned
	// with the ratio order the catalog publishes.
	require.Equal(t, []string{
		"1376x768", "768x1376", "1024x1024", "832x1248",
		"1248x832", "896x1184", "1184x896", "1568x672",
	}, item.WithDerivedCapabilities().Sizes)
}

func TestMiniMaxH3SizesUseThe1080pColumn(t *testing.T) {
	item := VideoModel{ID: "minimax-h3-original-1080p", Group: "minimax-h3", Resolution: "1080p", Ratios: h3Ratios}

	require.Equal(t, "1920x1088", item.RatioSizes()["16:9"])
	require.Equal(t, "2208x960", item.RatioSizes()["21:9"])
}

func TestMiniMaxH3CFModelsSubmitTheResolutionLabel(t *testing.T) {
	for _, tc := range []struct{ resolution, want string }{{"2K", "2K"}, {"4K", "4K"}, {"2k", "2K"}, {"4k", "4K"}} {
		item := VideoModel{ID: "minimax-h3-comic-cf-4k", Group: "minimax-h3", Resolution: tc.resolution, Ratios: h3Ratios}
		sizes := item.RatioSizes()
		require.Len(t, sizes, len(h3Ratios))
		for _, ratio := range h3Ratios {
			require.Equal(t, tc.want, sizes[ratio], "resolution %s ratio %s", tc.resolution, ratio)
		}
		require.Equal(t, []string{tc.want}, item.WithDerivedCapabilities().Sizes, "duplicates must collapse")
	}
}

func TestMiniMaxH3FirstLastFrameIsEnabled(t *testing.T) {
	h3 := VideoModel{ID: "minimax-h3-original-768p", Group: "minimax-h3"}
	require.True(t, h3.WithDerivedCapabilities().SupportsFirstLastFrame)
}

// Both integrations accept the frame workflow: the H3 doc documents
// workflow_id=fl2v, and CTMOAI's console offers the mode for the Seedance
// models. The capability flag only exists on an endpoint that needs a console
// session, so the gateway enables it by default rather than hiding a real
// feature behind a field it can never read.
func TestFirstLastFrameIsEnabledForEveryCatalogModel(t *testing.T) {
	for _, item := range []VideoModel{
		{ID: "minimax-h3-original-768p", Group: "minimax-h3"},
		{ID: "minimax-h3-comic-cf-4k", Group: "minimax-h3"},
		{ID: "sd-2-vip-480", Group: "video"},
		{ID: "seedance2.0-select-full-720p", Group: "video"},
		{ID: "seedance2.5-stable-480p", Group: "video"},
	} {
		require.True(t, item.WithDerivedCapabilities().SupportsFirstLastFrame, item.ID)
	}
}

func TestAdministratorOverrideWinsOverTheDerivedDefault(t *testing.T) {
	disable := false
	enable := true

	off := VideoModel{ID: "sd-2-vip-480", Group: "video", SupportsFirstLastFrameOverride: &disable}
	require.False(t, off.WithDerivedCapabilities().SupportsFirstLastFrame)

	// An override also survives on a model the default already enables.
	on := VideoModel{ID: "minimax-h3-original-768p", Group: "minimax-h3", SupportsFirstLastFrameOverride: &enable}
	require.True(t, on.WithDerivedCapabilities().SupportsFirstLastFrame)

	// A nil override follows the default.
	require.True(t, VideoModel{ID: "sd-2-vip-480", Group: "video"}.WithDerivedCapabilities().SupportsFirstLastFrame)
}

// The two integrations share an endpoint but not a request dialect, so callers
// that must pick a field spelling read the family instead of guessing.
func TestFamilySeparatesTheTwoIntegrations(t *testing.T) {
	require.Equal(t, VideoFamilyMiniMaxH3, VideoModel{ID: "minimax-h3-original-768p", Group: "minimax-h3"}.Family())
	require.Equal(t, VideoFamilyMiniMaxH3, VideoModel{ID: "minimax-h3-comic-cf-4k", Group: "minimax-h3"}.Family())
	require.Equal(t, VideoFamilySeedance, VideoModel{ID: "sd-2-vip-480", Group: "video"}.Family())
	require.Equal(t, VideoFamilySeedance, VideoModel{ID: "seedance2.5-stable-480p", Group: "video"}.Family())
	// A catalog that matches neither pattern must not inherit the H3 dialect.
	require.Equal(t, VideoFamilySeedance, VideoModel{ID: "unknown-video-model", Group: "misc"}.Family())
}

func TestRequiresReferenceImageOnlyCoversTheCFVariants(t *testing.T) {
	require.True(t, VideoModel{ID: "minimax-h3-comic-cf-4k", Group: "minimax-h3"}.RequiresReferenceImage())
	require.True(t, VideoModel{ID: "minimax-h3-original-cf-2k", Group: "minimax-h3"}.RequiresReferenceImage())
	require.False(t, VideoModel{ID: "minimax-h3-original-768p", Group: "minimax-h3"}.RequiresReferenceImage())
	require.False(t, VideoModel{ID: "minimax-h3-quantized-768p", Group: "minimax-h3"}.RequiresReferenceImage())
	// None of the Seedance models are CF variants.
	require.False(t, VideoModel{ID: "sd-2-vip-480", Group: "video"}.RequiresReferenceImage())
	require.False(t, VideoModel{ID: "seedance2.0-select-sdas-full-720p", Group: "video"}.RequiresReferenceImage())
}

func TestSeedanceModelsKeepTheirOwnDialect(t *testing.T) {
	for _, item := range []VideoModel{
		{ID: "sd-2-vip-480", Group: "video", Resolution: "480p", Ratios: []string{"9:16", "16:9"}},
		{ID: "seedance2.5-stable-480p", Group: "video", Resolution: "480p", Ratios: []string{"16:9"}},
	} {
		require.False(t, IsMiniMaxH3VideoModel(item.Group, item.ID), item.ID)
		// Seedance has no size field, so nothing may be derived for it.
		require.Nil(t, item.RatioSizes(), item.ID)
		require.Empty(t, item.WithDerivedCapabilities().Sizes, item.ID)
	}
}

func TestUnknownResolutionDerivesNoSizes(t *testing.T) {
	item := VideoModel{ID: "minimax-h3-future-4k-hdr", Group: "minimax-h3", Resolution: "8K", Ratios: h3Ratios}
	require.Nil(t, item.RatioSizes())
	require.Empty(t, item.WithDerivedCapabilities().Sizes)
}

func TestUpstreamSizesWin(t *testing.T) {
	item := VideoModel{
		ID:         "minimax-h3-original-768p",
		Group:      "minimax-h3",
		Resolution: "768p",
		Ratios:     h3Ratios,
		Sizes:      []string{"1000x1000"},
	}
	require.Equal(t, []string{"1000x1000"}, item.WithDerivedCapabilities().Sizes)
}

func TestModelCatalogDerivesCapabilities(t *testing.T) {
	account := &VideoAccount{Id: 1, Name: "h3", Groups: "视频-H3"}
	require.NoError(t, account.SetModelCatalog([]VideoModel{
		{ID: "minimax-h3-original-768p", Group: "minimax-h3", Resolution: "768p", Ratios: h3Ratios, Available: true},
		{ID: "sd-2-vip-480", Group: "video", Resolution: "480p", Ratios: []string{"9:16", "16:9"}, Available: true},
	}))

	catalog := account.ModelCatalog()
	require.Len(t, catalog, 2)

	h3, ok := account.FindModel("minimax-h3-original-768p")
	require.True(t, ok)
	require.Equal(t, "1376x768", h3.Sizes[0])
	require.True(t, h3.SupportsFirstLastFrame)

	sd, ok := account.FindModel("sd-2-vip-480")
	require.True(t, ok)
	// Seedance takes no size, but it does accept the frame workflow.
	require.Empty(t, sd.Sizes)
	require.True(t, sd.SupportsFirstLastFrame)
}

func TestSetModelCapabilityOverridesSetsAndClears(t *testing.T) {
	account := &VideoAccount{Id: 1, Name: "sd", Groups: "视频"}
	require.NoError(t, account.SetModelCatalog([]VideoModel{{ID: "sd-2-vip-480", Group: "video", Available: true}}))

	disable := false
	require.NoError(t, account.SetModelCapabilityOverrides(map[string]*VideoModelCapabilityOverride{
		"SD-2-VIP-480": {SupportsFirstLastFrame: &disable},
	}))
	item, ok := account.FindModel("sd-2-vip-480")
	require.True(t, ok)
	require.False(t, item.SupportsFirstLastFrame)

	// A nil override clears the decision and returns the model to the default.
	require.NoError(t, account.SetModelCapabilityOverrides(map[string]*VideoModelCapabilityOverride{"sd-2-vip-480": nil}))
	item, _ = account.FindModel("sd-2-vip-480")
	require.True(t, item.SupportsFirstLastFrame)

	// An unknown model is rejected rather than silently ignored.
	err := account.SetModelCapabilityOverrides(map[string]*VideoModelCapabilityOverride{"nope": {}})
	require.Error(t, err)
}
