package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestIsVideoGenerationModelRecognizesPlatformVideoFamilies(t *testing.T) {
	tests := map[string]bool{
		"seedance2.0-stable-full-720p": true,
		"seedance2.5-stable-480p":      true,
		"minimax-h3-original-768p":     true,
		"sora-2-pro":                   true,
		"gpt-image-2":                  false,
		"gpt-5.4":                      false,
	}
	for modelName, expected := range tests {
		t.Run(modelName, func(t *testing.T) {
			assert.Equal(t, expected, IsVideoGenerationModel(modelName))
		})
	}
}

func TestNewAPIStableVideoAdvertisesOpenAIVideoEndpoint(t *testing.T) {
	endpoints := GetEndpointTypesByChannelType(constant.ChannelTypeNewAPI, "seedance2.0-stable-full-720p")
	assert.Equal(t, constant.EndpointTypeOpenAIVideo, endpoints[0])
	assert.Contains(t, endpoints, constant.EndpointTypeOpenAI)
}

func TestSoraVideoEndpointIsNotDuplicated(t *testing.T) {
	endpoints := GetEndpointTypesByChannelType(constant.ChannelTypeSora, "sora-2")
	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAIVideo}, endpoints)
}
