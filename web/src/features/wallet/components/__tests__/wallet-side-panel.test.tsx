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
import userEvent from '@testing-library/user-event'
import { describe, expect, test, vi } from 'vitest'

import type { TopupRecord } from '../../types'
import { WalletSidePanel } from '../wallet-side-panel'

vi.mock('../wallet-contact-support-button', () => ({
  WalletContactSupportButton: () => (
    <button type='button'>Contact support</button>
  ),
}))

const billingHistoryResult = {
  records: [],
  loading: false,
  total: 0,
  page: 1,
  pageSize: 5,
  onPageChange: () => undefined,
}

describe('WalletSidePanel', () => {
  test('shows the latest personal orders and opens the full history', async () => {
    const user = userEvent.setup()
    const onOpenBilling = vi.fn()
    const onPageChange = vi.fn()
    const records: TopupRecord[] = [
      {
        id: 1,
        user_id: 7,
        amount: 10,
        money: 72.6,
        trade_no: 'ORDER-20260902',
        payment_method: 'wxpay',
        create_time: 1788314400,
        status: 'success',
      },
    ]
    const billing = {
      ...billingHistoryResult,
      total: 11,
      onPageChange,
      records,
    }

    const { container } = render(
      <WalletSidePanel onOpenBilling={onOpenBilling} billing={billing} />
    )

    expect(screen.getByText('ORDER-20260902')).toBeInTheDocument()
    expect(screen.getByText('Success')).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Order History' })
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /View all/ }))
    expect(onOpenBilling).toHaveBeenCalledOnce()
    await user.click(screen.getByRole('button', { name: 'Go to next page' }))
    expect(onPageChange).toHaveBeenCalledWith(2)
    expect(
      container.querySelector('.wallet-orders-pagination')
    ).toHaveTextContent('1 / 3')
  })

  test('keeps a clear empty state when the user has no orders', () => {
    render(
      <WalletSidePanel
        onOpenBilling={() => undefined}
        billing={billingHistoryResult}
      />
    )

    expect(screen.getByText('No orders yet')).toBeInTheDocument()
    expect(
      screen.getByText('Your transaction history will appear here')
    ).toBeInTheDocument()
  })
})
