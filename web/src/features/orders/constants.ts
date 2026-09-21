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
import type { TFunction } from 'i18next'

export const ORDER_STATUS = {
  SUCCESS: 'success',
  PENDING: 'pending',
  EXPIRED: 'expired',
  FAILED: 'failed',
} as const

// Two sources feed the ledger: package purchases and wallet recharges.
export const ORDER_KIND = {
  SUBSCRIPTION: 'subscription',
  TOP_UP: 'topup',
} as const

type BadgeVariant = 'success' | 'warning' | 'neutral' | 'danger' | 'info'

export function getOrderKindLabel(kind: string, t: TFunction): string {
  switch (kind) {
    case ORDER_KIND.SUBSCRIPTION:
      return t('Subscription order')
    case ORDER_KIND.TOP_UP:
      return t('Balance top-up')
    default:
      return kind || '-'
  }
}

export function getOrderKindVariant(kind: string): BadgeVariant {
  return kind === ORDER_KIND.TOP_UP ? 'success' : 'info'
}

export function getOrderKindOptions(
  t: TFunction
): { label: string; value: string }[] {
  return [ORDER_KIND.SUBSCRIPTION, ORDER_KIND.TOP_UP].map((kind) => ({
    label: getOrderKindLabel(kind, t),
    value: kind,
  }))
}

export function getOrderStatusLabel(status: string, t: TFunction): string {
  switch (status) {
    case ORDER_STATUS.SUCCESS:
      return t('Paid')
    case ORDER_STATUS.PENDING:
      return t('Pending payment')
    case ORDER_STATUS.EXPIRED:
      return t('Expired')
    case ORDER_STATUS.FAILED:
      return t('Failed')
    default:
      return status || '-'
  }
}

export function getOrderStatusVariant(status: string): BadgeVariant {
  switch (status) {
    case ORDER_STATUS.SUCCESS:
      return 'success'
    case ORDER_STATUS.PENDING:
      return 'warning'
    case ORDER_STATUS.FAILED:
      return 'danger'
    default:
      return 'neutral'
  }
}

export function getOrderStatusOptions(
  t: TFunction
): { label: string; value: string }[] {
  return [
    ORDER_STATUS.PENDING,
    ORDER_STATUS.SUCCESS,
    ORDER_STATUS.EXPIRED,
    ORDER_STATUS.FAILED,
  ].map((status) => ({
    label: getOrderStatusLabel(status, t),
    value: status,
  }))
}

// Payment methods arrive as gateway codes; the console shows the buyer-facing
// name with the gateway the order was routed through.
const PAYMENT_METHOD_KEYS: Record<string, string> = {
  alipay: 'Alipay',
  wxpay: 'WeChat Pay',
  stripe: 'Stripe',
  creem: 'Creem',
  waffo_pancake: 'Waffo Pancake',
}

export function formatPaymentMethod(
  method: string,
  provider: string,
  t: TFunction
): string {
  const key = PAYMENT_METHOD_KEYS[method]
  const label = key ? t(key) : method || '-'
  return provider ? `${label} · ${provider}` : label
}
