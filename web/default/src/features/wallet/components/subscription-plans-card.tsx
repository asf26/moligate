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
import { CalendarClock, Check, RefreshCw, Sparkles } from 'lucide-react'
import { useState, useEffect, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  StatusBadge,
  dotColorMap,
  textColorMap,
} from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  getPublicPlans,
  getSelfSubscriptionFull,
  updateBillingPreference,
} from '@/features/subscriptions/api'
import { SubscriptionPurchaseDialog } from '@/features/subscriptions/components/dialogs/subscription-purchase-dialog'
import {
  formatDuration,
  formatResetPeriod,
  formatSubscriptionPrice,
} from '@/features/subscriptions/lib'
import type {
  PlanRecord,
  UserSubscriptionRecord,
} from '@/features/subscriptions/types'
import { formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

import type { PaymentMethod, TopupInfo } from '../types'

interface SubscriptionPlansCardProps {
  topupInfo: TopupInfo | null
  userQuota?: number
  onPurchaseSuccess?: () => void | Promise<void>
}

interface QuotaUsageProps {
  label: string
  amount: number
  used: number
  resetTime?: number
}

function QuotaUsage(props: QuotaUsageProps) {
  const { t } = useTranslation()
  const remaining = Math.max(0, props.amount - props.used)
  const percent =
    props.amount > 0
      ? Math.min(100, Math.round((props.used / props.amount) * 100))
      : 0

  return (
    <div className='min-w-0 border-t px-2 py-3 sm:px-3 lg:border-t-0 lg:px-3'>
      <div>
        <div className='text-xs font-medium'>{props.label}</div>
        <div className='text-muted-foreground mt-0.5 text-xs tabular-nums'>
          {formatQuota(props.used)} / {formatQuota(props.amount)}
        </div>
      </div>
      <div className='mt-2 min-w-0'>
        <Progress value={percent} className='h-1.5' />
        {props.resetTime && props.resetTime > 0 ? (
          <div className='text-muted-foreground mt-1.5 flex items-center gap-1 text-[10px] leading-4'>
            <CalendarClock className='size-3 shrink-0' aria-hidden='true' />
            <span className='truncate'>
              {t('Next reset')}:{' '}
              {new Date(props.resetTime * 1000).toLocaleString()}
            </span>
          </div>
        ) : null}
      </div>
      <div className='mt-2 flex items-baseline justify-between'>
        <div className='order-2 text-sm font-semibold tabular-nums'>
          {formatQuota(remaining)}
        </div>
        <div className='text-muted-foreground text-[11px]'>
          {t('Remaining')}
        </div>
      </div>
    </div>
  )
}

interface PlanQuotaValueProps {
  label: string
  amount: number
}

function PlanQuotaValue(props: PlanQuotaValueProps) {
  const { t } = useTranslation()

  return (
    <div className='flex min-w-0 items-center justify-between gap-3 py-2'>
      <span className='text-muted-foreground text-xs'>{props.label}</span>
      <span className='max-w-full text-xs font-semibold tabular-nums'>
        {props.amount > 0 ? formatQuota(props.amount) : t('Unlimited')}
      </span>
    </div>
  )
}

function getEpayMethods(payMethods: PaymentMethod[] = []): PaymentMethod[] {
  return payMethods.filter(
    (m) => m?.type && m.type !== 'stripe' && m.type !== 'creem'
  )
}

function getBillingPreferenceLabel(
  preference: string,
  t: (key: string) => string
): string {
  switch (preference) {
    case 'subscription_first':
      return t('Subscription First')
    case 'wallet_first':
      return t('Wallet First')
    case 'subscription_only':
      return t('Subscription Only')
    case 'wallet_only':
      return t('Wallet Only')
    default:
      return preference
  }
}

export function SubscriptionPlansCard({
  topupInfo,
  userQuota,
  onPurchaseSuccess,
}: SubscriptionPlansCardProps) {
  const { t } = useTranslation()

  const [plans, setPlans] = useState<PlanRecord[]>([])
  const [activeSubscriptions, setActiveSubscriptions] = useState<
    UserSubscriptionRecord[]
  >([])
  const [allSubscriptions, setAllSubscriptions] = useState<
    UserSubscriptionRecord[]
  >([])
  const [billingPreference, setBillingPreference] =
    useState('subscription_first')
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  const [purchaseOpen, setPurchaseOpen] = useState(false)
  const [selectedPlan, setSelectedPlan] = useState<PlanRecord | null>(null)

  const enableStripe = !!topupInfo?.enable_stripe_topup
  const enableCreem = !!topupInfo?.enable_creem_topup
  const enableWaffoPancake = !!topupInfo?.enable_waffo_pancake_topup
  const enableOnlineTopUp = !!topupInfo?.enable_online_topup
  const epayMethods = useMemo(
    () => getEpayMethods(topupInfo?.pay_methods),
    [topupInfo?.pay_methods]
  )

  const fetchPlans = useCallback(async () => {
    try {
      const res = await getPublicPlans()
      if (res.success) {
        setPlans(res.data || [])
      }
    } catch {
      setPlans([])
    }
  }, [])

  const fetchSelfSubscription = useCallback(async () => {
    try {
      const res = await getSelfSubscriptionFull()
      if (res.success && res.data) {
        setBillingPreference(
          res.data.billing_preference || 'subscription_first'
        )
        setActiveSubscriptions(res.data.subscriptions || [])
        setAllSubscriptions(res.data.all_subscriptions || [])
      }
    } catch {
      // ignore
    }
  }, [])

  useEffect(() => {
    const init = async () => {
      setLoading(true)
      await Promise.all([fetchPlans(), fetchSelfSubscription()])
      setLoading(false)
    }
    init()
  }, [fetchPlans, fetchSelfSubscription])

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      await fetchSelfSubscription()
    } finally {
      setRefreshing(false)
    }
  }

  const handlePreferenceChange = async (pref: string) => {
    const previous = billingPreference
    setBillingPreference(pref)
    try {
      const res = await updateBillingPreference(pref)
      if (res.success) {
        toast.success(t('Updated successfully'))
        const normalized = res.data?.billing_preference || pref
        setBillingPreference(normalized)
      } else {
        toast.error(res.message || t('Update failed'))
        setBillingPreference(previous)
      }
    } catch {
      toast.error(t('Request failed'))
      setBillingPreference(previous)
    }
  }

  const hasActive = activeSubscriptions.length > 0
  const hasAny = allSubscriptions.length > 0
  const disablePref = !hasActive
  const isSubPref =
    billingPreference === 'subscription_first' ||
    billingPreference === 'subscription_only'
  const displayPref =
    disablePref && isSubPref ? 'wallet_first' : billingPreference

  const planPurchaseCountMap = useMemo(() => {
    const map = new Map<number, number>()
    for (const sub of allSubscriptions) {
      const planId = sub?.subscription?.plan_id
      if (!planId) continue
      map.set(planId, (map.get(planId) || 0) + 1)
    }
    return map
  }, [allSubscriptions])

  const planTitleMap = useMemo(() => {
    const map = new Map<number, string>()
    for (const p of plans) {
      if (p?.plan?.id) {
        map.set(p.plan.id, p.plan.title || '')
      }
    }
    return map
  }, [plans])

  const getRemainingDays = (sub: UserSubscriptionRecord) => {
    const endTime = sub?.subscription?.end_time || 0
    if (!endTime) return 0
    const now = Date.now() / 1000
    return Math.max(0, Math.ceil((endTime - now) / 86400))
  }

  if (loading) {
    return (
      <div className='flex flex-col gap-5'>
        <Skeleton className='h-40 w-full rounded-xl' />
        <div className='grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3'>
          {['first', 'second', 'third'].map((key) => (
            <Skeleton key={key} className='h-80 w-full rounded-xl' />
          ))}
        </div>
      </div>
    )
  }

  return (
    <>
      <div className='flex flex-col gap-6'>
        <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
          <CardHeader className='bg-muted/20 border-b px-4 py-3.5 sm:px-5'>
            <div className='flex flex-wrap items-center justify-between gap-3'>
              <div className='flex min-w-0 flex-wrap items-center gap-2'>
                <CardTitle className='text-sm'>
                  {t('My Subscriptions')}
                </CardTitle>
                <span className='flex items-center gap-1.5 text-xs font-medium'>
                  <span
                    className={cn(
                      'size-1.5 shrink-0 rounded-full',
                      hasActive ? dotColorMap.success : dotColorMap.neutral
                    )}
                    aria-hidden='true'
                  />
                  {hasActive ? (
                    <span className={cn(textColorMap.success)}>
                      {activeSubscriptions.length} {t('active')}
                    </span>
                  ) : (
                    <span className='text-muted-foreground'>
                      {t('No Active')}
                    </span>
                  )}
                  {allSubscriptions.length > activeSubscriptions.length && (
                    <>
                      <span className='text-muted-foreground/30'>·</span>
                      <span className='text-muted-foreground'>
                        {allSubscriptions.length - activeSubscriptions.length}{' '}
                        {t('expired')}
                      </span>
                    </>
                  )}
                </span>
              </div>
              <div className='flex w-full items-center gap-2 sm:w-auto'>
                <span className='text-muted-foreground hidden text-xs lg:inline'>
                  {t('Preferences')}
                </span>
                <Select
                  items={[
                    {
                      value: 'subscription_first',
                      label: (
                        <>
                          {getBillingPreferenceLabel('subscription_first', t)}
                          {disablePref ? ` (${t('No Active')})` : ''}
                        </>
                      ),
                    },
                    {
                      value: 'wallet_first',
                      label: getBillingPreferenceLabel('wallet_first', t),
                    },
                    {
                      value: 'subscription_only',
                      label: (
                        <>
                          {getBillingPreferenceLabel('subscription_only', t)}
                          {disablePref ? ` (${t('No Active')})` : ''}
                        </>
                      ),
                    },
                    {
                      value: 'wallet_only',
                      label: getBillingPreferenceLabel('wallet_only', t),
                    },
                  ]}
                  value={displayPref}
                  onValueChange={(v) => v !== null && handlePreferenceChange(v)}
                >
                  <SelectTrigger className='h-8 flex-1 text-xs sm:w-[150px] sm:flex-none'>
                    <SelectValue>
                      {getBillingPreferenceLabel(displayPref, t)}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      <SelectItem
                        value='subscription_first'
                        disabled={disablePref}
                      >
                        {getBillingPreferenceLabel('subscription_first', t)}
                        {disablePref ? ` (${t('No Active')})` : ''}
                      </SelectItem>
                      <SelectItem value='wallet_first'>
                        {getBillingPreferenceLabel('wallet_first', t)}
                      </SelectItem>
                      <SelectItem
                        value='subscription_only'
                        disabled={disablePref}
                      >
                        {getBillingPreferenceLabel('subscription_only', t)}
                        {disablePref ? ` (${t('No Active')})` : ''}
                      </SelectItem>
                      <SelectItem value='wallet_only'>
                        {getBillingPreferenceLabel('wallet_only', t)}
                      </SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Button
                        variant='ghost'
                        size='icon'
                        className='size-8'
                        onClick={handleRefresh}
                        disabled={refreshing}
                        aria-label={t('Refresh')}
                      />
                    }
                  >
                    <RefreshCw
                      className={cn('size-3.5', refreshing && 'animate-spin')}
                    />
                  </TooltipTrigger>
                  <TooltipContent>{t('Refresh')}</TooltipContent>
                </Tooltip>
              </div>
            </div>
          </CardHeader>

          {disablePref && isSubPref && (
            <p className='text-muted-foreground border-b px-4 py-3 text-xs sm:px-5'>
              {t(
                'Preference saved as {{pref}}, but no active subscription. Wallet will be used automatically.',
                {
                  pref:
                    billingPreference === 'subscription_only'
                      ? t('Subscription Only')
                      : t('Subscription First'),
                }
              )}
            </p>
          )}

          <CardContent className='flex flex-col gap-3 p-4 sm:px-5 sm:py-4'>
            {hasAny ? (
              <>
                {allSubscriptions.map((sub) => {
                  const subscription = sub.subscription
                  const totalAmount = Number(subscription?.amount_total || 0)
                  const usedAmount = Number(subscription?.amount_used || 0)
                  const planTitle =
                    planTitleMap.get(subscription?.plan_id) || ''
                  const remainDays = getRemainingDays(sub)
                  const now = Date.now() / 1000
                  const isExpired = (subscription?.end_time || 0) < now
                  const isCancelled = subscription?.status === 'cancelled'
                  const isActive =
                    subscription?.status === 'active' && !isExpired
                  const nextResetTime = subscription?.next_reset_time ?? 0
                  const quotaUsages: QuotaUsageProps[] = [
                    totalAmount > 0
                      ? {
                          label: t('Total Quota'),
                          amount: totalAmount,
                          used: usedAmount,
                          resetTime: nextResetTime,
                        }
                      : null,
                    Number(subscription?.daily_amount || 0) > 0
                      ? {
                          label: t('Daily Quota'),
                          amount: Number(subscription?.daily_amount || 0),
                          used: Number(subscription?.daily_used || 0),
                          resetTime: subscription?.daily_reset_time,
                        }
                      : null,
                    Number(subscription?.weekly_amount || 0) > 0
                      ? {
                          label: t('Weekly Quota'),
                          amount: Number(subscription?.weekly_amount || 0),
                          used: Number(subscription?.weekly_used || 0),
                          resetTime: subscription?.weekly_reset_time,
                        }
                      : null,
                    Number(subscription?.monthly_amount || 0) > 0
                      ? {
                          label: t('Monthly Quota'),
                          amount: Number(subscription?.monthly_amount || 0),
                          used: Number(subscription?.monthly_used || 0),
                          resetTime: subscription?.monthly_reset_time,
                        }
                      : null,
                  ].filter((quota) => quota !== null)
                  let statusBadge = (
                    <StatusBadge
                      label={t('Expired')}
                      variant='neutral'
                      copyable={false}
                    />
                  )
                  if (isActive) {
                    statusBadge = (
                      <StatusBadge
                        label={t('Active')}
                        variant='success'
                        copyable={false}
                      />
                    )
                  } else if (isCancelled) {
                    statusBadge = (
                      <StatusBadge
                        label={t('Cancelled')}
                        variant='neutral'
                        copyable={false}
                      />
                    )
                  }

                  let endTimeLabel = t('Expired at')
                  if (isActive) {
                    endTimeLabel = t('Until')
                  } else if (isCancelled) {
                    endTimeLabel = t('Cancelled at')
                  }

                  if (!isActive) {
                    return (
                      <div
                        key={subscription?.id}
                        className='bg-muted/25 flex flex-wrap items-center justify-between gap-2 rounded-lg px-3 py-2.5 text-xs'
                      >
                        <div className='flex min-w-0 flex-wrap items-center gap-2'>
                          <span className='font-medium break-words'>
                            {planTitle
                              ? `${planTitle} · ${t('Subscription')} #${subscription?.id}`
                              : `${t('Subscription')} #${subscription?.id}`}
                          </span>
                          {statusBadge}
                        </div>
                        <span className='text-muted-foreground shrink-0'>
                          {endTimeLabel}{' '}
                          {new Date(
                            (subscription?.end_time || 0) * 1000
                          ).toLocaleString()}
                        </span>
                      </div>
                    )
                  }

                  return (
                    <div key={subscription?.id} className='text-xs'>
                      <div className='flex flex-wrap items-start justify-between gap-3 border-b pb-3.5'>
                        <div className='min-w-0'>
                          <div className='flex min-w-0 flex-wrap items-center gap-2'>
                            <span className='text-sm font-semibold break-words'>
                              {planTitle
                                ? `${planTitle} · ${t('Subscription')} #${subscription?.id}`
                                : `${t('Subscription')} #${subscription?.id}`}
                            </span>
                            {statusBadge}
                          </div>
                          <div className='text-muted-foreground mt-1.5 flex items-center gap-1.5'>
                            <CalendarClock
                              className='size-3.5'
                              aria-hidden='true'
                            />
                            {endTimeLabel}{' '}
                            {new Date(
                              (subscription?.end_time || 0) * 1000
                            ).toLocaleString()}
                          </div>
                        </div>
                        {isActive && (
                          <div className='shrink-0 text-right'>
                            <div className='text-lg leading-5 font-semibold tabular-nums'>
                              {remainDays}
                            </div>
                            <div className='text-muted-foreground text-xs'>
                              {t('{{count}} days remaining', {
                                count: remainDays,
                              })}
                            </div>
                          </div>
                        )}
                      </div>
                      {quotaUsages.length > 0 ? (
                        <div className='grid grid-cols-2 gap-x-3 lg:grid-cols-4 lg:gap-x-0 lg:divide-x'>
                          {quotaUsages.map((quota) => (
                            <QuotaUsage key={quota.label} {...quota} />
                          ))}
                        </div>
                      ) : (
                        <div className='text-muted-foreground py-4'>
                          {t('Total Quota')}: {t('Unlimited')}
                        </div>
                      )}
                    </div>
                  )
                })}
              </>
            ) : (
              <div className='flex min-h-28 items-center justify-center text-sm'>
                <span className='text-muted-foreground'>
                  {t('Subscribe to a plan for model access')}
                </span>
              </div>
            )}
          </CardContent>
        </Card>

        {plans.length > 0 ? (
          <section aria-label={t('Subscription Plans')}>
            <div className='mb-3 flex items-end justify-between gap-4'>
              <div>
                <h3 className='text-base font-semibold'>
                  {t('Subscription Plans')}
                </h3>
                <p className='text-muted-foreground mt-0.5 text-xs'>
                  {t('Subscribe to a plan for model access')}
                </p>
              </div>
            </div>
            <div className='grid grid-cols-[repeat(auto-fill,minmax(260px,300px))] gap-3'>
              {plans.map((p, index) => {
                const plan = p?.plan
                if (!plan) return null
                const totalAmount = Number(plan.total_amount || 0)
                const dailyAmount = Number(plan.daily_amount || 0)
                const weeklyAmount = Number(plan.weekly_amount || 0)
                const monthlyAmount = Number(plan.monthly_amount || 0)
                const price = formatSubscriptionPrice(plan)
                const isPopular = index === 0 && plans.length > 1
                const limit = Number(plan.max_purchase_per_user || 0)
                const count = planPurchaseCountMap.get(plan.id) || 0
                const reached = limit > 0 && count >= limit
                const resetPeriod = formatResetPeriod(plan, t)

                return (
                  <Card
                    key={plan.id}
                    data-card-hover='false'
                    className={cn(
                      'relative gap-0 overflow-hidden py-0',
                      isPopular && 'border-foreground/30 shadow-sm'
                    )}
                  >
                    {isPopular && (
                      <div className='bg-foreground text-background flex h-7 items-center justify-center gap-1.5 text-[11px] font-medium'>
                        <Sparkles className='size-3.5' aria-hidden='true' />
                        {t('Recommended')}
                      </div>
                    )}
                    <CardHeader className='border-b px-3.5 py-3.5'>
                      <div className='min-h-9'>
                        <CardTitle className='text-sm'>
                          {plan.title || t('Subscription Plans')}
                        </CardTitle>
                        {plan.subtitle && (
                          <p className='text-muted-foreground mt-0.5 line-clamp-2 text-xs leading-5'>
                            {plan.subtitle}
                          </p>
                        )}
                      </div>
                      <div className='mt-2.5 flex items-end gap-1.5'>
                        <span className='text-xl font-semibold tabular-nums'>
                          {price}
                        </span>
                        <span className='text-muted-foreground pb-1 text-xs'>
                          / {formatDuration(plan, t)}
                        </span>
                      </div>
                    </CardHeader>
                    <CardContent className='flex flex-1 flex-col px-3.5 py-2.5'>
                      <div className='divide-y'>
                        <PlanQuotaValue
                          label={t('Total Quota')}
                          amount={totalAmount}
                        />
                        <PlanQuotaValue
                          label={t('Daily Quota')}
                          amount={dailyAmount}
                        />
                        <PlanQuotaValue
                          label={t('Weekly Quota')}
                          amount={weeklyAmount}
                        />
                        <PlanQuotaValue
                          label={t('Monthly Quota')}
                          amount={monthlyAmount}
                        />
                      </div>
                      <div className='text-muted-foreground mt-3 flex min-h-8 items-start gap-2 text-[11px] leading-4'>
                        <Check
                          className='text-foreground mt-0.5 size-3.5 shrink-0'
                          aria-hidden='true'
                        />
                        <span>
                          {resetPeriod !== t('No Reset')
                            ? `${t('Total Quota Reset')}: ${resetPeriod}`
                            : `${t('Validity Period')}: ${formatDuration(plan, t)}`}
                        </span>
                      </div>
                      <div className='mt-3'>
                        {reached ? (
                          <Tooltip>
                            <TooltipTrigger
                              render={
                                <Button
                                  variant='outline'
                                  size='sm'
                                  className='h-8 w-full text-xs'
                                  disabled
                                />
                              }
                            >
                              {t('Limit Reached')}
                            </TooltipTrigger>
                            <TooltipContent>
                              {t('Purchase limit reached')} ({count}/{limit})
                            </TooltipContent>
                          </Tooltip>
                        ) : (
                          <Button
                            variant={isPopular ? 'default' : 'outline'}
                            size='sm'
                            className='h-8 w-full text-xs'
                            onClick={() => {
                              setSelectedPlan(p)
                              setPurchaseOpen(true)
                            }}
                          >
                            {t('Subscribe Now')}
                          </Button>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          </section>
        ) : (
          <p className='text-muted-foreground py-4 text-center text-sm'>
            {t('No plans available')}
          </p>
        )}
      </div>

      <SubscriptionPurchaseDialog
        open={purchaseOpen}
        onOpenChange={(open) => {
          setPurchaseOpen(open)
          if (!open) {
            fetchSelfSubscription()
          }
        }}
        plan={selectedPlan}
        enableStripe={enableStripe}
        enableCreem={enableCreem}
        enableWaffoPancake={enableWaffoPancake}
        enableOnlineTopUp={enableOnlineTopUp}
        epayMethods={epayMethods}
        userQuota={userQuota}
        onPurchaseSuccess={onPurchaseSuccess}
        purchaseLimit={
          selectedPlan?.plan?.max_purchase_per_user
            ? Number(selectedPlan.plan.max_purchase_per_user)
            : undefined
        }
        purchaseCount={
          selectedPlan?.plan?.id
            ? planPurchaseCountMap.get(selectedPlan.plan.id)
            : undefined
        }
      />
    </>
  )
}
