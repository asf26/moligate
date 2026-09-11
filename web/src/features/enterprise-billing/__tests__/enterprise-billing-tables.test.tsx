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
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { CustomerRowsTable, RawRowsTable } from '../index'
import type { EnterpriseBillingReport } from '../types'

const report: EnterpriseBillingReport = {
  account: {
    id: 1,
    name: 'Acme AI',
    code: 'acme-ai',
    source: 'local',
    usernames: ['alice'],
    pricing_rule: 'p * 2.5',
    enabled: true,
    source_configured: true,
    created_at: 0,
    updated_at: 0,
  },
  month: '2026-03',
  period_start: 0,
  period_end: 0,
  summary: {
    raw_count: 1,
    billed_count: 1,
    prompt_tokens: 100,
    completion_tokens: 50,
    total_quota: 150,
    total_price: 0.001,
    currency: 'USD',
  },
  raw_rows: [
    {
      id: 42,
      time: 1772719567,
      type: 2,
      username: 'alice',
      token: 'acme-key',
      model: 'gpt-5.5',
      quota: 150,
      prompt_tokens: 100,
      completion_tokens: 50,
      use_time: 2,
      channel_id: 7,
      channel_name: 'OpenAI',
      group: 'default',
      request_id: 'req-42',
    },
  ],
  customer_rows: [
    {
      order_id: 'req-42',
      time: '2026-03-05 14:06:07',
      model_type: '文生文',
      model_name: 'gpt-5.5',
      use_time: 2,
      request_id: 'req-42',
      price: 0.001,
    },
  ],
  page: 1,
  page_size: 50,
  total: 1,
}

describe('Enterprise billing tables', () => {
  it('keeps the raw usage table horizontally scrollable on narrow layouts', () => {
    const { container } = render(<RawRowsTable report={report} />)

    expect(
      container.querySelector('[data-slot="table-container"]')
    ).toHaveClass('overflow-x-auto')
    expect(container.querySelector('[data-slot="table"]')).toHaveClass(
      'min-w-[1120px]'
    )
    expect(screen.getByText('req-42')).toBeInTheDocument()
  })

  it('keeps customer invoice columns and converted timestamps intact', () => {
    const { container } = render(<CustomerRowsTable report={report} />)

    expect(container.querySelector('[data-slot="table"]')).toHaveClass(
      'min-w-[760px]'
    )
    expect(screen.getByText('Order ID')).toBeInTheDocument()
    expect(screen.getByText('2026-03-05 14:06:07')).toBeInTheDocument()
    expect(screen.getByText('文生文')).toBeInTheDocument()
  })
})
