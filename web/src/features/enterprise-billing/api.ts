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
import type { AxiosRequestConfig } from 'axios'

import { api } from '@/lib/api'

import type {
  ApiResponse,
  EnterpriseBillingAccount,
  EnterpriseBillingAccountPayload,
  EnterpriseBillingQuery,
  EnterpriseBillingReport,
} from './types'

function queryString(query: object) {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === '') continue
    params.set(key, String(value))
  }
  return params.toString()
}

function filenameFromHeaders(value: unknown, fallback: string) {
  if (typeof value !== 'string') return fallback
  const encoded = value.match(/filename\*=UTF-8''([^;]+)/i)?.[1]
  if (encoded) {
    try {
      return decodeURIComponent(encoded.replaceAll(/["']/g, ''))
    } catch {
      return encoded.replaceAll(/["']/g, '')
    }
  }
  return value.match(/filename="?([^";]+)"?/i)?.[1]?.trim() || fallback
}

export async function getEnterpriseBillingAccounts(): Promise<
  ApiResponse<EnterpriseBillingAccount[]>
> {
  const res = await api.get('/api/enterprise-billing/accounts')
  return res.data
}

export async function createEnterpriseBillingAccount(
  payload: EnterpriseBillingAccountPayload
): Promise<ApiResponse<EnterpriseBillingAccount>> {
  const res = await api.post('/api/enterprise-billing/accounts', payload)
  return res.data
}

export async function updateEnterpriseBillingAccount(
  id: number,
  payload: EnterpriseBillingAccountPayload
): Promise<ApiResponse<EnterpriseBillingAccount>> {
  const res = await api.put(`/api/enterprise-billing/accounts/${id}`, payload)
  return res.data
}

export async function deleteEnterpriseBillingAccount(
  id: number
): Promise<ApiResponse> {
  const res = await api.delete(`/api/enterprise-billing/accounts/${id}`)
  return res.data
}

export async function getEnterpriseBillingReport(
  query: EnterpriseBillingQuery
): Promise<ApiResponse<EnterpriseBillingReport>> {
  const qs = queryString(query)
  const res = await api.get(`/api/enterprise-billing/report?${qs}`)
  return res.data
}

export async function exportEnterpriseBillingCsv(
  query: EnterpriseBillingQuery,
  kind: 'raw' | 'customer'
): Promise<{ blob: Blob; filename: string }> {
  const qs = queryString(query)
  const res = await api.get(`/api/enterprise-billing/export/${kind}?${qs}`, {
    responseType: 'blob',
    disableDuplicate: true,
  } as AxiosRequestConfig & { disableDuplicate: boolean })
  return {
    blob: res.data as Blob,
    filename: filenameFromHeaders(
      res.headers['content-disposition'],
      `enterprise-billing-${kind}.csv`
    ),
  }
}
