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
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import {
  getPublicPlans,
  getSelfSubscriptionFull,
} from '@/features/subscriptions/api'
import type {
  PlanRecord,
  SelfSubscriptionData,
} from '@/features/subscriptions/types'

import { SubscriptionPlansCard } from '../subscription-plans-card'

vi.mock('@/features/subscriptions/api', () => ({
  getPublicPlans: vi.fn(),
  getSelfSubscriptionFull: vi.fn(),
  updateBillingPreference: vi.fn(),
}))

vi.mock(
  '@/features/subscriptions/components/dialogs/subscription-purchase-dialog',
  () => ({
    SubscriptionPurchaseDialog: () => null,
  })
)

function makePlan(
  id: number,
  title: string,
  durationUnit: 'month' | 'year',
  extras: Partial<PlanRecord['plan']> = {}
): PlanRecord {
  return {
    plan: {
      id,
      title,
      subtitle: `${title} configured plan`,
      price_amount: durationUnit === 'year' ? 790 : 79,
      currency: 'CNY',
      duration_unit: durationUnit,
      duration_value: 1,
      quota_reset_period: 'monthly',
      quota_reset_custom_seconds: 0,
      enabled: true,
      sort_order: id,
      allow_balance_pay: true,
      allow_wallet_overflow: true,
      max_purchase_per_user: 0,
      total_amount: 40000000,
      daily_amount: 2000000,
      weekly_amount: 12000000,
      monthly_amount: 40000000,
      upgrade_group: '',
      downgrade_group: '',
      applicable_groups: [],
      stripe_price_id: '',
      creem_product_id: '',
      waffo_pancake_product_id: '',
      ...extras,
    },
  }
}

const activeData: SelfSubscriptionData = {
  billing_preference: 'subscription_first',
  subscriptions: [
    {
      subscription: {
        id: 1,
        user_id: 7,
        plan_id: 2,
        status: 'active',
        source: 'balance',
        start_time: 1,
        end_time: Math.floor(Date.now() / 1000) + 86400 * 30,
        amount_total: 40000000,
        amount_used: 1000000,
        daily_amount: 2000000,
        daily_used: 100000,
        daily_reset_time: 0,
        weekly_amount: 12000000,
        weekly_used: 500000,
        weekly_reset_time: 0,
        monthly_amount: 40000000,
        monthly_used: 1000000,
        monthly_reset_time: 0,
        next_reset_time: 0,
      },
    },
  ],
  all_subscriptions: [],
}

function renderPlans() {
  return render(
    <SubscriptionPlansCard
      topupInfo={null}
      onAvailabilityChange={() => undefined}
    />
  )
}

describe('subscription plans layout', () => {
  beforeEach(() => {
    vi.mocked(getPublicPlans).mockResolvedValue({
      success: true,
      data: [
        makePlan(1, 'Starter', 'month'),
        makePlan(2, 'Professional', 'month'),
      ],
    })
    vi.mocked(getSelfSubscriptionFull).mockResolvedValue({
      success: true,
      data: activeData,
    })
  })

  test('renders every configured plan without a billing-cycle switch', async () => {
    renderPlans()

    expect(
      await screen.findByRole('tablist', { name: 'Model packages' })
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('region', { name: 'My Current Plan' })
    ).not.toBeInTheDocument()
    expect(
      screen.getAllByRole('heading', { name: 'Professional' })
    ).toHaveLength(1)
    expect(screen.getAllByRole('article')).toHaveLength(2)
    expect(
      screen.queryByRole('button', { name: 'Monthly' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Yearly' })
    ).not.toBeInTheDocument()
    expect(screen.getByText('Hot recommendation')).toBeVisible()
  })

  test('renders reference-style entitlement details without decorative plan icons', async () => {
    vi.mocked(getPublicPlans).mockResolvedValue({
      success: true,
      data: [makePlan(1, 'Starter', 'month', { max_purchase_per_user: 3 })],
    })

    renderPlans()

    const card = await screen.findByRole('article', { name: 'Starter' })
    expect(within(card).getByText('Only 3 left')).toBeVisible()
    expect(
      within(card).getByText('Supports the All supported models model family')
    ).toBeVisible()
    expect(
      within(card).getAllByText(/Includes \d+ models from All supported models/)
    ).toHaveLength(2)
    expect(within(card).getByText('Package benefits')).toBeVisible()
    expect(within(card).getByText(/Valid for 1 months?/)).toBeVisible()
  })

  test('renders configured annual and monthly plans together', async () => {
    vi.mocked(getPublicPlans).mockResolvedValue({
      success: true,
      data: [
        makePlan(1, 'Starter', 'month'),
        makePlan(2, 'Professional', 'month'),
        makePlan(3, 'Professional Annual', 'year'),
      ],
    })

    renderPlans()

    await screen.findByRole('tabpanel', { name: /All supported models/ })

    expect(screen.getAllByRole('article')).toHaveLength(3)
    expect(screen.getByRole('heading', { name: 'Starter' })).toBeVisible()
    expect(screen.getByRole('heading', { name: 'Professional' })).toBeVisible()
    expect(
      screen.getByRole('heading', { name: 'Professional Annual' })
    ).toBeVisible()
    expect(
      screen.queryByRole('button', { name: 'Yearly' })
    ).not.toBeInTheDocument()
  })

  test('switches model-family tabs without rendering individual model names', async () => {
    const user = userEvent.setup()
    vi.mocked(getPublicPlans).mockResolvedValue({
      success: true,
      data: [
        makePlan(1, 'CC Max Basic', 'month', {
          model_family: 'ccmax',
          included_models: ['claude-opus-5'],
        }),
        makePlan(2, 'GPT Pro', 'month', {
          model_family: 'gpt',
          included_models: ['gpt-5.5', 'gpt-image-2'],
        }),
      ],
    })

    renderPlans()

    const tablist = await screen.findByRole('tablist', {
      name: 'Model packages',
    })
    const ccMaxTab = within(tablist).getByRole('tab', { name: /CC Max/ })
    const gptTab = within(tablist).getByRole('tab', { name: /GPT/ })

    expect(ccMaxTab).toHaveAttribute('aria-selected', 'true')
    expect(gptTab).toHaveAttribute('aria-selected', 'false')
    const ccMaxFamily = screen.getByRole('tabpanel', { name: /CC Max/ })
    expect(ccMaxFamily).toBeVisible()
    expect(
      screen.queryByRole('tabpanel', { name: /GPT/ })
    ).not.toBeInTheDocument()
    expect(screen.queryByText('claude-opus-5')).not.toBeInTheDocument()
    expect(screen.getAllByText('Estimated wallet credit')).toHaveLength(1)
    expect(screen.getAllByText('Bonus ratio (%)')).toHaveLength(1)
    expect(within(ccMaxFamily).getByText('+1%')).toBeVisible()
    expect(within(ccMaxFamily).getAllByRole('progressbar')).toHaveLength(1)

    await user.click(gptTab)

    expect(ccMaxTab).toHaveAttribute('aria-selected', 'false')
    expect(gptTab).toHaveAttribute('aria-selected', 'true')
    expect(screen.queryByText('claude-opus-5')).not.toBeInTheDocument()
    const gptFamily = screen.getByRole('tabpanel', { name: /GPT/ })
    expect(gptFamily).toBeVisible()
    expect(within(gptFamily).getByText('Estimated wallet credit')).toBeVisible()
    expect(within(gptFamily).getByText('Bonus ratio (%)')).toBeVisible()
    expect(within(gptFamily).queryByText('gpt-image-2')).not.toBeInTheDocument()
    expect(within(gptFamily).getAllByRole('article')).toHaveLength(1)
  })

  test('uses configured card badge, recommendation, and benefits', async () => {
    vi.mocked(getPublicPlans).mockResolvedValue({
      success: true,
      data: [
        makePlan(1, 'Starter', 'month', {
          title: 'Starter',
          model_family: 'ccmax',
          badge_text: '限量推荐',
          is_recommended: true,
          benefits: ['适合个人开发', '包含专属额度'],
        }),
        makePlan(2, 'Professional', 'month', {
          title: 'Professional',
          model_family: 'ccmax',
        }),
      ],
    })

    renderPlans()

    const card = await screen.findByRole('article', { name: 'Starter' })
    expect(within(card).getByText('限量推荐')).toBeVisible()
    expect(within(card).getByText('适合个人开发')).toBeVisible()
    expect(within(card).getByText('包含专属额度')).toBeVisible()
    expect(within(card).queryByText(/Supports the/)).not.toBeInTheDocument()
  })

  test('renders configured bonus resources without exposing a model catalog', async () => {
    vi.mocked(getPublicPlans).mockResolvedValue({
      success: true,
      data: [
        makePlan(1, 'GPT Pro', 'month', {
          model_family: 'gpt',
          bonus_resources: [
            {
              resource_key: 'gpt-image-2',
              resource_type: 'image_count',
              model_name: 'gpt-image-2',
              display_name: 'Image pack',
              amount: 10,
            },
            {
              resource_key: 'grok',
              resource_type: 'quota',
              model_name: 'grok-4',
              display_name: 'Grok bonus',
              amount: 2000000,
            },
          ],
        }),
      ],
    })

    renderPlans()

    const card = await screen.findByRole('article', { name: 'GPT Pro' })
    expect(within(card).getByText('Included extras')).toBeVisible()
    expect(within(card).getByText('Image pack')).toBeVisible()
    expect(within(card).getByText('10 generations')).toBeVisible()
    expect(within(card).getByText('Grok bonus')).toBeVisible()
    expect(within(card).queryByText('gpt-image-2')).not.toBeInTheDocument()
  })
})
