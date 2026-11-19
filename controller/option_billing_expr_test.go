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
	"github.com/stretchr/testify/require"
)

func TestValidateBillingExprOptionAcceptsRunnableExpressions(t *testing.T) {
	require.NoError(t, validateBillingExprOption(`{"m1":"tier(\"base\", p * 2 + c * 10)"}`))
	require.NoError(t, validateBillingExprOption(`{}`))
}

func TestValidateBillingExprOptionRejectsUnknownVariables(t *testing.T) {
	// cc12h is not part of the expression environment: it compiles to nil and
	// blows up at billing time.
	err := validateBillingExprOption(`{"gpt-6.2-sol":"len <= 272000 ? tier(\"0_272k\", p * 2 + cc * 2.5 + cc12h * 5 + c * 10) : tier(\"272k_plus\", p * 4 + c * 15)"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gpt-6.2-sol")
	assert.Contains(t, err.Error(), "invalid operation")
}

func TestValidateBillingExprOptionRejectsNonObjectPayload(t *testing.T) {
	assert.Error(t, validateBillingExprOption(`[]`))
	assert.Error(t, validateBillingExprOption(`not json`))
}
