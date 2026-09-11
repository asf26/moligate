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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { HandCoins, Percent, UsersRound } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import {
  getSelfAffiliateInvitees,
  getSelfAffiliateRewardPointSettlements,
  getSelfAffiliateSummary,
  quoteSelfAffiliateRewardPoints,
  redeemSelfAffiliateRewardPoints,
} from '@/features/affiliate-commissions/api'
import { AffiliateInviteesTable } from '@/features/affiliate-commissions/components/affiliate-invitees-table'
import type { AffiliateRewardPointSettlement } from '@/features/affiliate-commissions/types'
import { useDebounce } from '@/hooks'
import { getSelf } from '@/lib/api'
import { formatTimestamp } from '@/lib/format'

import { formatRewardPoints, formatWalletQuota } from '../lib'

function formatPoints(points: number | undefined) {
  return formatRewardPoints(points || 0)
}

function getProcessedPoints(
  summary: { redeemed_points?: number; settled_points?: number } | undefined
) {
  return summary?.redeemed_points ?? summary?.settled_points ?? 0
}

function getSettlementLabelKey(row: AffiliateRewardPointSettlement) {
  return row.settlement_type === 'wallet'
    ? 'Wallet redemption'
    : 'Processed by admin'
}

export function AffiliateCommissionsCard() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [redeemOpen, setRedeemOpen] = useState(false)
  const [redeemPointsInput, setRedeemPointsInput] = useState('')
  const summaryQuery = useQuery({
    queryKey: ['self-affiliate-summary'],
    queryFn: getSelfAffiliateSummary,
  })
  const settlementsQuery = useQuery({
    queryKey: ['self-affiliate-reward-settlements'],
    queryFn: () =>
      getSelfAffiliateRewardPointSettlements({ p: 1, page_size: 8 }),
  })
  const inviteesQuery = useQuery({
    queryKey: ['self-affiliate-invitees'],
    queryFn: () => getSelfAffiliateInvitees({ p: 1, page_size: 50 }),
  })

  const summary = summaryQuery.data?.data
  const rows = settlementsQuery.data?.data?.items || []
  const inviteeRows = inviteesQuery.data?.data?.items || []
  const inviteeTotal =
    inviteesQuery.data?.data?.total || summary?.invite_count || 0
  const loading =
    summaryQuery.isLoading ||
    settlementsQuery.isLoading ||
    inviteesQuery.isLoading
  const pendingPoints = summary?.pending_points || 0
  const processedPoints = getProcessedPoints(summary)

  const formatRate = (rateBps: number | undefined) =>
    `${((rateBps || 0) / 100).toFixed(2).replace(/\.00$/, '')}%`
  const redeemPoints = Number(redeemPointsInput)
  const redeemPointsEntered = redeemPointsInput.trim().length > 0
  const redeemPointsValid =
    redeemPointsEntered &&
    Number.isInteger(redeemPoints) &&
    redeemPoints >= 1 &&
    redeemPoints <= pendingPoints
  const redeemPointsError =
    redeemPointsEntered && !redeemPointsValid
      ? t('Points must be between 1 and {{max}}', {
          max: formatPoints(pendingPoints),
        })
      : undefined
  const debouncedRedeemPoints = useDebounce(
    redeemPointsValid ? redeemPoints : 0
  )
  const quoteQuery = useQuery({
    queryKey: ['self-affiliate-reward-quote', debouncedRedeemPoints],
    queryFn: () =>
      quoteSelfAffiliateRewardPoints({ points: debouncedRedeemPoints }),
    enabled:
      redeemOpen && redeemPointsValid && debouncedRedeemPoints === redeemPoints,
  })
  const quote =
    quoteQuery.data?.success &&
    debouncedRedeemPoints === redeemPoints &&
    redeemPointsValid
      ? quoteQuery.data.data
      : undefined
  const quoteError =
    quoteQuery.data && !quoteQuery.data.success ? quoteQuery.data.message : ''
  const redeemMutation = useMutation({
    mutationFn: (points: number) => redeemSelfAffiliateRewardPoints({ points }),
    onSuccess: async (res) => {
      if (!res.success) {
        toast.error(res.message || t('Failed to redeem reward points'))
        return
      }
      const redeemedPoints = formatPoints(res.data?.redeemed_points)
      const walletAmount = formatWalletQuota(res.data?.redeemed_quota || 0)
      toast.success(
        t('Redeemed {{points}} points, added {{walletAmount}}', {
          points: redeemedPoints,
          walletAmount,
        })
      )
      setRedeemOpen(false)
      setRedeemPointsInput('')
      await getSelf()
      queryClient.invalidateQueries({ queryKey: ['self-affiliate-summary'] })
      queryClient.invalidateQueries({
        queryKey: ['self-affiliate-reward-settlements'],
      })
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to redeem reward points'))
    },
  })

  const openRedeemDialog = () => {
    setRedeemPointsInput('')
    setRedeemOpen(true)
  }

  const handleRedeemOpenChange = (open: boolean) => {
    setRedeemOpen(open)
    if (!open) {
      setRedeemPointsInput('')
    }
  }

  const handleConfirmRedeem = () => {
    if (!redeemPointsValid) {
      toast.error(
        redeemPointsEntered
          ? t('Points must be between 1 and {{max}}', {
              max: formatPoints(pendingPoints),
            })
          : t('Enter points to redeem')
      )
      return
    }
    redeemMutation.mutate(redeemPoints)
  }

  return (
    <>
      <Card className='bg-background py-0'>
        <CardContent className='space-y-5 p-4 sm:p-6'>
          <div className='flex min-w-0 items-center gap-2.5'>
            <div className='bg-background flex size-8 shrink-0 items-center justify-center rounded-lg border'>
              <HandCoins className='text-muted-foreground size-4' />
            </div>
            <div className='min-w-0'>
              <h3 className='truncate text-sm font-semibold'>
                {t('Top-up Reward Points')}
              </h3>
              <p className='text-muted-foreground line-clamp-1 text-xs'>
                {t(
                  'Reward points can be redeemed to your wallet at 1 point = 500000 tokens.'
                )}
              </p>
            </div>
            <Button
              size='sm'
              className='ml-auto h-8 shrink-0'
              disabled={
                loading || pendingPoints <= 0 || redeemMutation.isPending
              }
              onClick={openRedeemDialog}
            >
              {redeemMutation.isPending ? t('Redeeming...') : t('Redeem')}
            </Button>
          </div>

          <div className='grid gap-2 sm:grid-cols-4'>
            {[
              {
                label: t('Pending Points'),
                value: formatPoints(summary?.pending_points),
                unit: t('points'),
              },
              {
                label: t('Redeemed Points'),
                value: formatPoints(processedPoints),
                unit: t('points'),
              },
              {
                label: t('Total Points'),
                value: formatPoints(summary?.total_points),
                unit: t('points'),
              },
              {
                label: t('Invited users'),
                value: String(inviteeTotal),
                unit: '',
              },
            ].map((item) => (
              <div
                key={item.label}
                className='bg-muted/20 rounded-xl border p-3.5'
              >
                <div className='text-muted-foreground text-xs font-medium'>
                  {item.label}
                </div>
                {loading ? (
                  <Skeleton className='mt-2 h-5 w-24' />
                ) : (
                  <div className='mt-1 text-sm font-semibold tabular-nums'>
                    {item.value} {item.unit}
                  </div>
                )}
              </div>
            ))}
          </div>

          <div className='space-y-3 border-t pt-5'>
            <div className='flex items-center gap-2'>
              <Percent className='text-primary size-4' />
              <div>
                <h4 className='text-sm font-semibold'>
                  {t('Invitation reward rates')}
                </h4>
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'Rates apply to each credited wallet top-up from your invitees.'
                  )}
                </p>
              </div>
            </div>
            <div className='grid gap-2 sm:grid-cols-2'>
              {[
                [
                  t('Level 1 reward rate'),
                  formatRate(summary?.level1_rate_bps),
                ],
                [
                  t('Points per paid unit'),
                  String(summary?.points_per_amount_unit || 0),
                ],
              ].map(([label, value]) => (
                <div key={label} className='bg-muted/20 rounded-xl border p-3'>
                  <div className='text-muted-foreground text-xs'>{label}</div>
                  <div className='mt-1 text-lg font-semibold tabular-nums'>
                    {value}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className='space-y-3 border-t pt-5'>
            <div className='flex items-center gap-2'>
              <UsersRound className='text-primary size-4' />
              <div>
                <h4 className='text-sm font-semibold'>{t('Invited users')}</h4>
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'See each invitee and the reward points their top-ups contributed.'
                  )}
                </p>
              </div>
            </div>
            <AffiliateInviteesTable
              items={inviteeRows}
              total={inviteeTotal}
              isLoading={inviteesQuery.isLoading}
            />
          </div>

          <div className='space-y-3 border-t pt-5'>
            <h4 className='text-sm font-semibold'>
              {t('Invitation activity')}
            </h4>
            <div className='bg-muted/10 divide-y rounded-xl border'>
              {loading ? (
                Array.from({ length: 3 }).map((_, index) => (
                  <div key={index} className='flex items-center gap-3 p-3'>
                    <Skeleton className='h-4 w-24' />
                    <Skeleton className='h-4 flex-1' />
                    <Skeleton className='h-4 w-20' />
                  </div>
                ))
              ) : rows.length === 0 ? (
                <div className='text-muted-foreground p-3 text-sm'>
                  {t('No reward point activity')}
                </div>
              ) : (
                rows.map((row) => (
                  <div
                    key={row.id}
                    className='grid gap-2 p-3 text-sm sm:grid-cols-[140px_minmax(0,1fr)_160px] sm:items-center'
                  >
                    <Badge
                      variant={
                        row.settlement_type === 'wallet'
                          ? 'secondary'
                          : 'outline'
                      }
                    >
                      {t(getSettlementLabelKey(row))}
                    </Badge>
                    <div className='min-w-0'>
                      <div className='font-medium tabular-nums'>
                        {formatPoints(row.points)} {t('points')}
                      </div>
                      <div className='text-muted-foreground text-xs'>
                        {row.wallet_quota > 0
                          ? t('Wallet credit: {{amount}}', {
                              amount: formatWalletQuota(row.wallet_quota),
                            })
                          : t('Processed points')}
                      </div>
                    </div>
                    <div className='text-muted-foreground text-xs'>
                      {formatTimestamp(row.settled_at || row.created_at)}
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      <Dialog open={redeemOpen} onOpenChange={handleRedeemOpenChange}>
        <DialogContent className='sm:max-w-md'>
          <form
            className='contents'
            onSubmit={(event) => {
              event.preventDefault()
              handleConfirmRedeem()
            }}
          >
            <DialogHeader>
              <DialogTitle>{t('Redeem points')}</DialogTitle>
              <DialogDescription>
                {t('Choose how many reward points to redeem into your wallet.')}
              </DialogDescription>
            </DialogHeader>

            <div className='space-y-4'>
              <div className='bg-muted/30 flex items-center justify-between rounded-lg border p-3'>
                <span className='text-muted-foreground text-sm'>
                  {t('Available points')}
                </span>
                <span className='text-sm font-semibold tabular-nums'>
                  {formatPoints(pendingPoints)} {t('points')}
                </span>
              </div>

              <div className='space-y-2'>
                <Label htmlFor='affiliate-redeem-points'>
                  {t('Points to redeem')}
                </Label>
                <div className='flex gap-2'>
                  <Input
                    id='affiliate-redeem-points'
                    type='number'
                    inputMode='numeric'
                    min={1}
                    max={pendingPoints}
                    step={1}
                    value={redeemPointsInput}
                    placeholder={t('Enter points to redeem')}
                    onChange={(event) =>
                      setRedeemPointsInput(event.target.value)
                    }
                  />
                  <Button
                    type='button'
                    variant='outline'
                    className='shrink-0'
                    onClick={() => setRedeemPointsInput(String(pendingPoints))}
                  >
                    {t('All')}
                  </Button>
                </div>
                {redeemPointsError ? (
                  <p className='text-destructive text-xs'>
                    {redeemPointsError}
                  </p>
                ) : null}
              </div>

              <div className='rounded-lg border p-3'>
                <div className='text-muted-foreground text-xs font-medium'>
                  {t('Estimated wallet credit')}
                </div>
                {quoteQuery.isFetching ? (
                  <Skeleton className='mt-2 h-6 w-28' />
                ) : quote ? (
                  <div className='mt-1 text-lg font-semibold tabular-nums'>
                    {formatWalletQuota(quote.redeemed_quota)}
                  </div>
                ) : (
                  <div className='text-muted-foreground mt-1 text-sm'>
                    {quoteError || t('Enter points to redeem')}
                  </div>
                )}
              </div>
            </div>

            <DialogFooter>
              <DialogClose render={<Button variant='outline' type='button' />}>
                {t('Cancel')}
              </DialogClose>
              <Button
                type='submit'
                disabled={
                  !redeemPointsValid ||
                  redeemMutation.isPending ||
                  quoteQuery.isFetching
                }
              >
                {redeemMutation.isPending
                  ? t('Redeeming...')
                  : t('Confirm redemption')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  )
}
