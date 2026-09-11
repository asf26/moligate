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
export type EnterpriseBillingSource = 'local' | 's10' | 'moligate'

export interface EnterpriseBillingAccount {
  id: number
  name: string
  code: string
  source: EnterpriseBillingSource
  usernames: string[]
  pricing_rule: string
  enabled: boolean
  source_configured: boolean
  created_at: number
  updated_at: number
}

export interface EnterpriseBillingAccountPayload {
  name: string
  code: string
  source: EnterpriseBillingSource
  usernames: string[]
  pricing_rule: string
  enabled: boolean
}

export interface EnterpriseBillingRawRow {
  id: number
  time: number
  type: number
  username: string
  token: string
  model: string
  quota: number
  prompt_tokens: number
  completion_tokens: number
  use_time: number
  channel_id: number
  channel_name: string
  group: string
  request_id: string
}

export interface EnterpriseBillingCustomerRow {
  order_id: string
  time: string
  model_type: string
  model_name: string
  use_time: number
  request_id: string
  price: number
}

export interface EnterpriseBillingSummary {
  raw_count: number
  billed_count: number
  prompt_tokens: number
  completion_tokens: number
  total_quota: number
  total_price: number
  currency: string
}

export interface EnterpriseBillingReport {
  account: EnterpriseBillingAccount
  month: string
  period_start: number
  period_end: number
  summary: EnterpriseBillingSummary
  raw_rows: EnterpriseBillingRawRow[]
  customer_rows: EnterpriseBillingCustomerRow[]
  page: number
  page_size: number
  total: number
}

export interface EnterpriseBillingQuery {
  account_id: number
  month: string
  p?: number
  page_size?: number
}

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}
