/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
)

func TestVideoAccountMediaTokenRoundTrip(t *testing.T) {
	account := &model.VideoAccount{Id: 7, ApiKey: "sk-account-key"}
	token := videoAccountMediaToken(account, "vidmedia_abc123")

	mediaID, ok := verifyVideoAccountMediaToken(account, token)
	assert.True(t, ok)
	assert.Equal(t, "vidmedia_abc123", mediaID)
}

func TestVideoAccountMediaTokenRejectsTampering(t *testing.T) {
	account := &model.VideoAccount{Id: 7, ApiKey: "sk-account-key"}
	token := videoAccountMediaToken(account, "vidmedia_abc123")

	// Another account must not be able to read this media by editing the id.
	other := &model.VideoAccount{Id: 7, ApiKey: "sk-other-key"}
	_, ok := verifyVideoAccountMediaToken(other, token)
	assert.False(t, ok)

	// Swapping the media id invalidates the signature.
	swapped := "7.vidmedia_zzz." + token[len("7.vidmedia_abc123."):]
	_, ok = verifyVideoAccountMediaToken(account, swapped)
	assert.False(t, ok)

	for _, bad := range []string{"", "7", "7.vidmedia_abc123", "x.vidmedia_abc123.00"} {
		_, ok := verifyVideoAccountMediaToken(account, bad)
		assert.False(t, ok, "token %q must be rejected", bad)
	}
}
