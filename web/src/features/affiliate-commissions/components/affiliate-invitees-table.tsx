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
import { UsersRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import type { AffiliateInvitee } from '@/features/affiliate-commissions/types'
import { formatRewardPoints } from '@/features/wallet/lib'
import { formatTimestamp } from '@/lib/format'
import { cn } from '@/lib/utils'

type Props = {
  items: AffiliateInvitee[]
  total?: number
  isLoading?: boolean
  className?: string
}

function points(value: number | undefined) {
  return formatRewardPoints(value || 0)
}

function InviteeSkeleton() {
  return (
    <div className='grid gap-3 p-4 md:grid-cols-[minmax(200px,1.5fr)_110px_80px_120px_110px_110px_130px] md:items-center'>
      <Skeleton className='h-10 w-44' />
      <Skeleton className='h-4 w-20' />
      <Skeleton className='h-4 w-24' />
      <Skeleton className='h-4 w-20' />
      <Skeleton className='h-4 w-24' />
      <Skeleton className='h-4 w-24' />
      <Skeleton className='h-4 w-24' />
    </div>
  )
}

const skeletonKeys = ['one', 'two', 'three']

export function AffiliateInviteesTable({
  items,
  total = items.length,
  isLoading = false,
  className,
}: Props) {
  const { t } = useTranslation()

  return (
    <div className={cn('overflow-hidden rounded-xl border', className)}>
      <div className='bg-muted/30 hidden grid-cols-[minmax(200px,1.5fr)_110px_80px_120px_110px_110px_130px] gap-3 px-4 py-3 text-xs font-medium md:grid'>
        <span>{t('Invited user')}</span>
        <span>{t('Joined')}</span>
        <span>{t('Top-ups')}</span>
        <span>{t('Reward points contributed')}</span>
        <span>{t('Pending reward points')}</span>
        <span>{t('Settled reward points')}</span>
        <span>{t('Last contribution')}</span>
      </div>

      <div className='divide-y'>
        {isLoading ? (
          skeletonKeys.map((key) => <InviteeSkeleton key={key} />)
        ) : items.length === 0 ? (
          <div className='text-muted-foreground flex min-h-32 flex-col items-center justify-center gap-2 p-6 text-center text-sm'>
            <UsersRound className='size-5 opacity-60' />
            <span>{t('No invited users yet')}</span>
          </div>
        ) : (
          items.map((item) => {
            const displayName = item.display_name?.trim()
            return (
              <div
                key={item.user_id}
                className='grid gap-3 p-4 md:grid-cols-[minmax(200px,1.5fr)_110px_80px_120px_110px_110px_130px] md:items-center'
              >
                <div className='min-w-0'>
                  <div className='truncate text-sm font-medium'>
                    {item.username}
                  </div>
                  <div className='text-muted-foreground truncate text-xs'>
                    {displayName && displayName !== item.username
                      ? displayName
                      : t('User ID {{id}}', { id: item.user_id })}
                  </div>
                </div>
                <div className='text-muted-foreground text-xs md:text-sm'>
                  {formatTimestamp(item.created_at)}
                </div>
                <div className='text-sm tabular-nums'>
                  <span className='text-muted-foreground mr-1 text-xs md:hidden'>
                    {t('Top-ups')}:
                  </span>
                  {item.top_up_count}
                </div>
                <div className='text-sm font-semibold tabular-nums'>
                  <span className='text-muted-foreground mr-1 text-xs font-normal md:hidden'>
                    {t('Reward points contributed')}:
                  </span>
                  {points(item.reward_points)}
                </div>
                <div className='text-sm tabular-nums'>
                  {points(item.pending_points)}
                </div>
                <div className='text-sm tabular-nums'>
                  {points(item.settled_points)}
                </div>
                <div className='text-muted-foreground text-xs md:text-sm'>
                  {item.last_contribution_at > 0
                    ? formatTimestamp(item.last_contribution_at)
                    : t('No contribution yet')}
                </div>

                <div className='bg-muted/30 grid grid-cols-2 gap-2 rounded-lg p-2 text-xs md:hidden'>
                  <div>
                    <div className='text-muted-foreground'>
                      {t('Pending reward points')}
                    </div>
                    <div className='mt-0.5 font-medium tabular-nums'>
                      {points(item.pending_points)}
                    </div>
                  </div>
                  <div>
                    <div className='text-muted-foreground'>
                      {t('Settled reward points')}
                    </div>
                    <div className='mt-0.5 font-medium tabular-nums'>
                      {points(item.settled_points)}
                    </div>
                  </div>
                </div>
              </div>
            )
          })
        )}
      </div>

      {!isLoading && total > 0 ? (
        <div className='bg-muted/20 text-muted-foreground border-t px-4 py-2 text-xs'>
          {t('{{count}} invited users', { count: total })}
        </div>
      ) : null}
    </div>
  )
}
