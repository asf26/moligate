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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import {
  getSelfAffiliateInvitees,
  getSelfAffiliateRewardPointSettlements,
  getSelfAffiliateSummary,
} from '@/features/affiliate-commissions/api'
import type { AffiliateCommissionSummary } from '@/features/affiliate-commissions/types'

import { AffiliateCommissionsCard } from '../affiliate-commissions-card'

vi.mock('@/features/affiliate-commissions/api', () => ({
  getSelfAffiliateInvitees: vi.fn(),
  getSelfAffiliateRewardPointSettlements: vi.fn(),
  getSelfAffiliateSummary: vi.fn(),
  quoteSelfAffiliateRewardPoints: vi.fn(),
  redeemSelfAffiliateRewardPoints: vi.fn(),
}))

vi.mock(
  '@/features/affiliate-commissions/components/affiliate-invitees-table',
  () => ({
    AffiliateInviteesTable: () => null,
  })
)

vi.mock('@/lib/api', () => ({
  getSelf: vi.fn(),
}))

const summary: AffiliateCommissionSummary = {
  pending_amount_micros: 0,
  settled_amount_micros: 0,
  total_amount_micros: 0,
  pending_points: 0,
  wallet_redeemed_points: 0,
  offline_settled_points: 0,
  settled_points: 0,
  total_points: 0,
  pending_count: 0,
  settled_count: 0,
  wallet_redeemed_count: 0,
  offline_settled_count: 0,
  redeemed_count: 0,
  total_count: 0,
  currency: 'USD',
  level1_rate_bps: 500,
  level2_rate_bps: 250,
  points_per_amount_unit: 1,
  invite_count: 0,
}

function renderCard() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <AffiliateCommissionsCard />
    </QueryClientProvider>
  )
}

describe('AffiliateCommissionsCard reward rate display', () => {
  beforeEach(() => {
    vi.mocked(getSelfAffiliateSummary).mockResolvedValue({
      success: true,
      data: summary,
    })
    vi.mocked(getSelfAffiliateRewardPointSettlements).mockResolvedValue({
      success: true,
      data: { page: 1, page_size: 8, total: 0, items: [] },
    })
    vi.mocked(getSelfAffiliateInvitees).mockResolvedValue({
      success: true,
      data: { page: 1, page_size: 50, total: 0, items: [] },
    })
  })

  test('shows the level one rate without a level two rate card', async () => {
    renderCard()

    await waitFor(() => {
      expect(screen.getByText('5%')).toBeInTheDocument()
    })

    expect(screen.getByText('Level 1 reward rate')).toBeInTheDocument()
    expect(screen.getByText('Points per paid unit')).toBeInTheDocument()
    expect(screen.queryByText('Level 2 reward rate')).not.toBeInTheDocument()
  })
})
