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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { getPublicPlans } from '@/features/subscriptions/api'
import type { PlanRecord } from '@/features/subscriptions/types'

import { PricingPreview } from '../sections/pricing'

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { children?: ReactNode; to?: string }) => (
    <a href={props.to}>{props.children}</a>
  ),
}))

vi.mock('@/features/subscriptions/api', () => ({
  getPublicPlans: vi.fn(),
}))

const configuredPlan: PlanRecord = {
  plan: {
    id: 101,
    title: 'Backend Starter',
    subtitle: 'Configured in the admin console',
    price_amount: 40,
    currency: 'CNY',
    duration_unit: 'month',
    duration_value: 1,
    quota_reset_period: 'monthly',
    quota_reset_custom_seconds: 0,
    enabled: true,
    sort_order: 10,
    allow_balance_pay: true,
    allow_wallet_overflow: true,
    max_purchase_per_user: 0,
    total_amount: 20000000,
    daily_amount: 15000000,
    weekly_amount: 52500000,
    monthly_amount: 150000000,
    upgrade_group: '',
    downgrade_group: '',
    applicable_groups: [],
    stripe_price_id: '',
    creem_product_id: '',
    waffo_pancake_product_id: '',
    model_family: '',
    included_models: [],
  },
}

describe('PricingPreview', () => {
  function renderPricing() {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    return render(
      <QueryClientProvider client={queryClient}>
        <PricingPreview />
      </QueryClientProvider>
    )
  }

  beforeEach(() => {
    vi.stubGlobal(
      'IntersectionObserver',
      class IntersectionObserverMock {
        observe() {}
        unobserve() {}
        disconnect() {}
      }
    )
  })

  it('renders the enabled plans returned by the backend', async () => {
    vi.mocked(getPublicPlans).mockResolvedValue({
      success: true,
      data: [configuredPlan],
    })

    renderPricing()

    expect(
      await screen.findByRole('heading', { name: 'Backend Starter' })
    ).toBeInTheDocument()
    expect(
      screen.getAllByText('Configured in the admin console')
    ).not.toHaveLength(0)
    expect(screen.getByText('¥40.00')).toBeInTheDocument()
    expect(screen.getByText('Daily limit $30')).toBeInTheDocument()
    expect(
      screen.queryByRole('heading', { name: 'Basic' })
    ).not.toBeInTheDocument()
  })

  it('shows an empty state when the backend has no enabled plans', async () => {
    vi.mocked(getPublicPlans).mockResolvedValue({
      success: true,
      data: [],
    })

    renderPricing()

    expect(await screen.findByText('No plans available')).toBeVisible()
  })

  it('shows a retry state when loading plans fails', async () => {
    vi.mocked(getPublicPlans).mockRejectedValue(new Error('network error'))

    renderPricing()

    expect(await screen.findByRole('alert')).toHaveTextContent('Failed to load')
    expect(screen.getByRole('button', { name: 'Retry' })).toBeVisible()
  })
})
