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

func TestSeedanceModelsKeepTheirOwnDialect(t *testing.T) {
	for _, item := range []VideoModel{
		{ID: "sd-2-vip-480", Group: "video", Resolution: "480p", Ratios: []string{"9:16", "16:9"}},
		{ID: "seedance2.5-stable-480p", Group: "video", Resolution: "480p", Ratios: []string{"16:9"}},
	} {
		require.False(t, IsMiniMaxH3VideoModel(item.Group, item.ID), item.ID)
		// Seedance has no size field and no workflow_id, so nothing may be derived.
		require.Nil(t, item.RatioSizes(), item.ID)
		derived := item.WithDerivedCapabilities()
		require.Empty(t, derived.Sizes, item.ID)
		require.False(t, derived.SupportsFirstLastFrame, item.ID)
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
	require.Empty(t, sd.Sizes)
	require.False(t, sd.SupportsFirstLastFrame)
}
