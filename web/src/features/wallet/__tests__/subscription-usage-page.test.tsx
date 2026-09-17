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
*/
import { render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { getUserQuotaDates } from '@/features/dashboard/api'
import {
  getPublicPlans,
  getSelfSubscriptionFull,
} from '@/features/subscriptions/api'
import type {
  PlanRecord,
  SelfSubscriptionData,
} from '@/features/subscriptions/types'

import { SubscriptionUsagePage } from '../subscription-usage-page'

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { children?: ReactNode; to?: string }) => (
    <a href={props.to}>{props.children}</a>
  ),
}))

vi.mock('@/features/dashboard/api', () => ({
  getUserQuotaDates: vi.fn(),
}))

vi.mock('@/features/subscriptions/api', () => ({
  getPublicPlans: vi.fn(),
  getSelfSubscriptionFull: vi.fn(),
  updateBillingPreference: vi.fn(),
}))

const plan: PlanRecord = {
  plan: {
    id: 10,
    title: 'GPT Pro',
    subtitle: 'GPT configured plan',
    price_amount: 79,
    currency: 'CNY',
    billing_ratio: 0,
    duration_unit: 'month',
    duration_value: 1,
    quota_reset_period: 'monthly',
    quota_reset_custom_seconds: 0,
    enabled: true,
    sort_order: 1,
    allow_balance_pay: true,
    allow_wallet_overflow: true,
    max_purchase_per_user: 0,
    total_amount: 40_000_000,
    daily_amount: 2_000_000,
    weekly_amount: 12_000_000,
    monthly_amount: 40_000_000,
    upgrade_group: '',
    downgrade_group: '',
    applicable_groups: [],
    model_family: 'gpt',
    included_models: ['gpt-5.5'],
    stripe_price_id: '',
    creem_product_id: '',
    waffo_pancake_product_id: '',
  },
}

const subscriptionData: SelfSubscriptionData = {
  billing_preference: 'subscription_first',
  subscriptions: [
    {
      subscription: {
        id: 3,
        user_id: 7,
        plan_id: 10,
        status: 'active',
        source: 'balance',
        start_time: 1_900_000_000,
        end_time: 1_900_259_200,
        amount_total: 40_000_000,
        amount_used: 5_000_000,
        daily_amount: 2_000_000,
        daily_used: 100_000,
        weekly_amount: 12_000_000,
        weekly_used: 1_000_000,
        monthly_amount: 40_000_000,
        monthly_used: 5_000_000,
        resource_grants: [
          {
            resource_key: 'gpt-image-2',
            resource_type: 'image_count',
            model_name: 'gpt-image-2',
            display_name: 'Image pack',
            amount: 10,
            used: 3,
          },
        ],
      },
    },
  ],
  all_subscriptions: [],
}

// An image package carries the placeholder quota (1 unit) that exists only so
// the plan row is not "unlimited"; its real entitlement is the counted
// generations grant.
const imagePlan: PlanRecord = {
  plan: {
    ...plan.plan,
    id: 45,
    title: 'Banana 3,000 pack',
    model_family: 'banana',
    included_models: ['gemini-3-pro-image-preview'],
    total_amount: 1,
    daily_amount: 1,
    weekly_amount: 1,
    monthly_amount: 1,
  },
}

const imageSubscriptionData: SelfSubscriptionData = {
  billing_preference: 'subscription_first',
  subscriptions: [
    {
      subscription: {
        id: 9,
        user_id: 7,
        plan_id: 45,
        status: 'active',
        source: 'order',
        start_time: 1_900_000_000,
        end_time: 1_900_259_200,
        amount_total: 1,
        amount_used: 0,
        daily_amount: 1,
        daily_used: 0,
        weekly_amount: 1,
        weekly_used: 0,
        monthly_amount: 1,
        monthly_used: 0,
        resource_grants: [
          {
            resource_key: 'nano-banana',
            resource_type: 'image_count',
            model_name: 'gemini-3-pro-image-preview',
            display_name: 'Banana pack',
            amount: 3000,
            used: 12,
          },
        ],
      },
    },
  ],
  all_subscriptions: [],
}

describe('SubscriptionUsagePage', () => {
  beforeEach(() => {
    vi.mocked(getPublicPlans).mockResolvedValue({ success: true, data: [plan] })
    vi.mocked(getSelfSubscriptionFull).mockResolvedValue({
      success: true,
      data: subscriptionData,
    })
    vi.mocked(getUserQuotaDates).mockResolvedValue({
      success: true,
      data: [
        {
          model_name: 'gpt-5.5',
          created_at: 1_900_100_000,
          quota: 3_000_000,
          token_used: 1200,
          count: 4,
        },
        {
          model_name: 'gpt-5.5',
          created_at: 1_900_101_000,
          quota: 1_000_000,
          token_used: 500,
          count: 2,
        },
        {
          model_name: 'gpt-image-2',
          created_at: 1_900_102_000,
          quota: 2_000_000,
          token_used: 0,
          count: 1,
        },
      ],
    })
  })

  test('shows active plan details and usage aggregated by model', async () => {
    render(<SubscriptionUsagePage />)

    expect(document.querySelector('.subscription-usage-page')).toHaveClass(
      'subscription-wide-page'
    )
    expect(
      await screen.findByRole('heading', { name: 'Usage by model' })
    ).toBeInTheDocument()
    expect(screen.getByText('GPT Pro')).toBeInTheDocument()
    expect(screen.getByText('gpt-5.5')).toBeInTheDocument()
    expect(screen.getByText('gpt-image-2')).toBeInTheDocument()
    expect(screen.getByText('Image pack')).toBeVisible()
    expect(screen.getByText('3 / 10 generations')).toBeVisible()
    expect(screen.getByRole('table')).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'Subscription Plans' })
    ).toHaveAttribute('href', '/subscription-plans')
  })

  test('requests a backend-compatible usage window', async () => {
    render(<SubscriptionUsagePage />)

    await screen.findByRole('heading', { name: 'Usage by model' })

    const [params] = vi.mocked(getUserQuotaDates).mock.calls[0] || []
    expect(params).toEqual(
      expect.objectContaining({
        default_time: 'day',
        start_timestamp: expect.any(Number),
        end_timestamp: expect.any(Number),
      })
    )
    if (params) {
      expect(params.end_timestamp - params.start_timestamp).toBeLessThanOrEqual(
        30 * 24 * 60 * 60
      )
    }
  })

  test('shows counted generations instead of the placeholder quota for image packages', async () => {
    vi.mocked(getPublicPlans).mockResolvedValue({
      success: true,
      data: [imagePlan],
    })
    vi.mocked(getSelfSubscriptionFull).mockResolvedValue({
      success: true,
      data: imageSubscriptionData,
    })

    render(<SubscriptionUsagePage />)

    expect(await screen.findByText('Banana 3,000 pack')).toBeVisible()

    expect(document.querySelector('.subscription-usage-plan-meters')).toBeNull()
    expect(screen.queryByText('Daily Quota')).not.toBeInTheDocument()
    expect(screen.queryByText('Weekly Quota')).not.toBeInTheDocument()
    expect(screen.queryByText('Monthly Quota')).not.toBeInTheDocument()
    expect(screen.queryByText(/0\.000002/)).not.toBeInTheDocument()

    expect(screen.getByText('12 / 3000 generations')).toBeVisible()
    expect(screen.getByText('12 generations')).toBeVisible()
    expect(screen.getByText('Remaining generations')).toBeVisible()
    expect(screen.getByText('2,988 generations')).toBeVisible()
  })

  test('keeps the quota meters when the package also carries real quota', async () => {
    render(<SubscriptionUsagePage />)

    expect(await screen.findByText('GPT Pro')).toBeVisible()

    expect(
      document.querySelector('.subscription-usage-plan-meters')
    ).not.toBeNull()
    expect(screen.getByText('Daily Quota')).toBeVisible()
    expect(screen.getByText('Remaining quota')).toBeVisible()
  })
})
