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
import {
  ArrowRight,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  Headphones,
  History,
  LockKeyhole,
  ShieldCheck,
  Zap,
} from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { IconBadge } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import { formatSystemCurrencyUSD } from '@/lib/currency'

import { formatTimestamp, getStatusConfig } from '../lib/billing'
import type { TopupRecord } from '../types'
import { WalletContactSupportButton } from './wallet-contact-support-button'

interface WalletSidePanelProps {
  onOpenBilling: () => void
  billing: {
    records: TopupRecord[]
    loading: boolean
    total: number
    page: number
    pageSize: number
    onPageChange: (page: number) => void
  }
}

export function WalletSidePanel(props: WalletSidePanelProps) {
  const { t } = useTranslation()
  const { records, loading, total, page, pageSize, onPageChange } =
    props.billing
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const assurances = [
    {
      title: t('Encrypted checkout'),
      description: t(
        'Payment details are handled by secure provider channels.'
      ),
      icon: LockKeyhole,
      tone: 'info' as const,
    },
    {
      title: t('Instant credit'),
      description: t('Successful payments are credited automatically.'),
      icon: Zap,
      tone: 'warning' as const,
    },
    {
      title: t('Privacy protection'),
      description: t('Your account and transaction data stay protected.'),
      icon: ShieldCheck,
      tone: 'success' as const,
    },
  ]

  let recentOrdersContent: ReactNode
  if (loading) {
    recentOrdersContent = (
      <div className='space-y-3 p-5'>
        {['order-1', 'order-2', 'order-3'].map((key) => (
          <div key={key} className='flex items-center gap-3'>
            <Skeleton className='h-8 flex-1' />
            <Skeleton className='h-8 w-16' />
          </div>
        ))}
      </div>
    )
  } else if (records.length === 0) {
    recentOrdersContent = (
      <div className='flex min-h-36 flex-col items-center justify-center px-5 py-8 text-center'>
        <CheckCircle2
          className='text-muted-foreground/50 size-8'
          aria-hidden='true'
        />
        <p className='mt-3 text-sm font-medium'>{t('No orders yet')}</p>
        <p className='text-muted-foreground mt-1 text-xs'>
          {t('Your transaction history will appear here')}
        </p>
      </div>
    )
  } else {
    recentOrdersContent = (
      <div className='overflow-x-auto'>
        <table className='w-full table-fixed text-left text-xs'>
          <thead className='bg-muted/35 text-muted-foreground'>
            <tr>
              <th className='w-[48%] px-5 py-2.5 font-medium'>{t('Order')}</th>
              <th className='w-[27%] px-2 py-2.5 font-medium'>{t('Amount')}</th>
              <th className='w-[25%] px-3 py-2.5 text-right font-medium'>
                {t('Status')}
              </th>
            </tr>
          </thead>
          <tbody className='divide-y'>
            {records.map((record) => {
              const status = getStatusConfig(record.status)
              return (
                <tr key={record.id}>
                  <td className='px-5 py-3 align-middle'>
                    <div className='truncate font-mono font-medium'>
                      {record.trade_no}
                    </div>
                    <div className='text-muted-foreground mt-0.5 truncate text-[11px]'>
                      {formatTimestamp(record.create_time)}
                    </div>
                  </td>
                  <td className='px-2 py-3 font-medium tabular-nums'>
                    {formatSystemCurrencyUSD(record.amount, {
                      digitsLarge: 2,
                      digitsSmall: 2,
                      abbreviate: false,
                    })}
                  </td>
                  <td className='px-3 py-3 text-right'>
                    <StatusBadge
                      label={t(status.label)}
                      variant={status.variant}
                      copyable={false}
                    />
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    )
  }

  return (
    <aside
      className='wallet-support-rail'
      aria-label={t('Wallet support information')}
    >
      <section className='wallet-security-card bg-card rounded-xl border p-5 shadow-sm'>
        <div className='mb-4 flex items-center gap-2.5'>
          <IconBadge tone='primary' size='md'>
            <ShieldCheck />
          </IconBadge>
          <h2 className='text-base font-semibold'>{t('Security')}</h2>
        </div>
        <div className='space-y-4'>
          {assurances.map((item) => (
            <div key={item.title} className='flex items-start gap-3'>
              <IconBadge tone={item.tone} size='sm' className='mt-0.5'>
                <item.icon />
              </IconBadge>
              <div className='min-w-0'>
                <h3 className='text-sm font-medium'>{item.title}</h3>
                <p className='text-muted-foreground mt-0.5 text-xs leading-5'>
                  {item.description}
                </p>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className='wallet-help-card bg-card rounded-xl border p-5 shadow-sm'>
        <div className='flex items-start gap-3'>
          <IconBadge tone='info' size='md'>
            <Headphones />
          </IconBadge>
          <div className='min-w-0'>
            <h2 className='text-base font-semibold'>
              {t('Need help with a payment?')}
            </h2>
            <p className='text-muted-foreground mt-1 text-xs leading-5'>
              {t('Our support team can help with payment questions.')}
            </p>
          </div>
        </div>
        <div className='mt-4'>
          <WalletContactSupportButton className='wallet-help-contact-button' />
        </div>
      </section>

      <section className='wallet-orders-card bg-card overflow-hidden rounded-xl border shadow-sm'>
        <div className='flex items-center justify-between gap-3 border-b px-5 py-4'>
          <div className='flex items-center gap-2.5'>
            <IconBadge tone='neutral' size='md'>
              <History />
            </IconBadge>
            <h2 className='text-base font-semibold'>{t('Recent Orders')}</h2>
          </div>
          <Button
            variant='ghost'
            size='sm'
            className='text-primary -mr-2'
            onClick={props.onOpenBilling}
          >
            {t('View all')}
            <ArrowRight data-icon='inline-end' />
          </Button>
        </div>

        {recentOrdersContent}
        {!loading && total > 0 && (
          <div className='wallet-orders-pagination flex items-center justify-between gap-3 border-t px-5 py-3'>
            <span className='text-muted-foreground min-w-0 truncate text-xs'>
              {t('Showing')} {(page - 1) * pageSize + 1}-
              {Math.min(page * pageSize, total)} {t('of')} {total}
            </span>
            <div className='flex shrink-0 items-center gap-1.5'>
              <Button
                variant='outline'
                size='icon'
                className='size-8'
                onClick={() => onPageChange(page - 1)}
                disabled={page <= 1}
                aria-label={t('Go to previous page')}
              >
                <ChevronLeft className='size-4' aria-hidden='true' />
              </Button>
              <span className='text-muted-foreground min-w-12 text-center text-xs tabular-nums'>
                {page} / {totalPages}
              </span>
              <Button
                variant='outline'
                size='icon'
                className='size-8'
                onClick={() => onPageChange(page + 1)}
                disabled={page >= totalPages}
                aria-label={t('Go to next page')}
              >
                <ChevronRight className='size-4' aria-hidden='true' />
              </Button>
            </div>
          </div>
        )}
      </section>
    </aside>
  )
}
