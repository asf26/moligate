package common

import "strings"

var (
	// OpenAIResponseOnlyModels is a list of models that are only available for OpenAI responses.
	OpenAIResponseOnlyModels = []string{
		"o3-pro",
		"o3-deep-research",
		"o4-mini-deep-research",
	}
	ImageGenerationModels = []string{
		"dall-e-3",
		"dall-e-2",
		"prefix:gpt-image-",
		"prefix:imagen-",
		"flux-",
		"flux.1-",
	}
	VideoGenerationModels = []string{
		"prefix:sora-",
		"prefix:minimax-h3-",
		"prefix:seedance2.",
		"prefix:seedance-",
		"prefix:veo-",
		"prefix:kling-",
		"prefix:vidu-",
		"prefix:hailuo-",
		"prefix:wan-",
	}
	OpenAITextModels = []string{
		"gpt-",
		"o1",
		"o3",
		"o4",
		"chatgpt",
	}
)

func IsOpenAIResponseOnlyModel(modelName string) bool {
	for _, m := range OpenAIResponseOnlyModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

func IsImageGenerationModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range ImageGenerationModels {
		if strings.Contains(modelName, m) {
			return true
		}
		if strings.HasPrefix(m, "prefix:") && strings.HasPrefix(modelName, strings.TrimPrefix(m, "prefix:")) {
			return true
		}
	}
	return false
}

func IsVideoGenerationModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, model := range VideoGenerationModels {
		if strings.HasPrefix(model, "prefix:") {
			if strings.HasPrefix(modelName, strings.TrimPrefix(model, "prefix:")) {
				return true
			}
			continue
		}
		if strings.Contains(modelName, model) {
			return true
		}
	}
	return false
}

func IsOpenAITextModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range OpenAITextModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}
