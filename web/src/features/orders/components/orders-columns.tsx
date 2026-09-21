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
import { type ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'

import { DataTableColumnHeader } from '@/components/data-table/core/column-header'
import { StatusBadge } from '@/components/status-badge'
import { formatSubscriptionPrice } from '@/features/subscriptions/lib'
import { formatQuota, formatTimestamp } from '@/lib/format'

import {
  ORDER_KIND,
  formatPaymentMethod,
  getOrderKindLabel,
  getOrderKindVariant,
  getOrderStatusLabel,
  getOrderStatusVariant,
} from '../constants'
import type { OrderRecord } from '../types'

export function useOrdersColumns(): ColumnDef<OrderRecord>[] {
  const { t } = useTranslation()

  return [
    {
      accessorKey: 'trade_no',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Trade No')} />
      ),
      meta: { mobileTitle: true },
      cell: ({ row }) => (
        <span
          className='font-[family-name:var(--font-geist-mono)] text-xs break-all'
          title={row.original.trade_no}
        >
          {row.original.trade_no}
        </span>
      ),
      size: 230,
    },
    {
      accessorKey: 'username',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('User')} />
      ),
      cell: ({ row }) => (
        <div className='flex min-w-0 flex-col'>
          <span className='truncate font-medium'>
            {row.original.username || '-'}
          </span>
          <span className='text-muted-foreground text-xs tabular-nums'>
            ID {row.original.user_id}
          </span>
        </div>
      ),
      size: 150,
    },
    {
      accessorKey: 'kind',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Type')} />
      ),
      cell: ({ row }) => (
        <StatusBadge
          label={getOrderKindLabel(row.original.kind, t)}
          variant={getOrderKindVariant(row.original.kind)}
          copyable={false}
          showDot
        />
      ),
      size: 130,
    },
    {
      accessorKey: 'plan_title',
      header: t('Content'),
      cell: ({ row }) => {
        if (row.original.kind === ORDER_KIND.TOP_UP) {
          // The kind badge already says "Balance top-up"; show the credited quota.
          return (
            <span className='text-muted-foreground text-xs tabular-nums'>
              +{formatQuota(row.original.amount)}
            </span>
          )
        }
        return (
          <span className='truncate' title={row.original.plan_title}>
            {row.original.plan_title || '-'}
          </span>
        )
      },
      size: 180,
    },
    {
      accessorKey: 'money',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Amount')} />
      ),
      cell: ({ row }) => (
        <span className='font-[family-name:var(--font-geist-mono)] tabular-nums'>
          {formatSubscriptionPrice(row.original.money)}
        </span>
      ),
      size: 110,
    },
    {
      accessorKey: 'payment_method',
      header: t('Payment Method'),
      meta: { mobileHidden: true },
      cell: ({ row }) => (
        <span className='text-muted-foreground text-xs'>
          {formatPaymentMethod(
            row.original.payment_method,
            row.original.payment_provider,
            t
          )}
        </span>
      ),
      size: 150,
    },
    {
      accessorKey: 'status',
      header: t('Status'),
      cell: ({ row }) => (
        <StatusBadge
          label={getOrderStatusLabel(row.original.status, t)}
          variant={getOrderStatusVariant(row.original.status)}
          copyable={false}
          showDot
        />
      ),
      size: 120,
    },
    {
      accessorKey: 'create_time',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Created At')} />
      ),
      cell: ({ row }) => (
        <span className='text-muted-foreground text-xs tabular-nums'>
          {formatTimestamp(row.original.create_time)}
        </span>
      ),
      size: 160,
    },
    {
      accessorKey: 'complete_time',
      header: t('Completed At'),
      meta: { mobileHidden: true },
      cell: ({ row }) => (
        <span className='text-muted-foreground text-xs tabular-nums'>
          {row.original.complete_time
            ? formatTimestamp(row.original.complete_time)
            : '-'}
        </span>
      ),
      size: 160,
    },
  ]
}
