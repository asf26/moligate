/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your
option) any later version.

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
  getIncludedModels,
  getPlanModelFamily,
  groupPlansByModelFamily,
} from '../model-family'

function makePlan(
  id: number,
  title: string,
  extras: Partial<SubscriptionPlan> = {}
): SubscriptionPlan {
  return {
    id,
    title,
    subtitle: '',
    price_amount: 10,
    currency: 'CNY',
    billing_ratio: 0,
    duration_unit: 'month',
    duration_value: 1,
    quota_reset_period: 'never',
    enabled: true,
    sort_order: id,
    allow_balance_pay: true,
    allow_wallet_overflow: true,
    max_purchase_per_user: 0,
    total_amount: 0,
    daily_amount: 0,
    weekly_amount: 0,
    monthly_amount: 0,
    upgrade_group: '',
    downgrade_group: '',
    applicable_groups: [],
    stripe_price_id: '',
    creem_product_id: '',
    waffo_pancake_product_id: '',
    ...extras,
  }
}

describe('subscription model families', () => {
  test('uses an explicit model family and model allow-list when configured', () => {
    const plan = makePlan(1, 'Developer', {
      model_family: 'gpt',
      included_models: ['gpt-5.5', 'gpt-image-2'],
    })

    const family = getPlanModelFamily(plan)
    expect(family?.key).toBe('gpt')
    expect(family).toBeDefined()
    if (!family) throw new Error('expected GPT family')
    expect(getIncludedModels(plan, family)).toEqual(['gpt-5.5', 'gpt-image-2'])
  })

  test('infers a family from legacy plan metadata without changing the plan', () => {
    const plan = makePlan(2, 'Chinese models Pro', {
      applicable_groups: ['cn-premium'],
    })

    expect(getPlanModelFamily(plan)?.key).toBe('chinese')
    expect(plan.model_family).toBeUndefined()
  })

  test('recognizes each configured family without a catch-all group', () => {
    const groups = groupPlansByModelFamily([
      makePlan(1, 'Banana pack', { model_family: 'banana' }),
      makePlan(2, 'GPT Image pack', { model_family: 'gpt-image' }),
      makePlan(3, 'Kiro Claude Basic', { model_family: 'kiro-claude' }),
      makePlan(4, 'GPT Pro', { model_family: 'gpt' }),
      makePlan(5, 'CC Max Basic', { model_family: 'ccmax' }),
    ])

    expect(groups.map((group) => group.family.key)).toEqual([
      'ccmax',
      'kiro-claude',
      'gpt',
      'gpt-image',
      'banana',
    ])
    expect(
      groups.find((group) => group.family.key === 'kiro-claude')?.plans
    ).toHaveLength(1)
    expect(
      groups.find((group) => group.family.key === 'gpt-image')?.plans
    ).toHaveLength(1)
    expect(
      groups.find((group) => group.family.key === 'banana')?.plans
    ).toHaveLength(1)
  })

  test('does not classify legacy all or unknown plans as Kiro or another family', () => {
    const unknown = makePlan(1, 'General')
    const legacyAll = makePlan(2, 'All access', { model_family: 'all' })

    expect(getPlanModelFamily(unknown)).toBeUndefined()
    expect(getPlanModelFamily(legacyAll)).toBeUndefined()
    expect(groupPlansByModelFamily([unknown, legacyAll])).toEqual([])
  })

  test('infers image families from legacy metadata', () => {
    expect(
      getPlanModelFamily(
        makePlan(1, 'GPT Image 3,000 pack', {
          included_models: ['gpt-image-2'],
        })
      )?.key
    ).toBe('gpt-image')
    expect(
      getPlanModelFamily(
        makePlan(2, 'Nano Banana 3,000 pack', {
          included_models: ['gemini-3-pro-image-preview'],
        })
      )?.key
    ).toBe('banana')
  })
})
