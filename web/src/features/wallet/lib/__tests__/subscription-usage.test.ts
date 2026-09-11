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
import { describe, expect, test } from 'vitest'

import type { QuotaDataItem } from '@/features/dashboard/types'

import {
  aggregateUsageByModel,
  getSubscriptionUsageWindow,
  SUBSCRIPTION_USAGE_WINDOW_SECONDS,
} from '../subscription-usage'

describe('subscription usage aggregation', () => {
  test('groups hourly rows by model and sorts by quota', () => {
    const rows: QuotaDataItem[] = [
      {
        model_name: 'gpt-5.5',
        created_at: 1,
        quota: 4,
        token_used: 30,
        count: 2,
      },
      {
        model_name: 'claude-sonnet-5',
        created_at: 2,
        quota: 8,
        token_used: 20,
        count: 1,
      },
      {
        model_name: 'gpt-5.5',
        created_at: 3,
        quota: 6,
        token_used: 40,
        count: 3,
      },
      { model_name: '', created_at: 4, quota: -2, token_used: 5, count: 1 },
    ]

    expect(aggregateUsageByModel(rows)).toEqual([
      { model: 'gpt-5.5', quota: 10, tokenUsed: 70, requests: 5 },
      { model: 'claude-sonnet-5', quota: 8, tokenUsed: 20, requests: 1 },
      { model: '__unknown__', quota: 0, tokenUsed: 5, requests: 1 },
    ])
  })

  test('keeps the usage query within the backend 30-day limit', () => {
    const now = 2_000_000_000
    const window = getSubscriptionUsageWindow(
      [
        {
          subscription: {
            id: 1,
            user_id: 1,
            plan_id: 1,
            status: 'active',
            start_time: now - 2 * SUBSCRIPTION_USAGE_WINDOW_SECONDS,
            end_time: now + 3600,
            amount_total: 1,
            amount_used: 0,
          },
        },
      ],
      now
    )

    expect(window.endTimestamp).toBe(now)
    expect(window.startTimestamp).toBe(now - SUBSCRIPTION_USAGE_WINDOW_SECONDS)
    expect(window.endTimestamp - window.startTimestamp).toBe(
      SUBSCRIPTION_USAGE_WINDOW_SECONDS
    )
  })
})
