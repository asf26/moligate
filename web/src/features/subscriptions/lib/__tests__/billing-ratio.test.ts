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
import { describe, expect, test } from 'vitest'

import { formatPlanBillingRatio } from '../format'

describe('formatPlanBillingRatio', () => {
  test('renders the configured multiplier as-is', () => {
    expect(formatPlanBillingRatio(1)).toBe('1')
    expect(formatPlanBillingRatio(1.5)).toBe('1.5')
    expect(formatPlanBillingRatio('2')).toBe('2')
    expect(formatPlanBillingRatio({ billing_ratio: 3 })).toBe('3')
  })

  test('falls back to 1× for unconfigured or out-of-range values', () => {
    // Mirrors the backend's EffectiveBillingRatio: 0 means "not configured".
    expect(formatPlanBillingRatio({ billing_ratio: 0 })).toBe('1')
    expect(formatPlanBillingRatio(undefined)).toBe('1')
    expect(formatPlanBillingRatio(null)).toBe('1')
    expect(formatPlanBillingRatio(Number.NaN)).toBe('1')
    expect(formatPlanBillingRatio(-2)).toBe('1')
    expect(formatPlanBillingRatio(120)).toBe('1')
  })
})
