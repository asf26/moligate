/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package minimax_h3

const ChannelName = "minimax-h3"

const VideoEndpoint = "/v1/videos"

// ModelList mirrors the public MiniMax H3 model catalog. Pricing is configured
// separately in the gateway ratio settings, while the duration multiplier is
// applied by the task adaptor.
var ModelList = []string{
	"minimax-h3-original-768p",
	"minimax-h3-original-1080p",
	"minimax-h3-original-cf-2k",
	"minimax-h3-original-cf-4k",
	"minimax-h3-comic-768p",
	"minimax-h3-comic-cf-2k",
	"minimax-h3-comic-cf-4k",
	"minimax-h3-quantized-768p",
}

var supportedAspectRatios = map[string]bool{
	"16:9": true,
	"9:16": true,
	"1:1":  true,
	"2:3":  true,
	"3:2":  true,
	"3:4":  true,
	"4:3":  true,
	"21:9": true,
}

func IsModel(model string) bool {
	for _, candidate := range ModelList {
		if candidate == model {
			return true
		}
	}
	return false
}

func IsModelPrefix(model string) bool {
	for _, candidate := range ModelList {
		if len(model) >= len(candidate) && model[:len(candidate)] == candidate {
			return true
		}
	}
	return false
}

func IsAspectRatio(value string) bool {
	return supportedAspectRatios[value]
}
