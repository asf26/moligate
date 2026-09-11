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
import { render, screen } from '@testing-library/react'
import { describe, expect, test } from 'vitest'

import type { UserWalletData } from '../../types'
import { AffiliateRewardsCard } from '../affiliate-rewards-card'

const user: UserWalletData = {
  id: 7,
  username: 'wallet-user',
  quota: 500000000,
  used_quota: 0,
  request_count: 0,
  aff_quota: 0,
  aff_history_quota: 0,
  aff_count: 0,
  group: 'default',
}

describe('AffiliateRewardsCard', () => {
  test('exposes the invitation link for sharing on the rewards page', () => {
    const link = 'https://example.test/sign-up?aff=wallet-user'
    render(
      <AffiliateRewardsCard
        user={user}
        affiliateLink={link}
        onTransfer={() => undefined}
      />
    )

    expect(screen.getByRole('textbox', { name: 'Referral link:' })).toHaveValue(
      link
    )
    expect(
      screen.getByRole('button', { name: 'Copy referral link' })
    ).toBeInTheDocument()
  })

  test('keeps the registration card focused on the share link', () => {
    render(
      <AffiliateRewardsCard
        user={user}
        affiliateLink='https://example.test/sign-up?aff=wallet-user'
        onTransfer={() => undefined}
      />
    )

    expect(screen.queryByText('Pending')).not.toBeInTheDocument()
    expect(screen.queryByText('Total Earned')).not.toBeInTheDocument()
    expect(screen.queryByText('Invites')).not.toBeInTheDocument()
  })
})
