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
import { Share2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'

import type { UserWalletData } from '../types'

interface AffiliateRewardsCardProps {
  user: UserWalletData | null
  affiliateLink: string
  onTransfer: () => void
  complianceConfirmed?: boolean
  loading?: boolean
}

export function AffiliateRewardsCard({
  user,
  affiliateLink,
  onTransfer,
  complianceConfirmed = true,
  loading,
}: AffiliateRewardsCardProps) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <Card
        data-card-hover='false'
        className='border-primary/15 bg-primary/[0.03] py-0'
      >
        <CardContent className='space-y-5 p-4 sm:p-5'>
          <div className='flex items-center gap-3'>
            <Skeleton className='size-10 rounded-xl' />
            <div className='space-y-2'>
              <Skeleton className='h-5 w-40' />
              <Skeleton className='h-4 w-64 max-w-[60vw]' />
            </div>
          </div>
          <Skeleton className='h-20 rounded-xl' />
        </CardContent>
      </Card>
    )
  }

  const hasRewards = (user?.aff_quota ?? 0) > 0
  return (
    <Card
      data-card-hover='false'
      className='border-primary/20 bg-primary/[0.035] py-0'
    >
      <CardContent className='p-4 sm:p-5'>
        <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between'>
          <div className='flex min-w-0 items-start gap-3'>
            <IconBadge tone='chart-3' size='lg'>
              <Share2 />
            </IconBadge>
            <div className='min-w-0'>
              <h3 className='text-base font-semibold tracking-tight sm:text-lg'>
                {t('Registration Invite Rewards')}
              </h3>
              <p className='text-muted-foreground mt-1 max-w-2xl text-xs leading-5 sm:text-sm'>
                {t(
                  'Earn quota rewards when invited users register. Transfer accumulated rewards to your balance anytime.'
                )}
              </p>
            </div>
          </div>

          {hasRewards && (
            <Button
              onClick={onTransfer}
              disabled={!complianceConfirmed}
              size='sm'
              className='w-full shrink-0 sm:w-auto'
            >
              {t('Transfer to Balance')}
            </Button>
          )}
        </div>

        <div className='bg-background/80 mt-5 rounded-xl border p-3 sm:p-4'>
          <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
            <div className='min-w-0'>
              <p className='text-muted-foreground text-xs font-medium tracking-[0.14em] uppercase'>
                {t('Referral link:')}
              </p>
              <p className='text-muted-foreground mt-1 text-xs'>
                {t('Share your link and earn rewards')}
              </p>
            </div>
            <div className='flex w-full min-w-0 items-center gap-2 sm:max-w-[34rem]'>
              <Input
                value={affiliateLink}
                readOnly
                aria-label={t('Referral link:')}
                className='border-muted bg-background h-9 min-w-0 flex-1 font-mono text-xs'
              />
              <CopyButton
                value={affiliateLink}
                variant='outline'
                className='bg-background size-9'
                iconClassName='size-4'
                tooltip={t('Copy referral link')}
                aria-label={t('Copy referral link')}
              />
            </div>
          </div>
        </div>

        {!complianceConfirmed ? (
          <p className='text-muted-foreground mt-3 text-xs leading-5'>
            {t(
              'Referral reward transfer is disabled until the administrator confirms compliance terms.'
            )}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
