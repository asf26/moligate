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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { getAdminOrders } from '../api'
import { OrdersTable } from '../components/orders-table'
import type { OrderRecord } from '../types'

vi.mock('../api', () => ({
  getAdminOrders: vi.fn(),
}))

const orders: OrderRecord[] = [
  {
    id: 68,
    kind: 'subscription',
    user_id: 86,
    username: 'yaoyingjie',
    plan_id: 30,
    plan_title: '国产模型 入门版',
    trade_no: 'SUBUSR86NOYECmmO1789971944',
    money: 105,
    amount: 0,
    payment_method: 'wxpay',
    payment_provider: 'epay',
    status: 'success',
    create_time: 1789971944,
    complete_time: 1789972000,
    name: '',
  },
  {
    id: 90,
    kind: 'topup',
    user_id: 86,
    username: 'yaoyingjie',
    plan_id: 0,
    plan_title: '',
    trade_no: 'TOPUP86AAAA0000000001',
    money: 50,
    amount: 150,
    payment_method: 'stripe',
    payment_provider: 'stripe',
    status: 'success',
    create_time: 1789970000,
    complete_time: 1789970100,
    name: '',
  },
  {
    id: 91,
    kind: 'redemption',
    user_id: 86,
    username: 'yaoyingjie',
    plan_id: 0,
    plan_title: '',
    trade_no: '',
    money: 0,
    amount: 10000000,
    payment_method: '',
    payment_provider: '',
    status: 'success',
    create_time: 1789960000,
    complete_time: 1789960000,
    name: '九月活动码',
  },
  {
    id: 67,
    kind: 'subscription',
    user_id: 1,
    username: 'guochangrong',
    plan_id: 45,
    plan_title: '香蕉生图 3,000 张包',
    trade_no: 'SUBUSR1NOGYtH671789651647',
    money: 450,
    amount: 0,
    payment_method: 'wxpay',
    payment_provider: 'epay',
    status: 'pending',
    create_time: 1789651647,
    complete_time: 0,
    name: '',
  },
]

function renderOrdersTable() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <OrdersTable />
    </QueryClientProvider>
  )
}

describe('OrdersTable', () => {
  beforeEach(() => {
    vi.mocked(getAdminOrders).mockResolvedValue({
      success: true,
      data: { page: 1, page_size: 20, total: 4, items: orders },
    })
  })

  test('lists package purchases and wallet recharges in one ledger', async () => {
    renderOrdersTable()

    expect(await screen.findByText('SUBUSR86NOYECmmO1789971944')).toBeVisible()
    expect(screen.getAllByText('yaoyingjie').length).toBeGreaterThan(0)
    expect(screen.getByText('国产模型 入门版')).toBeVisible()
    expect(screen.getByText('¥105.00')).toBeVisible()
    expect(screen.getAllByText('Subscription order')[0]).toBeVisible()
    expect(screen.getAllByText('Balance top-up')[0]).toBeVisible()
    expect(screen.getAllByText('Redemption Code')[0]).toBeVisible()
    expect(screen.getByText('九月活动码')).toBeVisible()
    expect(screen.getByText('+$20')).toBeVisible()
    expect(screen.getByText('Redeemed')).toBeVisible()
    expect(screen.getByText('Pending payment')).toBeVisible()
    // The recharge row shows the credited face value (¥150 for a ¥50 payment).
    expect(screen.getByText('+¥150')).toBeVisible()
    expect(vi.mocked(getAdminOrders)).toHaveBeenCalledWith(
      expect.objectContaining({ p: 1, page_size: 20 })
    )
  })

  test('sends every filter to the API when searching', async () => {
    const user = userEvent.setup()
    renderOrdersTable()
    await screen.findByText('SUBUSR86NOYECmmO1789971944')

    await user.type(screen.getByLabelText('Trade No'), 'SUBUSR86')
    await user.type(screen.getByLabelText('Username'), 'yaoying')
    await user.type(screen.getByLabelText('User ID'), '86')
    await user.click(screen.getByRole('button', { name: 'Search' }))

    await waitFor(() => {
      expect(vi.mocked(getAdminOrders)).toHaveBeenLastCalledWith(
        expect.objectContaining({
          p: 1,
          trade_no: 'SUBUSR86',
          username: 'yaoying',
          user_id: 86,
        })
      )
    })
  })

  test('keeps the filters untouched until the search is submitted', async () => {
    const user = userEvent.setup()
    renderOrdersTable()
    await screen.findByText('SUBUSR86NOYECmmO1789971944')

    const callsBefore = vi.mocked(getAdminOrders).mock.calls.length
    await user.type(screen.getByLabelText('Username'), 'yaoying')

    expect(vi.mocked(getAdminOrders).mock.calls.length).toBe(callsBefore)
  })

  test('reports an empty result instead of stale rows', async () => {
    vi.mocked(getAdminOrders).mockResolvedValue({
      success: true,
      data: { page: 1, page_size: 20, total: 0, items: [] },
    })

    renderOrdersTable()

    expect(await screen.findByText('No orders yet')).toBeVisible()
    expect(
      screen.getByText(
        'No orders match the current filters. Try widening the time range.'
      )
    ).toBeVisible()
  })
})
