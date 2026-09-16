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
import { describe, expect, test } from 'vitest'

import type { SubscriptionPlan } from '../../types'
import {
  formValuesToPlanPayload,
  PLAN_FORM_DEFAULTS,
  planToFormValues,
} from '../plan-form'

const plan: SubscriptionPlan = {
  id: 10,
  title: 'Preview plan',
  subtitle: '',
  price_amount: 42,
  currency: 'CNY',
  billing_ratio: 1,
  duration_unit: 'day',
  duration_value: 30,
  quota_reset_period: 'never',
  enabled: true,
  sale_enabled: true,
  sort_order: 10,
  allow_balance_pay: false,
  allow_wallet_overflow: false,
  max_purchase_per_user: 0,
  total_amount: 3_000_000,
  daily_amount: 300_000,
  weekly_amount: 1_000_000,
  monthly_amount: 3_000_000,
  applicable_groups: ['gpt'],
  included_models: ['gpt-5.5'],
}

describe('subscription plan sale form', () => {
  test('new plans fail closed until sale is explicitly enabled', () => {
    expect(PLAN_FORM_DEFAULTS.enabled).toBe(true)
    expect(PLAN_FORM_DEFAULTS.sale_enabled).toBe(false)
    expect(PLAN_FORM_DEFAULTS.billing_ratio).toBe(1)
  })

  test('round-trips the independent sale switch in admin payloads', () => {
    const values = planToFormValues(plan)
    expect(values.sale_enabled).toBe(true)

    const payload = formValuesToPlanPayload({
      ...values,
      sale_enabled: false,
    })
    expect(payload.plan.enabled).toBe(true)
    expect(payload.plan.sale_enabled).toBe(false)
  })
})
