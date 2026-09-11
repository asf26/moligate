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
import { WalletStatsCard } from '../wallet-stats-card'

const user: UserWalletData = {
  id: 7,
  username: 'wallet-user',
  quota: 500000000,
  used_quota: 125000000,
  request_count: 42,
  aff_quota: 0,
  aff_history_quota: 0,
  aff_count: 0,
  group: 'default',
}

describe('WalletStatsCard', () => {
  test('keeps the balance, usage, and request summaries visible together', () => {
    const { container } = render(<WalletStatsCard user={user} topupCount={3} />)

    expect(screen.getByText('Current Balance')).toBeInTheDocument()
    expect(screen.getByText('Top-ups')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('Total Usage')).toBeInTheDocument()
    expect(screen.getByText('API Requests')).toBeInTheDocument()
    expect(screen.getByText('42')).toBeInTheDocument()
    expect(container.querySelector('.wallet-summary-grid')).toBeInTheDocument()
    expect(container.querySelectorAll('.wallet-summary-stat')).toHaveLength(3)
  })

  test('keeps summary cards mobile-first with a fluid inner grid', () => {
    const { container } = render(<WalletStatsCard user={user} />)

    expect(container.querySelector('.wallet-summary-grid')).toHaveClass('grid')
    expect(container.querySelector('.wallet-overview-inner')).toHaveClass(
      'grid',
      'min-h-52'
    )
    expect(container.querySelector('.wallet-summary-stats')).toHaveClass(
      'grid-cols-1',
      'sm:grid-cols-3'
    )
    expect(container.querySelector('.wallet-balance-explainer')).toHaveClass(
      'flex'
    )
  })
})
