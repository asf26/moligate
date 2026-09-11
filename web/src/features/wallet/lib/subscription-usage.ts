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

For commercial licensing, please contact support@quantumnous.com
*/
import type { QuotaDataItem } from '@/features/dashboard/types'
import type { UserSubscriptionRecord } from '@/features/subscriptions/types'

export const SUBSCRIPTION_USAGE_WINDOW_SECONDS = 30 * 24 * 60 * 60

export interface ModelUsageSummary {
  model: string
  quota: number
  tokenUsed: number
  requests: number
}

export interface SubscriptionUsageWindow {
  startTimestamp: number
  endTimestamp: number
}

function nonNegativeFinite(value: number | undefined): number {
  const numeric = Number(value)
  return Number.isFinite(numeric) && numeric > 0 ? numeric : 0
}

/** Aggregate the dashboard's hourly rows into one row per model. */
export function aggregateUsageByModel(
  rows: QuotaDataItem[]
): ModelUsageSummary[] {
  const summaries = new Map<string, ModelUsageSummary>()

  for (const row of rows) {
    const model = row.model_name?.trim() || '__unknown__'
    const current = summaries.get(model) || {
      model,
      quota: 0,
      tokenUsed: 0,
      requests: 0,
    }
    current.quota += nonNegativeFinite(row.quota)
    current.tokenUsed += nonNegativeFinite(row.token_used)
    current.requests += nonNegativeFinite(row.count)
    summaries.set(model, current)
  }

  return [...summaries.values()].sort((left, right) => {
    if (right.quota !== left.quota) return right.quota - left.quota
    if (right.requests !== left.requests) return right.requests - left.requests
    return left.model.localeCompare(right.model)
  })
}

/**
 * Return a backend-compatible, at-most-30-day window for the active plans.
 * Usage data is exported in hourly buckets, so the end is inclusive.
 */
export function getSubscriptionUsageWindow(
  subscriptions: UserSubscriptionRecord[],
  now = Math.floor(Date.now() / 1000)
): SubscriptionUsageWindow {
  const safeNow = Number.isFinite(now) && now > 0 ? Math.floor(now) : 0
  const active = subscriptions
    .map((record) => record.subscription)
    .filter((subscription) => subscription?.status === 'active')

  const earliestStart = active.reduce((earliest, subscription) => {
    const start = nonNegativeFinite(subscription?.start_time)
    return start > 0 ? Math.min(earliest, start) : earliest
  }, Number.POSITIVE_INFINITY)

  const latestEnd = active.reduce((latest, subscription) => {
    const end = nonNegativeFinite(subscription?.end_time)
    return end > latest ? end : latest
  }, 0)

  const endTimestamp = Math.max(
    1,
    Math.min(safeNow || 1, latestEnd > 0 ? latestEnd : safeNow || 1)
  )
  const subscriptionStart =
    Number.isFinite(earliestStart) && earliestStart > 0
      ? earliestStart
      : endTimestamp - SUBSCRIPTION_USAGE_WINDOW_SECONDS
  const startTimestamp = Math.max(
    1,
    Math.max(
      subscriptionStart,
      endTimestamp - SUBSCRIPTION_USAGE_WINDOW_SECONDS
    )
  )

  return { startTimestamp, endTimestamp }
}
