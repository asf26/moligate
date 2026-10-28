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

	"github.com/stretchr/testify/assert"
)

func TestValidateVideoAccountBaseURLAcceptsOrigins(t *testing.T) {
	// Empty keeps the default host; otherwise any http(s) origin is allowed
	// because video accounts may point at any CTMOAI-compatible deployment.
	assert.NoError(t, validateVideoAccountBaseURL(""))
	assert.NoError(t, validateVideoAccountBaseURL("https://cdn.gavinteam.online"))
	assert.NoError(t, validateVideoAccountBaseURL("https://cdn.gavinteam.online/"))
	assert.NoError(t, validateVideoAccountBaseURL("http://127.0.0.1:8080"))
}

func TestValidateVideoAccountBaseURLRejectsPathsAndBadSchemes(t *testing.T) {
	// The gateway appends /v1/models and /v1/videos itself, so a path or query
	// in this field would silently build a wrong endpoint.
	assert.Error(t, validateVideoAccountBaseURL("https://cdn.gavinteam.online/v1"))
	assert.Error(t, validateVideoAccountBaseURL("https://cdn.gavinteam.online?x=1"))
	assert.Error(t, validateVideoAccountBaseURL("https://cdn.gavinteam.online#frag"))
	assert.Error(t, validateVideoAccountBaseURL("cdn.gavinteam.online"))
	assert.Error(t, validateVideoAccountBaseURL("ftp://cdn.gavinteam.online"))
}
