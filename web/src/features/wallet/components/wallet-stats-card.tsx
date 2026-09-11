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
  Activity,
  BarChart3,
  CheckCircle2,
  CreditCard,
  Eye,
  ShieldCheck,
  WalletCards,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { IconBadge, type IconBadgeTone } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import { useSystemConfig } from '@/hooks/use-system-config'
import { formatSystemCurrencyUSD } from '@/lib/currency'
import { DEFAULT_CURRENCY_CONFIG } from '@/stores/system-config-store'

import type { UserWalletData } from '../types'

interface WalletStatsCardProps {
  user: UserWalletData | null
  loading?: boolean
  topupCount?: number
}

export function WalletStatsCard(props: WalletStatsCardProps) {
  const { t } = useTranslation()
  const { currency } = useSystemConfig()
  const quotaPerUnit =
    currency?.quotaPerUnit > 0
      ? currency.quotaPerUnit
      : DEFAULT_CURRENCY_CONFIG.quotaPerUnit
  if (props.loading) {
    return <WalletStatsSkeleton />
  }

  const stats: {
    label: string
    value: string
    description: string
    icon: typeof WalletCards
    tone: IconBadgeTone
  }[] = [
    {
      label: t('Top-ups'),
      value: (props.topupCount ?? 0).toLocaleString(),
      description: t('Billing History'),
      icon: CreditCard,
      tone: 'primary',
    },
    {
      label: t('Total Usage'),
      value: formatSystemCurrencyUSD(
        (props.user?.used_quota ?? 0) / quotaPerUnit,
        {
          digitsLarge: 2,
          digitsSmall: 4,
          abbreviate: true,
        }
      ),
      description: t('Total consumed quota'),
      icon: BarChart3,
      tone: 'info',
    },
    {
      label: t('API Requests'),
      value: (props.user?.request_count ?? 0).toLocaleString(),
      description: t('Total requests made'),
      icon: Activity,
      tone: 'chart-4',
    },
  ]

  return (
    <div className='wallet-summary-grid grid gap-5 @6xl/content:grid-cols-[minmax(0,1.4fr)_minmax(360px,1fr)]'>
      <section
        aria-label={t('Wallet overview')}
        className='wallet-overview-card bg-card overflow-hidden rounded-xl border shadow-sm'
      >
        <div className='wallet-overview-inner grid min-h-52 md:grid-cols-[minmax(300px,0.9fr)_minmax(0,1.6fr)]'>
          <div className='wallet-balance-panel flex flex-col justify-between p-5 sm:p-7'>
            <div>
              <div className='wallet-eyebrow text-muted-foreground flex items-center gap-2 text-xs font-semibold tracking-wider uppercase'>
                <IconBadge
                  tone='primary'
                  size='sm'
                  className='wallet-summary-icon'
                >
                  <WalletCards />
                </IconBadge>
                {t('Current Balance')}
                <Eye className='size-3.5' aria-hidden='true' />
              </div>
              <div className='wallet-balance-value text-foreground mt-5 font-mono text-4xl font-bold tracking-normal break-all tabular-nums sm:text-5xl'>
                {formatSystemCurrencyUSD(
                  (props.user?.quota ?? 0) / quotaPerUnit,
                  {
                    digitsLarge: 2,
                    digitsSmall: 4,
                    abbreviate: true,
                  }
                )}
              </div>
              <p className='text-muted-foreground mt-2 text-sm'>
                {t('Remaining quota')}
              </p>
            </div>
            <div className='wallet-balance-note text-muted-foreground mt-6 flex items-center gap-2 text-xs'>
              <CheckCircle2
                className='text-success size-4'
                aria-hidden='true'
              />
              {t('Available for all supported models')}
            </div>
          </div>

          <div className='wallet-summary-stats-wrap min-w-0 p-3 sm:p-4'>
            <div className='wallet-summary-stats bg-primary/[0.035] grid min-h-full grid-cols-1 overflow-hidden rounded-lg border sm:grid-cols-3'>
              {stats.map((item) => (
                <div
                  key={item.label}
                  className='wallet-summary-stat flex min-h-36 flex-col justify-between border-b p-4 last:border-b-0 sm:border-b-0 sm:p-4 sm:[&:nth-child(n+2)]:border-l'
                >
                  <div className='flex items-center gap-2.5'>
                    <IconBadge tone={item.tone} size='md'>
                      <item.icon />
                    </IconBadge>
                    <span className='text-muted-foreground text-xs font-semibold tracking-wider whitespace-nowrap uppercase'>
                      {item.label}
                    </span>
                  </div>
                  <div>
                    <div className='text-foreground mt-5 font-mono text-2xl font-bold tracking-normal break-all tabular-nums'>
                      {item.value}
                    </div>
                    <p className='text-muted-foreground mt-1 text-xs'>
                      {item.description}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className='wallet-balance-explainer bg-card flex min-h-52 items-center gap-5 rounded-xl border p-5 shadow-sm sm:p-7'>
        <div className='wallet-illustration bg-info/10 relative flex size-24 shrink-0 items-center justify-center rounded-xl sm:size-28'>
          <div className='wallet-illustration-stack' aria-hidden='true'>
            <span className='wallet-illustration-card wallet-illustration-card-back' />
            <span className='wallet-illustration-card wallet-illustration-card-front'>
              <WalletCards className='text-info size-9 sm:size-10' />
            </span>
          </div>
          <span className='bg-card absolute -right-2 -bottom-2 flex size-9 items-center justify-center rounded-lg border shadow-sm'>
            <ShieldCheck className='text-success size-5' aria-hidden='true' />
          </span>
        </div>
        <div className='min-w-0'>
          <h3 className='text-base leading-6 font-semibold'>
            {t('One balance for every API call')}
          </h3>
          <p className='text-muted-foreground mt-2 text-sm leading-6'>
            {t('Use your balance across every supported AI model.')}
          </p>
          <p className='text-muted-foreground mt-1 text-sm leading-6'>
            {t('Clear usage records keep every charge easy to verify.')}
          </p>
        </div>
      </section>
    </div>
  )
}

function WalletStatsSkeleton() {
  return (
    <div className='wallet-summary-grid grid gap-5 @6xl/content:grid-cols-[minmax(0,1.4fr)_minmax(360px,1fr)]'>
      <div className='grid min-h-52 rounded-xl border md:grid-cols-[minmax(300px,0.9fr)_minmax(0,1.6fr)]'>
        <div className='p-5 sm:p-6'>
          <Skeleton className='h-7 w-36' />
          <Skeleton className='mt-5 h-10 w-48' />
          <Skeleton className='mt-2 h-4 w-24' />
        </div>
        <div className='min-w-0 p-3 sm:p-4'>
          <div className='grid min-h-full grid-cols-1 overflow-hidden rounded-lg border sm:grid-cols-3'>
            {['topups', 'usage', 'requests'].map((key) => (
              <div
                key={key}
                className='border-b p-4 last:border-b-0 sm:border-b-0 sm:p-5 sm:[&:nth-child(n+2)]:border-l'
              >
                <Skeleton className='h-8 w-32' />
                <Skeleton className='mt-8 h-8 w-28' />
                <Skeleton className='mt-2 h-3.5 w-24' />
              </div>
            ))}
          </div>
        </div>
      </div>
      <div className='flex min-h-48 items-center gap-5 rounded-xl border p-5 sm:p-6'>
        <Skeleton className='size-24 rounded-xl sm:size-28' />
        <div className='flex-1 space-y-3'>
          <Skeleton className='h-5 w-48 max-w-full' />
          <Skeleton className='h-4 w-full' />
          <Skeleton className='h-4 w-4/5' />
        </div>
      </div>
    </div>
  )
}
