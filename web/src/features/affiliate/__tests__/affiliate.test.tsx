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
*/
import { render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, test, vi } from 'vitest'

import { Affiliate } from '../index'

vi.mock('@/components/layout', () => {
  const Slot = ({ children }: { children?: ReactNode }) => <>{children}</>
  const Layout = ({ children }: { children?: ReactNode }) => (
    <div>{children}</div>
  )
  Layout.Title = Slot
  Layout.Description = Slot
  Layout.Content = Slot
  Layout.Actions = Slot
  Layout.Breadcrumb = Slot
  return { SectionPageLayout: Layout }
})

vi.mock('@/features/wallet/components/affiliate-rewards-card', () => ({
  AffiliateRewardsCard: () => <div data-testid='affiliate-rewards-card' />,
}))

vi.mock('@/features/wallet/components/affiliate-commissions-card', () => ({
  AffiliateCommissionsCard: () => (
    <div data-testid='affiliate-commissions-card' />
  ),
}))

vi.mock('@/features/wallet/components/dialogs/transfer-dialog', () => ({
  TransferDialog: () => null,
}))

vi.mock('@/features/wallet/hooks', () => ({
  useAffiliate: () => ({
    affiliateLink: 'https://example.test/sign-up?aff=test',
    loading: false,
    transferQuota: vi.fn(),
    transferring: false,
  }),
  useTopupInfo: () => ({ topupInfo: null }),
}))

vi.mock('@/lib/api', () => ({
  getSelf: vi.fn().mockResolvedValue({ success: true, data: null }),
}))

describe('Affiliate page layout', () => {
  test('uses the full available content width for the reward sections', () => {
    render(<Affiliate />)

    const content = document.querySelector('.affiliate-page-content')
    expect(content).toBeInTheDocument()
    expect(content).toHaveClass('w-full', 'min-w-0')
    expect(screen.getByTestId('affiliate-rewards-card')).toBeInTheDocument()
    expect(screen.getByTestId('affiliate-commissions-card')).toBeInTheDocument()
  })
})
