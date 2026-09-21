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
import { useQuery } from '@tanstack/react-query'
import type { PaginationState } from '@tanstack/react-table'
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { CompactDateTimeRangePicker } from '@/components/compact-date-time-range-picker'
import { DataTablePage, useDataTable } from '@/components/data-table'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { getAdminOrders } from '../api'
import { getOrderStatusOptions } from '../constants'
import type { OrderRecord, OrdersQuery } from '../types'
import { useOrdersColumns } from './orders-columns'

interface OrderFilters {
  tradeNo: string
  username: string
  userId: string
  status: string
  start?: Date
  end?: Date
}

const EMPTY_FILTERS: OrderFilters = {
  tradeNo: '',
  username: '',
  userId: '',
  status: '',
}

const ALL_STATUSES = 'all'

function toUnixSeconds(date?: Date): number | undefined {
  return date ? Math.floor(date.getTime() / 1000) : undefined
}

function hasAnyFilter(filters: OrderFilters): boolean {
  return Boolean(
    filters.tradeNo.trim() ||
    filters.username.trim() ||
    filters.userId.trim() ||
    filters.status ||
    filters.start ||
    filters.end
  )
}

export function OrdersTable() {
  const { t } = useTranslation()
  const columns = useOrdersColumns()
  const [draft, setDraft] = useState<OrderFilters>(EMPTY_FILTERS)
  const [applied, setApplied] = useState<OrderFilters>(EMPTY_FILTERS)
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })

  const statusOptions = useMemo(
    () => [
      { value: ALL_STATUSES, label: t('All') },
      ...getOrderStatusOptions(t),
    ],
    [t]
  )

  const queryParams = useMemo<OrdersQuery>(() => {
    const userId = Number.parseInt(applied.userId.trim(), 10)
    return {
      p: pagination.pageIndex + 1,
      page_size: pagination.pageSize,
      trade_no: applied.tradeNo.trim() || undefined,
      username: applied.username.trim() || undefined,
      user_id: Number.isFinite(userId) && userId > 0 ? userId : undefined,
      status: applied.status || undefined,
      start_time: toUnixSeconds(applied.start),
      end_time: toUnixSeconds(applied.end),
    }
  }, [applied, pagination.pageIndex, pagination.pageSize])

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['admin-orders', queryParams],
    queryFn: async () => {
      const result = await getAdminOrders(queryParams)
      if (!result.success) {
        toast.error(result.message || t('Request failed'))
        return { items: [] as OrderRecord[], total: 0 }
      }
      return {
        items: result.data?.items || [],
        total: result.data?.total || 0,
      }
    },
    placeholderData: (previous) => previous,
  })

  const applyFilters = useCallback(() => {
    setApplied(draft)
    setPagination((previous) => ({ ...previous, pageIndex: 0 }))
  }, [draft])

  const resetFilters = useCallback(() => {
    setDraft(EMPTY_FILTERS)
    setApplied(EMPTY_FILTERS)
    setPagination((previous) => ({ ...previous, pageIndex: 0 }))
  }, [])

  const submitOnEnter = useCallback(
    (event: React.KeyboardEvent<HTMLInputElement>) => {
      if (event.key === 'Enter') {
        applyFilters()
      }
    },
    [applyFilters]
  )

  const { table } = useDataTable({
    data: data?.items || [],
    columns,
    manualPagination: true,
    pagination,
    onPaginationChange: setPagination,
    totalCount: data?.total || 0,
  })

  return (
    <DataTablePage
      table={table}
      columns={columns}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyTitle={t('No orders yet')}
      emptyDescription={t(
        'No orders match the current filters. Try widening the time range.'
      )}
      skeletonKeyPrefix='orders-skeleton'
      applyHeaderSize
      toolbarProps={{
        customSearch: (
          <div className='flex flex-wrap items-center gap-2'>
            <Input
              value={draft.tradeNo}
              onChange={(event) =>
                setDraft((previous) => ({
                  ...previous,
                  tradeNo: event.target.value,
                }))
              }
              onKeyDown={submitOnEnter}
              placeholder={t('Trade No')}
              aria-label={t('Trade No')}
              className='h-8 w-full sm:w-44'
            />
            <Input
              value={draft.username}
              onChange={(event) =>
                setDraft((previous) => ({
                  ...previous,
                  username: event.target.value,
                }))
              }
              onKeyDown={submitOnEnter}
              placeholder={t('Username')}
              aria-label={t('Username')}
              className='h-8 w-full sm:w-36'
            />
            <Input
              value={draft.userId}
              onChange={(event) =>
                setDraft((previous) => ({
                  ...previous,
                  userId: event.target.value,
                }))
              }
              onKeyDown={submitOnEnter}
              placeholder={t('User ID')}
              aria-label={t('User ID')}
              inputMode='numeric'
              className='h-8 w-full sm:w-28'
            />
            <CompactDateTimeRangePicker
              start={draft.start}
              end={draft.end}
              onChange={(range) =>
                setDraft((previous) => ({
                  ...previous,
                  start: range.start,
                  end: range.end,
                }))
              }
              className='h-8 w-full sm:w-64'
            />
            <Select
              items={statusOptions}
              value={draft.status || ALL_STATUSES}
              onValueChange={(value) =>
                setDraft((previous) => ({
                  ...previous,
                  status: !value || value === ALL_STATUSES ? '' : value,
                }))
              }
            >
              <SelectTrigger
                className='h-8 w-full sm:w-32'
                aria-label={t('Status')}
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  {statusOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
        ),
        hasAdditionalFilters: hasAnyFilter(draft),
        onSearch: applyFilters,
        searchLoading: isFetching,
        onReset: resetFilters,
        hideViewOptions: true,
      }}
    />
  )
}
