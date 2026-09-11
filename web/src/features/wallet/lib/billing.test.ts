import { describe, expect, test, vi } from 'vitest'

import { getPaymentMethodName } from './billing'

describe('getPaymentMethodName', () => {
  test('keeps unknown backend payment methods readable without translating them', () => {
    const translate = vi.fn((key: string) => `translated:${key}`)

    expect(getPaymentMethodName('balance', translate)).toBe('balance')
    expect(translate).not.toHaveBeenCalled()
  })

  test('translates known payment methods', () => {
    const translate = vi.fn((key: string) => `translated:${key}`)

    expect(getPaymentMethodName('stripe', translate)).toBe('translated:Stripe')
    expect(translate).toHaveBeenCalledWith('Stripe')
  })
})
