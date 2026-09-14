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
import { Check, Gift, Sparkles } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  getPublicPlans,
  getSelfSubscriptionFull,
} from '@/features/subscriptions/api'
import { SubscriptionPurchaseDialog } from '@/features/subscriptions/components/dialogs/subscription-purchase-dialog'
import {
  formatSubscriptionValidity,
  formatSubscriptionPrice,
} from '@/features/subscriptions/lib'
import {
  getIncludedModels,
  getPlanModelFamily,
  groupPlansByModelFamily,
  type ModelFamilyKey,
  type ModelFamilyPlanGroup,
} from '@/features/subscriptions/lib/model-family'
import type {
  PlanRecord,
  SubscriptionPlan,
  UserSubscriptionRecord,
} from '@/features/subscriptions/types'
import { cn } from '@/lib/utils'

import { formatWalletQuota } from '../lib'
import type { PaymentMethod, TopupInfo } from '../types'

interface SubscriptionPlansCardProps {
  topupInfo: TopupInfo | null
  onAvailabilityChange?: (available: boolean) => void
  userQuota?: number
  onPurchaseSuccess?: () => void | Promise<void>
}

function ModelFamilyHeader({
  group,
  headingId,
}: {
  group: ModelFamilyPlanGroup
  headingId: string
}) {
  const { t } = useTranslation()
  const modelCount = new Set(
    group.plans.flatMap((plan) => getIncludedModels(plan, group.family))
  ).size

  return (
    <div className='subscription-model-family-header'>
      <div className='subscription-package-family-kicker'>
        <span
          className={cn(
            'subscription-model-family-mark',
            `subscription-model-family-mark-${group.family.tone}`
          )}
          aria-hidden='true'
        />
        <h3
          id={headingId}
          className='text-base font-semibold tracking-tight sm:text-lg'
        >
          {t(group.family.labelKey)}
        </h3>
        <span className='subscription-model-family-count'>
          {modelCount} {t('models')}
        </span>
      </div>
      <p className='subscription-package-family-description text-muted-foreground text-xs leading-5 sm:text-sm'>
        {t(group.family.descriptionKey)}
      </p>
      <span className='subscription-model-family-tier-count'>
        {group.plans.length} {t('tiers')}
      </span>
    </div>
  )
}

interface SubscriptionPlanCardProps {
  record: PlanRecord
  family: ModelFamilyPlanGroup['family']
  isPopular: boolean
  purchaseCount: number
  rechargeEnabled: boolean
  onSelect: (record: PlanRecord) => void
}

function SubscriptionPlanCard(props: SubscriptionPlanCardProps) {
  const { t } = useTranslation()
  const plan = props.record.plan
  const price = formatSubscriptionPrice(plan)
  const limit = Number(plan.max_purchase_per_user || 0)
  const reached = limit > 0 && props.purchaseCount >= limit
  const includedModels = getIncludedModels(plan, props.family)
  const bonusResources = (plan.bonus_resources || []).filter(
    (resource) => Number(resource.amount) > 0 && resource.model_name?.trim()
  )
  const imageAllowance = bonusResources
    .filter((resource) => resource.resource_type === 'image_count')
    .reduce((total, resource) => total + Number(resource.amount), 0)
  const displayBonusResources = bonusResources.filter(
    (resource) => resource.resource_type !== 'image_count'
  )
  const badgeText =
    plan.badge_text?.trim() || (props.isPopular ? t('Hot recommendation') : '')
  const totalQuota = Number(plan.total_amount || 0)
  const isImagePlan = imageAllowance > 0
  const validity = formatSubscriptionValidity(plan, t)
  let summaryCredit = t('Unlimited')
  if (isImagePlan) {
    summaryCredit = `${imageAllowance.toLocaleString()} ${t('generations')}`
  } else if (totalQuota > 0) {
    summaryCredit = formatWalletQuota(totalQuota)
  }
  const stockRemaining =
    limit > 0 ? Math.max(0, limit - props.purchaseCount) : 0
  const stockPercent =
    limit > 0 ? Math.min(100, Math.round((stockRemaining / limit) * 100)) : 100
  const benefits = getPlanBenefits(
    plan,
    props.family,
    t,
    price,
    totalQuota,
    includedModels.length
  )

  return (
    <article
      className={cn(
        'subscription-billing-plan-card relative flex min-w-0 flex-col bg-card p-4 transition-all duration-200',
        `subscription-billing-plan-card-tone-${props.family.tone}`,
        (badgeText || limit > 0) && 'subscription-billing-plan-card-with-badge',
        props.isPopular && 'subscription-billing-plan-featured'
      )}
      aria-label={plan.title || t('Subscription Plans')}
    >
      {(badgeText || limit > 0) && (
        <div className='subscription-plan-badges'>
          {badgeText && (
            <span
              className={cn(
                'subscription-billing-plan-badge',
                props.isPopular && 'subscription-billing-plan-badge-featured',
                !props.isPopular && 'subscription-billing-plan-badge-custom'
              )}
            >
              {props.isPopular && (
                <Sparkles className='size-3' aria-hidden='true' />
              )}
              {badgeText}
            </span>
          )}
          {limit > 0 && (
            <span className='subscription-billing-plan-badge subscription-billing-plan-badge-stock'>
              {t('Only {{count}} left', { count: stockRemaining })}
            </span>
          )}
        </div>
      )}

      <div className='subscription-plan-card-header'>
        <h4 className='text-base font-semibold tracking-tight'>
          {plan.title || t('Subscription Plans')}
        </h4>
        <p className='text-muted-foreground mt-1.5 line-clamp-2 min-h-9 text-xs leading-5'>
          {plan.subtitle || t('Subscribe to a plan for model access')}
        </p>
      </div>

      <div className='subscription-plan-price-row mt-4 flex items-baseline gap-1'>
        <span className='font-[family-name:var(--font-geist-mono)] text-3xl font-bold tracking-tight tabular-nums'>
          {price}
        </span>
        <span className='text-muted-foreground text-sm'>/ {validity}</span>
      </div>

      <div className='subscription-plan-summary-grid mt-4'>
        <div className='subscription-plan-summary-block subscription-plan-summary-block-credit'>
          <span>
            {isImagePlan ? t('Image allowance') : t('Estimated wallet credit')}
          </span>
          <strong>{summaryCredit}</strong>
        </div>
        <div className='subscription-plan-summary-block subscription-plan-summary-block-multiplier'>
          <span>{t('Validity Period')}</span>
          <strong>{validity}</strong>
        </div>
      </div>

      <div className='subscription-plan-access-strip mt-4'>
        <Gift
          className='subscription-plan-access-icon size-4 shrink-0'
          aria-hidden='true'
        />
        <div className='min-w-0'>
          <p className='truncate text-xs font-semibold'>
            {t('Package benefits')}
          </p>
          <p className='text-muted-foreground mt-0.5 text-[11px]'>
            {t('Includes {{count}} models from {{family}}', {
              count: includedModels.length,
              family: t(props.family.labelKey),
            })}
          </p>
        </div>
      </div>

      {displayBonusResources.length > 0 && (
        <div className='subscription-plan-bonus-resources mt-3'>
          <div className='flex items-center gap-2 text-xs font-semibold'>
            <Gift
              className='subscription-plan-access-icon size-3.5 shrink-0'
              aria-hidden='true'
            />
            {t('Included extras')}
          </div>
          <div className='mt-2 flex flex-wrap gap-1.5'>
            {displayBonusResources.map((resource) => (
              <span
                key={`${resource.resource_key}-${resource.resource_type}`}
                className='subscription-plan-bonus-resource'
              >
                {resource.display_name?.trim() || resource.model_name}
                <strong>{formatWalletQuota(Number(resource.amount))}</strong>
              </span>
            ))}
          </div>
        </div>
      )}

      <ul className='subscription-plan-benefits mt-4 space-y-2'>
        {benefits.map((benefit) => (
          <li
            key={benefit}
            className='text-foreground/90 flex items-start gap-2 text-sm'
          >
            <Check
              className='subscription-plan-check mt-0.5 size-4 shrink-0'
              aria-hidden='true'
            />
            <span className='leading-snug'>{benefit}</span>
          </li>
        ))}
      </ul>

      <div className='subscription-plan-stock mt-4'>
        <div className='flex items-center justify-between gap-3 text-[11px]'>
          <span className='text-muted-foreground'>{t('Available')}</span>
          <span className='font-medium tabular-nums'>
            {limit > 0 ? `${stockRemaining}/${limit}` : t('Unlimited')}
          </span>
        </div>
        <Progress
          value={stockPercent}
          className='subscription-plan-stock-progress mt-2 h-1.5'
          aria-label={`${t('Available')} ${stockPercent}%`}
        />
      </div>

      <div className='subscription-plan-card-footer mt-auto pt-4'>
        {reached ? (
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant='outline'
                  size='sm'
                  className='h-10 w-full text-xs'
                  disabled
                />
              }
            >
              {t('Limit Reached')}
            </TooltipTrigger>
            <TooltipContent>
              {t('Purchase limit reached')} ({props.purchaseCount}/{limit})
            </TooltipContent>
          </Tooltip>
        ) : (
          <Button
            variant='default'
            size='sm'
            className='subscription-plan-cta h-10 w-full text-xs'
            onClick={() => {
              if (props.rechargeEnabled) props.onSelect(props.record)
            }}
            disabled={!props.rechargeEnabled}
          >
            {props.rechargeEnabled
              ? t('Subscribe Now')
              : t('Recharge permission required')}
          </Button>
        )}
      </div>
    </article>
  )
}

function getPlanBenefits(
  plan: SubscriptionPlan,
  family: ModelFamilyPlanGroup['family'],
  t: TFunction,
  price: string,
  totalQuota: number,
  includedModelCount: number
): string[] {
  const configuredBenefits = (plan.benefits || [])
    .map((benefit) => benefit.trim())
    .filter(Boolean)
  if (configuredBenefits.length > 0) return configuredBenefits

  const familyLabel = t(family.labelKey)
  const credit = totalQuota > 0 ? formatWalletQuota(totalQuota) : t('Unlimited')
  return [
    t('Supports the {{family}} model family', { family: familyLabel }),
    t('Pay {{price}}, receive {{credit}} in {{family}} quota', {
      price,
      credit,
      family: familyLabel,
    }),
    t('{{family}} usage is billed by configured ratios', {
      family: familyLabel,
    }),
    t(family.descriptionKey),
    t('Includes {{count}} models from {{family}}', {
      count: includedModelCount,
      family: familyLabel,
    }),
    t('Valid for {{duration}}', {
      duration: formatSubscriptionValidity(plan, t),
    }),
  ]
}

function getEpayMethods(payMethods: PaymentMethod[] = []): PaymentMethod[] {
  return payMethods.filter(
    (method) =>
      method?.type && method.type !== 'stripe' && method.type !== 'creem'
  )
}

export function SubscriptionPlansCard({
  topupInfo,
  onAvailabilityChange,
  userQuota,
  onPurchaseSuccess,
}: SubscriptionPlansCardProps) {
  const { t } = useTranslation()

  const [plans, setPlans] = useState<PlanRecord[]>([])
  const [allSubscriptions, setAllSubscriptions] = useState<
    UserSubscriptionRecord[]
  >([])
  const [loading, setLoading] = useState(true)
  const [purchaseOpen, setPurchaseOpen] = useState(false)
  const [selectedPlan, setSelectedPlan] = useState<PlanRecord | null>(null)
  const [selectedFamilyKey, setSelectedFamilyKey] = useState<
    ModelFamilyKey | undefined
  >()

  const enableStripe = !!topupInfo?.enable_stripe_topup
  const enableCreem = !!topupInfo?.enable_creem_topup
  const enableWaffoPancake = !!topupInfo?.enable_waffo_pancake_topup
  const enableOnlineTopUp = !!topupInfo?.enable_online_topup
  const rechargeEnabled = topupInfo?.top_up_enabled !== false
  const epayMethods = useMemo(
    () => getEpayMethods(topupInfo?.pay_methods),
    [topupInfo?.pay_methods]
  )

  const fetchPlans = useCallback(async () => {
    try {
      const response = await getPublicPlans()
      setPlans(response.success ? response.data || [] : [])
    } catch {
      setPlans([])
    }
  }, [])

  const fetchSelfSubscription = useCallback(async () => {
    try {
      const response = await getSelfSubscriptionFull()
      if (response.success && response.data) {
        setAllSubscriptions(response.data.all_subscriptions || [])
      }
    } catch {
      // The page remains usable when subscription data is temporarily unavailable.
    }
  }, [])

  useEffect(() => {
    const initialize = async () => {
      setLoading(true)
      await Promise.all([fetchPlans(), fetchSelfSubscription()])
      setLoading(false)
    }
    void initialize()
  }, [fetchPlans, fetchSelfSubscription])

  const displayPlans = useMemo(
    () =>
      plans
        .filter((record) => record?.plan?.enabled !== false)
        .sort((left, right) => {
          const priceDelta =
            Number(left.plan.price_amount) - Number(right.plan.price_amount)
          if (priceDelta !== 0) return priceDelta
          const orderDelta = left.plan.sort_order - right.plan.sort_order
          return orderDelta !== 0 ? orderDelta : left.plan.id - right.plan.id
        }),
    [plans]
  )

  // The administrator controls every sellable plan. Keep all configured
  // durations visible together so a plan is never hidden behind a cycle toggle.
  const visiblePlans = useMemo(
    () =>
      displayPlans.filter((record) => Boolean(getPlanModelFamily(record.plan))),
    [displayPlans]
  )

  const modelFamilyGroups = useMemo(
    () => groupPlansByModelFamily(visiblePlans.map((record) => record.plan)),
    [visiblePlans]
  )

  const selectedFamilyGroup =
    modelFamilyGroups.find((group) => group.family.key === selectedFamilyKey) ||
    modelFamilyGroups[0]

  const planRecordMap = useMemo(
    () => new Map(visiblePlans.map((record) => [record.plan.id, record])),
    [visiblePlans]
  )

  const planPurchaseCountMap = useMemo(() => {
    const map = new Map<number, number>()
    for (const subscription of allSubscriptions) {
      const planId = subscription.subscription?.plan_id
      if (planId) map.set(planId, (map.get(planId) || 0) + 1)
    }
    return map
  }, [allSubscriptions])

  useEffect(() => {
    onAvailabilityChange?.(true)
  }, [onAvailabilityChange])

  if (loading) {
    return (
      <div className='flex flex-col gap-6'>
        <div className='grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3'>
          {['first', 'second', 'third'].map((key) => (
            <Skeleton key={key} className='h-72 w-full rounded-2xl' />
          ))}
        </div>
      </div>
    )
  }

  return (
    <>
      <div className='subscription-billing-card flex flex-col gap-8'>
        {visiblePlans.length > 0 && (
          <section
            id='subscription-plan-options'
            className='subscription-plan-options scroll-mt-6'
            aria-label={t('Subscription Plans')}
          >
            <div className='subscription-plans-heading mb-5'>
              <div>
                <p className='subscription-section-kicker'>
                  {t('Model packages')}
                </p>
                <h2 className='mt-2 text-xl font-semibold tracking-tight sm:text-2xl'>
                  {t('Choose a model package')}
                </h2>
                <p className='text-muted-foreground mt-2 text-sm'>
                  {t(
                    'Select a model family, then choose the tier that fits your workload.'
                  )}
                </p>
              </div>
            </div>

            {!rechargeEnabled && (
              <Alert variant='destructive' className='mb-5'>
                <AlertDescription>
                  {t(
                    'Subscription is locked for this account. Please join the official group and contact an administrator to enable subscription access.'
                  )}
                </AlertDescription>
              </Alert>
            )}

            <div className='subscription-package-toolbar'>
              <div
                className='subscription-model-family-tabs'
                role='tablist'
                aria-label={t('Model packages')}
              >
                {modelFamilyGroups.map((group) => {
                  const isSelected =
                    selectedFamilyGroup?.family.key === group.family.key
                  return (
                    <button
                      key={group.family.key}
                      id={`subscription-family-tab-${group.family.key}`}
                      type='button'
                      role='tab'
                      aria-selected={isSelected}
                      aria-controls={
                        isSelected
                          ? `subscription-family-panel-${group.family.key}`
                          : undefined
                      }
                      className={cn(
                        'subscription-model-family-tab',
                        isSelected && 'subscription-model-family-tab-active'
                      )}
                      onClick={() => setSelectedFamilyKey(group.family.key)}
                    >
                      <span
                        className={cn(
                          'subscription-model-family-tab-dot',
                          `subscription-model-family-tab-dot-${group.family.tone}`
                        )}
                        aria-hidden='true'
                      />
                      <span>{t(group.family.labelKey)}</span>
                      <span className='subscription-model-family-tab-count'>
                        {group.plans.length}
                      </span>
                    </button>
                  )
                })}
              </div>
            </div>

            {selectedFamilyGroup && (
              <div
                id={`subscription-family-panel-${selectedFamilyGroup.family.key}`}
                role='tabpanel'
                aria-labelledby={`subscription-family-heading-${selectedFamilyGroup.family.key}`}
                className='subscription-model-family-panel'
                tabIndex={0}
              >
                <ModelFamilyHeader
                  group={selectedFamilyGroup}
                  headingId={`subscription-family-heading-${selectedFamilyGroup.family.key}`}
                />
                <div className='subscription-billing-plan-grid'>
                  {selectedFamilyGroup.plans.map((plan) => {
                    const familyPlans = selectedFamilyGroup.plans
                    const familyRecommended =
                      familyPlans.find((candidate) => candidate.is_recommended)
                        ?.id ??
                      familyPlans.find(
                        (candidate) =>
                          candidate.title.trim().toLowerCase() === 'standard'
                      )?.id ??
                      familyPlans[Math.floor((familyPlans.length - 1) / 2)]?.id
                    const record = planRecordMap.get(plan.id)
                    if (!record) return null
                    return (
                      <SubscriptionPlanCard
                        key={plan.id}
                        record={record}
                        family={selectedFamilyGroup.family}
                        isPopular={plan.id === familyRecommended}
                        purchaseCount={planPurchaseCountMap.get(plan.id) || 0}
                        rechargeEnabled={rechargeEnabled}
                        onSelect={(selected) => {
                          setSelectedPlan(selected)
                          setPurchaseOpen(true)
                        }}
                      />
                    )
                  })}
                </div>
              </div>
            )}
          </section>
        )}

        {visiblePlans.length === 0 &&
          (!rechargeEnabled ? (
            <Alert variant='destructive'>
              <AlertDescription>
                {t(
                  'Subscription is locked for this account. Please join the official group and contact an administrator to enable subscription access.'
                )}
              </AlertDescription>
            </Alert>
          ) : (
            <p className='text-muted-foreground py-4 text-center text-sm'>
              {t('No plans available')}
            </p>
          ))}
      </div>

      <SubscriptionPurchaseDialog
        open={purchaseOpen}
        onOpenChange={(open) => {
          setPurchaseOpen(open)
          if (!open) void fetchSelfSubscription()
        }}
        plan={selectedPlan}
        enableStripe={enableStripe}
        enableCreem={enableCreem}
        enableWaffoPancake={enableWaffoPancake}
        enableOnlineTopUp={enableOnlineTopUp}
        topUpEnabled={rechargeEnabled}
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
