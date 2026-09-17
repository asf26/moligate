import { Link } from '@tanstack/react-router'
/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your
option) any later version.

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
  ArrowRight,
  CalendarClock,
  CheckCircle2,
  Clock3,
  Coins,
  Gift,
  RefreshCw,
  WalletCards,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
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
import { getUserQuotaDates } from '@/features/dashboard/api'
import {
  getPublicPlans,
  getSelfSubscriptionFull,
  updateBillingPreference,
} from '@/features/subscriptions/api'
import type {
  PlanRecord,
  UserSubscription,
  UserSubscriptionRecord,
} from '@/features/subscriptions/types'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

import { formatWalletQuota } from './lib'
import {
  aggregateUsageByModel,
  getSubscriptionUsageWindow,
  type ModelUsageSummary,
} from './lib/subscription-usage'

function formatResetDate(resetTime: number | undefined): string {
  if (!resetTime || resetTime <= 0) return ''
  return new Date(resetTime * 1000).toLocaleString(undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

function getPreferenceLabel(preference: string, t: (key: string) => string) {
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

function SubscriptionQuotaMeter(props: {
  label: string
  amount: number
  used: number
  resetTime?: number
  tone: 'blue' | 'coral' | 'mint'
}) {
  const { t } = useTranslation()
  const amount =
    Number.isFinite(props.amount) && props.amount > 0 ? props.amount : 0
  const used = Number.isFinite(props.used) && props.used > 0 ? props.used : 0
  const remaining = Math.max(0, amount - used)
  const percent =
    amount > 0 ? Math.min(100, Math.round((used / amount) * 100)) : 0
  const resetLabel = props.resetTime
    ? `${t('Next reset')}: ${formatResetDate(props.resetTime)}`
    : t('Remaining')

  return (
    <div
      className={cn(
        'subscription-usage-meter',
        `subscription-quota-meter-${props.tone}`
      )}
    >
      <div className='flex items-baseline justify-between gap-2'>
        <span className='subscription-quota-meter-label truncate'>
          {props.label}
        </span>
        <span className='subscription-quota-meter-value shrink-0 font-[family-name:var(--font-geist-mono)] tabular-nums'>
          {amount > 0
            ? `${formatWalletQuota(used)} / ${formatWalletQuota(amount)}`
            : t('Unlimited')}
        </span>
      </div>
      <Progress
        value={percent}
        className='subscription-quota-progress mt-2 h-1.5'
        aria-label={`${props.label} ${percent}%`}
      />
      <div className='subscription-quota-meter-meta text-muted-foreground mt-1.5 flex items-center justify-between gap-2'>
        <span className='truncate'>{resetLabel}</span>
        <span className='shrink-0 tabular-nums'>
          {amount > 0 ? formatWalletQuota(remaining) : '—'}
        </span>
      </div>
    </div>
  )
}

function UsageStat(props: {
  icon: typeof Coins
  label: string
  value: string
  tone: 'blue' | 'mint' | 'violet' | 'coral'
}) {
  const Icon = props.icon
  return (
    <article
      className={cn(
        'subscription-usage-stat',
        `subscription-usage-stat-${props.tone}`
      )}
    >
      <span className='subscription-usage-stat-icon' aria-hidden='true'>
        <Icon className='size-4' />
      </span>
      <div className='min-w-0'>
        <p className='text-muted-foreground truncate text-xs'>{props.label}</p>
        <p className='mt-1 truncate font-[family-name:var(--font-geist-mono)] text-xl font-semibold tabular-nums'>
          {props.value}
        </p>
      </div>
    </article>
  )
}

function ModelUsageRow(props: { row: ModelUsageSummary; totalQuota: number }) {
  const { t } = useTranslation()
  const share =
    props.totalQuota > 0 ? (props.row.quota / props.totalQuota) * 100 : 0
  const modelLabel =
    props.row.model === '__unknown__' ? t('Other') : props.row.model

  return (
    <tr>
      <td data-label={t('Model')}>
        <div className='flex min-w-0 items-center gap-2'>
          <span className='subscription-usage-model-dot' aria-hidden='true' />
          <span className='truncate font-medium' title={modelLabel}>
            {modelLabel}
          </span>
        </div>
      </td>
      <td
        data-label={t('Usage')}
        className='font-[family-name:var(--font-geist-mono)] tabular-nums'
      >
        {formatWalletQuota(props.row.quota)}
      </td>
      <td
        data-label={t('Requests')}
        className='font-[family-name:var(--font-geist-mono)] tabular-nums'
      >
        {formatNumber(props.row.requests)}
      </td>
      <td
        data-label={t('Tokens')}
        className='font-[family-name:var(--font-geist-mono)] tabular-nums'
      >
        {formatNumber(props.row.tokenUsed)}
      </td>
      <td data-label={t('Share')}>
        <div className='subscription-usage-share'>
          <Progress
            value={share}
            aria-label={`${modelLabel} ${Math.round(share)}%`}
          />
          <span className='shrink-0 font-[family-name:var(--font-geist-mono)] text-xs tabular-nums'>
            {Math.round(share)}%
          </span>
        </div>
      </td>
    </tr>
  )
}

// A plan whose entitlement is a counted number of image generations (the
// gpt-image and banana packages) never consumes wallet quota: the relay charges
// the image_count grant per generation instead, and the plan's quota fields only
// carry the placeholder value 1. Rendering those fields would draw three
// meaningless "$0.000002" meters, so an image package is recognised by
// "counted image grants + no real quota entitlement". The same grant signal
// drives the plan card and the purchase dialog.
const QUOTA_PLACEHOLDER_LIMIT = 1

function imageGrantTotals(subscription?: UserSubscription) {
  let amount = 0
  let used = 0
  for (const grant of subscription?.resource_grants || []) {
    if (grant.resource_type !== 'image_count') continue
    const grantAmount = Math.max(0, Number(grant.amount) || 0)
    if (grantAmount === 0) continue
    amount += grantAmount
    used += Math.min(grantAmount, Math.max(0, Number(grant.used) || 0))
  }
  return { amount, used, remaining: Math.max(0, amount - used) }
}

function isImagePackage(subscription?: UserSubscription) {
  const quota = Math.max(0, Number(subscription?.amount_total) || 0)
  return (
    imageGrantTotals(subscription).amount > 0 &&
    quota <= QUOTA_PLACEHOLDER_LIMIT
  )
}

function SubscriptionResourceGrants(props: {
  grants: NonNullable<UserSubscriptionRecord['subscription']>['resource_grants']
}) {
  const { t } = useTranslation()
  const grants = (props.grants || []).filter(
    (grant) => Number(grant.amount) > 0 && grant.model_name?.trim()
  )
  if (grants.length === 0) return null

  return (
    <div className='subscription-usage-resource-grants'>
      <div className='subscription-usage-resource-heading'>
        <Gift className='size-3.5' aria-hidden='true' />
        <span>{t('Bonus resources')}</span>
      </div>
      <div className='subscription-usage-resource-list'>
        {grants.map((grant) => {
          const amount = Math.max(0, Number(grant.amount) || 0)
          const used = Math.min(amount, Math.max(0, Number(grant.used) || 0))
          const percent = amount > 0 ? Math.round((used / amount) * 100) : 0
          const label = grant.display_name?.trim() || grant.model_name
          const value =
            grant.resource_type === 'image_count'
              ? `${used} / ${amount} ${t('generations')}`
              : `${formatWalletQuota(used)} / ${formatWalletQuota(amount)}`

          return (
            <div
              key={`${grant.resource_key}-${grant.resource_type}`}
              className='subscription-usage-resource-item'
            >
              <div className='flex items-center justify-between gap-3 text-xs'>
                <span className='truncate font-medium' title={label}>
                  {label}
                </span>
                <span className='shrink-0 font-[family-name:var(--font-geist-mono)] tabular-nums'>
                  {value}
                </span>
              </div>
              <Progress
                value={percent}
                className='subscription-quota-progress mt-1.5 h-1'
                aria-label={`${label} ${percent}%`}
              />
            </div>
          )
        })}
      </div>
    </div>
  )
}

export function SubscriptionUsagePage() {
  const { t } = useTranslation()
  const [subscriptions, setSubscriptions] = useState<UserSubscriptionRecord[]>(
    []
  )
  const [plans, setPlans] = useState<PlanRecord[]>([])
  const [usageRows, setUsageRows] = useState<
    Awaited<ReturnType<typeof getUserQuotaDates>>['data']
  >([])
  const [billingPreference, setBillingPreference] =
    useState('subscription_first')
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState(false)

  const fetchUsage = useCallback(async () => {
    setError(false)
    const [subscriptionResult, planResult] = await Promise.allSettled([
      getSelfSubscriptionFull(),
      getPublicPlans(),
    ])

    let nextSubscriptions: UserSubscriptionRecord[] = []
    if (
      subscriptionResult.status === 'fulfilled' &&
      subscriptionResult.value.success
    ) {
      const data = subscriptionResult.value.data
      nextSubscriptions = data?.subscriptions || []
      setSubscriptions(nextSubscriptions)
      setBillingPreference(data?.billing_preference || 'subscription_first')
    }
    if (planResult.status === 'fulfilled' && planResult.value.success) {
      setPlans(planResult.value.data || [])
    }

    try {
      const window = getSubscriptionUsageWindow(nextSubscriptions)
      const response = await getUserQuotaDates({
        start_timestamp: window.startTimestamp,
        end_timestamp: window.endTimestamp,
        default_time: 'day',
      })
      if (!response.success) throw new Error('usage request failed')
      setUsageRows(response.data || [])
    } catch {
      setUsageRows([])
      setError(true)
    }
  }, [])

  useEffect(() => {
    void fetchUsage().finally(() => setLoading(false))
  }, [fetchUsage])

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      await fetchUsage()
    } finally {
      setRefreshing(false)
    }
  }

  const handlePreferenceChange = async (preference: string) => {
    const previous = billingPreference
    setBillingPreference(preference)
    try {
      const response = await updateBillingPreference(preference)
      if (!response.success) {
        throw new Error(response.message || t('Update failed'))
      }
      setBillingPreference(response.data?.billing_preference || preference)
      toast.success(t('Updated successfully'))
    } catch (cause) {
      setBillingPreference(previous)
      toast.error(cause instanceof Error ? cause.message : t('Request failed'))
    }
  }

  const activeSubscriptions = useMemo(
    () =>
      subscriptions.filter((item) => item.subscription?.status === 'active'),
    [subscriptions]
  )
  const planTitleMap = useMemo(
    () => new Map(plans.map((record) => [record.plan.id, record.plan.title])),
    [plans]
  )
  const modelUsage = useMemo(
    () => aggregateUsageByModel(usageRows || []),
    [usageRows]
  )
  const totalUsage = useMemo(
    () => modelUsage.reduce((sum, row) => sum + row.quota, 0),
    [modelUsage]
  )
  const totalTokens = useMemo(
    () => modelUsage.reduce((sum, row) => sum + row.tokenUsed, 0),
    [modelUsage]
  )
  const totalRequests = useMemo(
    () => modelUsage.reduce((sum, row) => sum + row.requests, 0),
    [modelUsage]
  )
  const quotaTotals = useMemo(() => {
    let total = 0
    let used = 0
    for (const item of activeSubscriptions) {
      if (isImagePackage(item.subscription)) continue
      total += Math.max(0, Number(item.subscription?.amount_total) || 0)
      used += Math.max(0, Number(item.subscription?.amount_used) || 0)
    }
    return {
      total,
      used,
      remaining: total > 0 ? Math.max(0, total - used) : null,
    }
  }, [activeSubscriptions])
  const imagePackageTotals = useMemo(() => {
    let amount = 0
    let remaining = 0
    for (const item of activeSubscriptions) {
      if (!isImagePackage(item.subscription)) continue
      const totals = imageGrantTotals(item.subscription)
      amount += totals.amount
      remaining += totals.remaining
    }
    return { amount, remaining }
  }, [activeSubscriptions])
  // An image package holds no wallet quota, so its remaining entitlement is the
  // counted generations instead of a quota amount.
  const showsImageRemaining =
    quotaTotals.total === 0 && imagePackageTotals.amount > 0
  const latestEndTime = useMemo(
    () =>
      Math.max(
        0,
        ...activeSubscriptions.map((item) => item.subscription?.end_time || 0)
      ),
    [activeSubscriptions]
  )
  const remainingDays = latestEndTime
    ? Math.max(0, Math.ceil((latestEndTime - Date.now() / 1000) / 86400))
    : 0

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Subscription Usage')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t('Review active plans and usage aggregated by model.')}
      </SectionPageLayout.Description>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          size='sm'
          className='gap-1.5'
          onClick={handleRefresh}
          disabled={refreshing || loading}
        >
          <RefreshCw
            className={cn('size-3.5', refreshing && 'animate-spin')}
            aria-hidden='true'
          />
          {t('Refresh')}
        </Button>
        <Button
          size='sm'
          className='gap-1.5'
          render={<Link to='/subscription-plans' />}
        >
          {t('Subscription Plans')}
          <ArrowRight className='size-3.5' aria-hidden='true' />
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='subscription-usage-page subscription-wide-page'>
          {error && (
            <Alert variant='destructive' className='mb-4'>
              <AlertDescription>{t('Failed to fetch usage')}</AlertDescription>
            </Alert>
          )}

          {loading ? (
            <div className='space-y-4'>
              <Skeleton className='h-48 w-full rounded-2xl' />
              <Skeleton className='h-28 w-full rounded-2xl' />
              <Skeleton className='h-80 w-full rounded-2xl' />
            </div>
          ) : (
            <>
              <section
                className='subscription-usage-current subscription-billing-active-card subscription-current-plan bg-card ring-foreground/10 rounded-2xl p-4 ring-1 sm:p-5 lg:p-6'
                aria-label={t('My Current Plan')}
              >
                <div className='subscription-usage-current-header'>
                  <div className='flex min-w-0 items-start gap-3'>
                    <span
                      className='subscription-usage-heading-icon'
                      aria-hidden='true'
                    >
                      <WalletCards className='size-5' />
                    </span>
                    <div className='min-w-0'>
                      <div className='flex flex-wrap items-center gap-2'>
                        <p className='text-muted-foreground text-xs font-medium tracking-[0.12em] uppercase'>
                          {t('My Current Plan')}
                        </p>
                        {activeSubscriptions.length > 0 && (
                          <span className='subscription-status-badge'>
                            <CheckCircle2
                              className='mr-1 size-3'
                              aria-hidden='true'
                            />
                            {t('Active')}
                          </span>
                        )}
                      </div>
                      <h2 className='mt-1.5 text-lg font-semibold tracking-tight'>
                        {t('Usage')}
                      </h2>
                      <p className='text-muted-foreground mt-1 text-xs'>
                        {t(
                          'Usage is calculated from consume logs in the selected plan groups'
                        )}
                      </p>
                    </div>
                  </div>
                  <div className='subscription-usage-current-window'>
                    <Clock3 className='size-3.5' aria-hidden='true' />
                    <span>{t('Last 30 days')}</span>
                  </div>
                </div>

                {activeSubscriptions.length > 0 ? (
                  <div className='subscription-usage-plan-list'>
                    {activeSubscriptions.map((item, index) => {
                      const subscription = item.subscription
                      const imagePackage = isImagePackage(subscription)
                      const imageTotals = imageGrantTotals(subscription)
                      const dailyAmount =
                        Number(subscription?.daily_amount) || 0
                      const weeklyAmount =
                        Number(subscription?.weekly_amount) || 0
                      const monthlyAmount =
                        Number(subscription?.monthly_amount) || 0
                      const meters = imagePackage
                        ? []
                        : [
                            {
                              label: t('Daily Quota'),
                              amount: dailyAmount,
                              used: Number(subscription?.daily_used) || 0,
                              resetTime: subscription?.daily_reset_time,
                              tone: 'mint' as const,
                            },
                            {
                              label: t('Weekly Quota'),
                              amount: weeklyAmount,
                              used: Number(subscription?.weekly_used) || 0,
                              resetTime: subscription?.weekly_reset_time,
                              tone: 'blue' as const,
                            },
                            {
                              label: t('Monthly Quota'),
                              amount: monthlyAmount,
                              used: Number(subscription?.monthly_used) || 0,
                              resetTime: subscription?.monthly_reset_time,
                              tone: 'coral' as const,
                            },
                          ].filter((meter) => meter.amount > 0)
                      if (!imagePackage && meters.length === 0) {
                        meters.push({
                          label: t('Total Quota'),
                          amount: Number(subscription?.amount_total) || 0,
                          used: Number(subscription?.amount_used) || 0,
                          resetTime: subscription?.next_reset_time,
                          tone: 'blue',
                        })
                      }
                      return (
                        <article
                          key={subscription?.id || index}
                          className='subscription-usage-plan-row'
                        >
                          <div className='subscription-usage-plan-identity'>
                            <span className='subscription-usage-plan-index'>
                              {index + 1}
                            </span>
                            <div className='min-w-0'>
                              <h3 className='truncate text-sm font-semibold'>
                                {planTitleMap.get(subscription?.plan_id) ||
                                  t('Subscription Plans')}
                              </h3>
                              <p className='text-muted-foreground mt-1 flex items-center gap-1 text-xs'>
                                <CalendarClock
                                  className='size-3.5 shrink-0'
                                  aria-hidden='true'
                                />
                                {t('Until')}{' '}
                                {subscription?.end_time
                                  ? new Date(
                                      subscription.end_time * 1000
                                    ).toLocaleString()
                                  : t('Unlimited')}
                              </p>
                            </div>
                          </div>
                          {meters.length > 0 && (
                            <div className='subscription-usage-plan-meters'>
                              {meters.map((meter) => (
                                <SubscriptionQuotaMeter
                                  key={meter.label}
                                  {...meter}
                                />
                              ))}
                            </div>
                          )}
                          <SubscriptionResourceGrants
                            grants={subscription?.resource_grants}
                          />
                          <div className='subscription-usage-plan-total'>
                            <span className='text-muted-foreground text-xs'>
                              {t('Usage')}
                            </span>
                            <strong>
                              {imagePackage
                                ? `${formatNumber(imageTotals.used)} ${t('generations')}`
                                : formatWalletQuota(
                                    Number(subscription?.amount_used) || 0
                                  )}
                            </strong>
                            <span className='text-muted-foreground text-xs'>
                              {t('{{count}} days remaining', {
                                count: remainingDays,
                              })}
                            </span>
                          </div>
                        </article>
                      )
                    })}
                  </div>
                ) : (
                  <div className='subscription-usage-empty-plan'>
                    <Activity className='size-5' aria-hidden='true' />
                    <div>
                      <p className='font-medium'>{t('No Active')}</p>
                      <p className='text-muted-foreground mt-1 text-xs'>
                        {t('Subscribe to a plan for model access')}
                      </p>
                    </div>
                  </div>
                )}

                <div className='subscription-usage-current-footer'>
                  {activeSubscriptions.length > 0 && (
                    <Select
                      items={[
                        'subscription_first',
                        'wallet_first',
                        'subscription_only',
                        'wallet_only',
                      ].map((value) => ({
                        value,
                        label: getPreferenceLabel(value, t),
                      }))}
                      value={billingPreference}
                      onValueChange={(value) =>
                        value !== null && void handlePreferenceChange(value)
                      }
                    >
                      <SelectTrigger
                        className='h-9 w-full text-xs sm:w-48'
                        aria-label={t('Preferences')}
                      >
                        <SelectValue>
                          {getPreferenceLabel(billingPreference, t)}
                        </SelectValue>
                      </SelectTrigger>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          <SelectItem value='subscription_first'>
                            {getPreferenceLabel('subscription_first', t)}
                          </SelectItem>
                          <SelectItem value='wallet_first'>
                            {getPreferenceLabel('wallet_first', t)}
                          </SelectItem>
                          <SelectItem value='subscription_only'>
                            {getPreferenceLabel('subscription_only', t)}
                          </SelectItem>
                          <SelectItem value='wallet_only'>
                            {getPreferenceLabel('wallet_only', t)}
                          </SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                  )}
                  <p className='text-muted-foreground text-xs'>
                    {t(
                      'Usage is calculated from consume logs in the selected plan groups'
                    )}
                  </p>
                </div>
              </section>

              <section
                className='subscription-usage-kpi-grid'
                aria-label={t('Usage')}
              >
                <UsageStat
                  icon={Coins}
                  tone='blue'
                  label={t('Total Usage')}
                  value={formatWalletQuota(totalUsage)}
                />
                <UsageStat
                  icon={Activity}
                  tone='mint'
                  label={t('API Requests')}
                  value={formatNumber(totalRequests)}
                />
                <UsageStat
                  icon={WalletCards}
                  tone='violet'
                  label={t('Tokens')}
                  value={formatNumber(totalTokens)}
                />
                <UsageStat
                  icon={CheckCircle2}
                  tone='coral'
                  label={
                    showsImageRemaining
                      ? t('Remaining generations')
                      : t('Remaining quota')
                  }
                  value={
                    showsImageRemaining
                      ? `${formatNumber(imagePackageTotals.remaining)} ${t('generations')}`
                      : quotaTotals.remaining === null
                        ? t('Unlimited')
                        : formatWalletQuota(quotaTotals.remaining)
                  }
                />
              </section>

              <section
                className='subscription-usage-models bg-card ring-foreground/10 rounded-2xl p-4 ring-1 sm:p-5 lg:p-6'
                aria-labelledby='subscription-usage-models-heading'
              >
                <div className='subscription-usage-section-header'>
                  <div>
                    <p className='subscription-section-kicker'>{t('Usage')}</p>
                    <h2
                      id='subscription-usage-models-heading'
                      className='mt-1 text-lg font-semibold tracking-tight'
                    >
                      {t('Usage by model')}
                    </h2>
                  </div>
                  <span className='text-muted-foreground text-xs'>
                    {t('{{count}} models', { count: modelUsage.length })}
                  </span>
                </div>
                {modelUsage.length > 0 ? (
                  <div className='subscription-usage-table-wrap'>
                    <table className='subscription-usage-table'>
                      <thead>
                        <tr>
                          <th>{t('Model')}</th>
                          <th>{t('Usage')}</th>
                          <th>{t('Requests')}</th>
                          <th>{t('Tokens')}</th>
                          <th>{t('Share')}</th>
                        </tr>
                      </thead>
                      <tbody>
                        {modelUsage.map((row) => (
                          <ModelUsageRow
                            key={row.model}
                            row={row}
                            totalQuota={totalUsage}
                          />
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <div className='subscription-usage-empty-models'>
                    <Activity className='size-5' aria-hidden='true' />
                    <p>{t('No model usage yet')}</p>
                  </div>
                )}
              </section>
            </>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
