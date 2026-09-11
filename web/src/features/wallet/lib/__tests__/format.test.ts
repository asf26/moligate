import { describe, expect, test } from 'vitest'

import {
  calculatePresetPricing,
  formatLocalPaymentAmount,
  formatUsdCreditAmount,
} from '../format'

describe('wallet currency presentation', () => {
  test('keeps quota in USD while presenting the matching payment in CNY', () => {
    expect(formatUsdCreditAmount(10)).toBe('$10')
    expect(formatLocalPaymentAmount(10)).toBe('¥10')
  })

  test('uses a one-to-one payment rate for an undiscounted preset', () => {
    const pricing = calculatePresetPricing(10, 1, 1, 1)

    expect(pricing.actualPrice).toBe(10)
    expect(pricing.displayValue).toBe(10)
  })
})
