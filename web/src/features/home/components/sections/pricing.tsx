/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowRight, Check, Zap } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { getPublicPlans } from '@/features/subscriptions/api'
import {
  formatDuration,
  formatSubscriptionPrice,
} from '@/features/subscriptions/lib'
import type { PlanRecord } from '@/features/subscriptions/types'
import { formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

interface PricingProps {
  isAuthenticated?: boolean
}

const PLAN_DESCRIPTION_KEYS: Record<string, string> = {
  starter: 'For daily development and light usage',
  basic: 'For daily development and light usage',
  professional: 'An advanced choice, twice the Basic plan',
  plus: 'An advanced choice, twice the Basic plan',
  standard: 'Most popular for everyday individual development',
  team: 'Most popular for everyday individual development',
  studio: 'Professional development and high-concurrency production',
  pro: 'Professional development and high-concurrency production',
  business: 'Professional development and high-concurrency production',
  ultra: 'For high-demand users, $1 = 12x quota',
}

function sortPlans(records: PlanRecord[]): PlanRecord[] {
  return records
    .filter((record) => record.plan.enabled !== false)
    .slice()
    .sort((left, right) => {
      const priceDelta =
        Number(left.plan.price_amount) - Number(right.plan.price_amount)
      if (priceDelta !== 0) return priceDelta
      const orderDelta = left.plan.sort_order - right.plan.sort_order
      return orderDelta !== 0 ? orderDelta : left.plan.id - right.plan.id
    })
}

function getPlanDescriptionKey(plan: PlanRecord): string {
  const normalizedTitle = plan.plan.title.trim().toLowerCase()
  return (
    PLAN_DESCRIPTION_KEYS[normalizedTitle] ||
    'Subscribe to a plan for model access'
  )
}

function getRecommendedIndex(plans: PlanRecord[]): number {
  const standardIndex = plans.findIndex(
    (record) => record.plan.title.trim().toLowerCase() === 'standard'
  )
  if (standardIndex >= 0) return standardIndex
  return plans.length > 0 ? Math.floor((plans.length - 1) / 2) : -1
}

export function PricingPreview(props: PricingProps) {
  const { t } = useTranslation()
  const planHref = props.isAuthenticated ? '/subscription-plans' : '/sign-up'
  const plansQuery = useQuery({
    queryKey: ['public-subscription-plans'],
    queryFn: getPublicPlans,
    retry: false,
    staleTime: 60 * 1000,
  })
  const isLoading = plansQuery.isLoading
  const plans = useMemo(
    () =>
      plansQuery.data?.success && plansQuery.data.data
        ? sortPlans(plansQuery.data.data)
        : [],
    [plansQuery.data]
  )
  const recommendedIndex = useMemo(() => getRecommendedIndex(plans), [plans])
  const recommendedPlan = recommendedIndex >= 0 ? plans[recommendedIndex] : null
  const highlightedPlan = plans.at(-1) ?? null
  const hasPlanError =
    plansQuery.isError || Boolean(plansQuery.data && !plansQuery.data.success)

  return (
    <section id='pricing' className='home-reference-section px-4 pb-24'>
      <div className='mx-auto max-w-6xl'>
        <div className='text-center'>
          <p className='home-reference-kicker'>{t('Pricing')}</p>
          <h2 className='home-reference-heading mt-3'>
            {t('Monthly subscriptions · ready to use')}
          </h2>
          <p className='text-muted-foreground mx-auto mt-4 max-w-2xl text-sm leading-6'>
            {recommendedPlan ? (
              <>
                <span className='text-foreground font-semibold'>
                  {formatSubscriptionPrice(recommendedPlan.plan)}{' '}
                  {recommendedPlan.plan.title}
                </span>{' '}
                {t(
                  'is the most popular choice for developers. You can also start with pay-as-you-go billing from $5.'
                )}
              </>
            ) : (
              t('Subscribe to a plan for model access')
            )}
          </p>
        </div>

        {highlightedPlan && (
          <AnimateInView className='home-reference-ultra-banner mt-10'>
            <span className='home-reference-ultra-icon'>
              <Zap aria-hidden='true' />
            </span>
            <div className='min-w-0 flex-1'>
              <div className='home-reference-banner-label'>{t('New')}</div>
              <h3 className='text-sm font-semibold'>
                {highlightedPlan.plan.title} ·{' '}
                {formatSubscriptionPrice(highlightedPlan.plan)}
              </h3>
              <p className='text-muted-foreground mt-1 text-xs leading-5'>
                {highlightedPlan.plan.subtitle ||
                  t(getPlanDescriptionKey(highlightedPlan))}
                <span className='home-reference-banner-limits'>
                  {t('Daily limit {{value}}', {
                    value:
                      highlightedPlan.plan.daily_amount > 0
                        ? formatQuota(highlightedPlan.plan.daily_amount)
                        : t('Unlimited'),
                  })}{' '}
                  ·{' '}
                  {t('Monthly limit {{value}}', {
                    value:
                      highlightedPlan.plan.monthly_amount > 0
                        ? formatQuota(highlightedPlan.plan.monthly_amount)
                        : t('Unlimited'),
                  })}
                </span>
              </p>
            </div>
            <Button
              className='home-reference-banner-button'
              render={<Link to={planHref} />}
              nativeButton={false}
            >
              {t('Subscribe Now')}
              <ArrowRight data-icon='inline-end' aria-hidden='true' />
            </Button>
          </AnimateInView>
        )}

        {isLoading && (
          <div
            className='home-reference-pricing-grid home-reference-pricing-grid-count-5 mt-8'
            role='status'
            aria-label={t('Loading...')}
          >
            {Array.from({ length: 5 }, (_, index) => (
              <div key={index} className='home-reference-plan-card'>
                <Skeleton className='h-5 w-1/2' />
                <Skeleton className='h-10 w-2/3' />
                <div className='flex flex-col gap-3'>
                  <Skeleton className='h-4 w-full' />
                  <Skeleton className='h-4 w-5/6' />
                  <Skeleton className='h-4 w-4/5' />
                </div>
                <Skeleton className='mt-auto h-9 w-full' />
              </div>
            ))}
          </div>
        )}

        {!isLoading && plans.length > 0 && (
          <div
            className={cn(
              'home-reference-pricing-grid mt-8',
              `home-reference-pricing-grid-count-${Math.min(plans.length, 6)}`
            )}
          >
            {plans.map((record, index) => {
              const plan = record.plan
              const isRecommended = index === recommendedIndex
              const isUltra = plans.length > 1 && index === plans.length - 1
              const quotaItems = [
                ['Daily limit {{value}}', plan.daily_amount],
                ['Weekly limit {{value}}', plan.weekly_amount],
                ['Monthly limit {{value}}', plan.monthly_amount],
              ] as const

              return (
                <AnimateInView
                  key={plan.id}
                  delay={index * 60}
                  className={cn(
                    'home-reference-plan-card',
                    isRecommended && 'home-reference-plan-featured',
                    isUltra && 'home-reference-plan-ultra'
                  )}
                  data-plan-id={plan.id}
                >
                  {isRecommended && (
                    <span className='home-reference-plan-badge'>
                      {t('Recommended')}
                    </span>
                  )}
                  {isUltra && (
                    <span className='home-reference-plan-badge home-reference-plan-badge-ultra'>
                      {t('New')}
                    </span>
                  )}
                  <div>
                    <h3 className='text-base font-semibold'>{plan.title}</h3>
                    <p className='text-muted-foreground mt-1.5 min-h-10 text-xs leading-5'>
                      {plan.subtitle || t(getPlanDescriptionKey(record))}
                    </p>
                  </div>
                  <div className='flex items-baseline gap-1'>
                    <span className='font-mono text-3xl font-bold tabular-nums'>
                      {formatSubscriptionPrice(plan)}
                    </span>
                    <span className='text-muted-foreground text-sm'>
                      /{formatDuration(plan, t)}
                    </span>
                  </div>
                  <ul className='flex flex-1 flex-col gap-2.5'>
                    {quotaItems.map(([label, amount]) => (
                      <li
                        key={label}
                        className='flex items-start gap-2 text-sm'
                      >
                        <Check
                          className='home-reference-check mt-0.5 size-4 shrink-0'
                          aria-hidden='true'
                        />
                        <span>
                          {t(label, {
                            value:
                              amount > 0 ? formatQuota(amount) : t('Unlimited'),
                          })}
                        </span>
                      </li>
                    ))}
                    <li className='flex items-start gap-2 text-sm'>
                      <Check
                        className='home-reference-check mt-0.5 size-4 shrink-0'
                        aria-hidden='true'
                      />
                      <span>
                        {t('Compatible with Claude Code and OpenAI SDK')}
                      </span>
                    </li>
                    <li className='flex items-start gap-2 text-sm'>
                      <Check
                        className='home-reference-check mt-0.5 size-4 shrink-0'
                        aria-hidden='true'
                      />
                      <span>{t('Claude and GPT model families')}</span>
                    </li>
                  </ul>
                  <Button
                    variant={isRecommended || isUltra ? 'default' : 'outline'}
                    className='home-reference-plan-button'
                    render={<Link to={planHref} />}
                    nativeButton={false}
                  >
                    {isUltra && plan.title.trim().toLowerCase() === 'ultra'
                      ? t('Subscribe to Ultra')
                      : t('Subscribe Now')}
                  </Button>
                </AnimateInView>
              )
            })}
          </div>
        )}

        {!isLoading && hasPlanError && (
          <div className='home-reference-pricing-error mt-8' role='alert'>
            <span>{t('Failed to load')}</span>
            <Button
              variant='outline'
              size='sm'
              onClick={() => plansQuery.refetch()}
            >
              {t('Retry')}
            </Button>
          </div>
        )}

        {!isLoading && !hasPlanError && plans.length === 0 && (
          <p className='text-muted-foreground mt-8 text-center text-sm'>
            {t('No plans available')}
          </p>
        )}

        {plans.length > 0 && (
          <p className='text-muted-foreground mt-6 text-center text-xs'>
            {t(
              'When a subscription limit is reached, balance billing continues automatically by token.'
            )}
          </p>
        )}
      </div>
    </section>
  )
}
