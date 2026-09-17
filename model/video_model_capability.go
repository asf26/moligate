package model

import "strings"

// MiniMax H3 fixes the output canvas per aspect ratio and resolution, and its
// API expects the matching `size` value to be submitted alongside
// `aspect_ratio`. CTMOAI's /v1/models endpoint only publishes `resolution` and
// `ratios`, so the ratio -> size mapping has to live here. Values come from the
// "比例、清晰度与 size 尺寸对应关系" table in the upstream H3 API doc (v1.2).
var videoSizesByRatio = map[string]map[string]string{
	"16:9": {"480": "864x480", "768": "1376x768", "1080": "1920x1088"},
	"9:16": {"480": "480x864", "768": "768x1376", "1080": "1088x1920"},
	"1:1":  {"480": "640x640", "768": "1024x1024", "1080": "1440x1440"},
	"2:3":  {"480": "544x800", "768": "832x1248", "1080": "1184x1760"},
	"3:2":  {"480": "800x544", "768": "1248x832", "1080": "1760x1184"},
	"3:4":  {"480": "576x736", "768": "896x1184", "1080": "1248x1664"},
	"4:3":  {"480": "736x576", "768": "1184x896", "1080": "1664x1248"},
	"21:9": {"480": "992x416", "768": "1568x672", "1080": "2208x960"},
}

// videoSizeColumns maps the resolution labels CTMOAI publishes onto the column
// keys of videoSizesByRatio.
var videoSizeColumns = map[string]string{
	"480":   "480",
	"480p":  "480",
	"768":   "768",
	"768p":  "768",
	"1080":  "1080",
	"1080p": "1080",
}

// The CF (super resolution) models do not take a pixel size; the doc requires
// the uppercase resolution label instead, together with aspect_ratio.
var videoSizeLabels = map[string]string{
	"2k": "2K",
	"4k": "4K",
}

// IsMiniMaxH3VideoModel reports whether a catalog entry belongs to the MiniMax
// H3 integration. H3 and the Seedance accounts share one gateway and one
// endpoint but differ in their request dialect: H3 takes a `size` plus
// `reference_videos` / `reference_audios`, Seedance takes neither.
func IsMiniMaxH3VideoModel(group, modelID string) bool {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(modelID)), "minimax-h3-") {
		return true
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(group)), "h3")
}

// MiniMaxH3RatioSizes returns the size to submit for each supported aspect
// ratio, keyed by ratio. An empty result means the caller must not send a size.
func MiniMaxH3RatioSizes(resolution string, ratios []string) map[string]string {
	if len(ratios) == 0 {
		return nil
	}
	normalized := strings.ToLower(strings.TrimSpace(resolution))
	if label, ok := videoSizeLabels[normalized]; ok {
		result := make(map[string]string, len(ratios))
		for _, ratio := range ratios {
			ratio = strings.TrimSpace(ratio)
			if ratio != "" {
				result[ratio] = label
			}
		}
		return result
	}
	column, ok := videoSizeColumns[normalized]
	if !ok {
		return nil
	}
	result := make(map[string]string, len(ratios))
	for _, ratio := range ratios {
		ratio = strings.TrimSpace(ratio)
		if ratio == "" {
			continue
		}
		if size, ok := videoSizesByRatio[ratio][column]; ok {
			result[ratio] = size
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// RatioSizes maps every aspect ratio this model supports onto the size value to
// submit with it. It is empty for models that take no size.
func (item VideoModel) RatioSizes() map[string]string {
	if !IsMiniMaxH3VideoModel(item.Group, item.ID) {
		return nil
	}
	return MiniMaxH3RatioSizes(item.Resolution, item.Ratios)
}

// RequiresReferenceImage reports whether the model refuses a text-only request.
// The CF (super resolution) variants have no text-to-video workflow, so a
// request without reference images fails upstream.
func (item VideoModel) RequiresReferenceImage() bool {
	return strings.Contains(strings.ToLower(item.Group+" "+item.ID), "cf-")
}

// WithDerivedCapabilities fills in the capabilities CTMOAI does not publish but
// the documented integrations require. Sizes are only derived when the catalog
// left them empty, so an upstream response that starts reporting them keeps
// working.
//
// supports_first_last_frame is different: the field is a plain bool, so an
// omitted value is indistinguishable from an explicit false, and CTMOAI omits
// it for every H3 model. H3's API doc documents first/last frame via
// workflow_id=fl2v, so it is enabled for the H3 family. If upstream ever
// disables the mode for a model, the relay still forwards the request and
// upstream rejects it with a clear error.
func (item VideoModel) WithDerivedCapabilities() VideoModel {
	if !IsMiniMaxH3VideoModel(item.Group, item.ID) {
		return item
	}
	if len(item.Sizes) == 0 {
		if sizes := item.RatioSizes(); len(sizes) > 0 {
			// Keep the order of Ratios so Sizes and Ratios stay aligned, and
			// dedupe because every CF ratio maps onto the same label.
			ordered := make([]string, 0, len(sizes))
			for _, ratio := range item.Ratios {
				size := sizes[strings.TrimSpace(ratio)]
				if size == "" || containsVideoString(ordered, size) {
					continue
				}
				ordered = append(ordered, size)
			}
			item.Sizes = ordered
		}
	}
	// H3 supports first/last frame through workflow_id=fl2v. The catalog never
	// reports it, and leaving it false would make the relay reject a documented
	// request.
	item.SupportsFirstLastFrame = true
	return item
}

func containsVideoString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}
