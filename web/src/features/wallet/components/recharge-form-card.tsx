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
  CircleDollarSign,
  Check,
  CheckCircle2,
  Gift,
  ExternalLink,
  Loader2,
  ShieldCheck,
  Trophy,
  WalletCards,
  Zap,
} from 'lucide-react'
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatSystemCurrencyUSD } from '@/lib/currency'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

import { PAYMENT_TYPES } from '../constants'
import {
  getDiscountLabel,
  getBonusLabel,
  formatCampaignDate,
  getPaymentIcon,
  getMinTopupAmount,
  calculatePresetPricing,
  resolveAmountDiscount,
  formatLocalPaymentAmount,
} from '../lib'
import type {
  PaymentMethod,
  PresetAmount,
  TopupInfo,
  CreemProduct,
  WaffoPayMethod,
} from '../types'
import { CreemProductsSection } from './creem-products-section'

const presetSkeletonKeys = [
  'preset-skeleton-1',
  'preset-skeleton-2',
  'preset-skeleton-3',
  'preset-skeleton-4',
  'preset-skeleton-5',
  'preset-skeleton-6',
  'preset-skeleton-7',
  'preset-skeleton-8',
]
const paymentSkeletonKeys = [
  'payment-skeleton-1',
  'payment-skeleton-2',
  'payment-skeleton-3',
]
const emptyPaymentMethods: PaymentMethod[] = []

interface RechargeFormCardProps {
  topupInfo: TopupInfo | null
  topUpEnabled?: boolean
  presetAmounts: PresetAmount[]
  selectedPreset: number | null
  onSelectPreset: (preset: PresetAmount) => void
  topupAmount: number
  onTopupAmountChange: (amount: number) => void
  paymentAmount: number
  calculating: boolean
  onPaymentMethodSelect: (method: PaymentMethod) => void
  onPaymentMethodChange?: (method: PaymentMethod) => void
  paymentLoading: string | null
  redemptionCode: string
  onRedemptionCodeChange: (code: string) => void
  onRedeem: () => void
  redeeming: boolean
  topupLink?: string
  loading?: boolean
  priceRatio?: number
  usdExchangeRate?: number
  creemProducts?: CreemProduct[]
  enableCreemTopup?: boolean
  onCreemProductSelect?: (product: CreemProduct) => void
  enableWaffoTopup?: boolean
  waffoPayMethods?: WaffoPayMethod[]
  waffoMinTopup?: number
  onWaffoMethodSelect?: (method: WaffoPayMethod, index: number) => void
  onWaffoMethodChange?: (method: WaffoPayMethod, index: number) => void
  enableWaffoPancakeTopup?: boolean
  showRedemptionCode?: boolean
}

export function RechargeFormCard({
  topupInfo,
  topUpEnabled,
  presetAmounts,
  selectedPreset,
  onSelectPreset,
  topupAmount,
  onTopupAmountChange,
  paymentAmount,
  calculating,
  onPaymentMethodSelect,
  onPaymentMethodChange,
  paymentLoading,
  redemptionCode,
  onRedemptionCodeChange,
  onRedeem,
  redeeming,
  topupLink,
  loading,
  priceRatio = 1,
  usdExchangeRate = 1,
  creemProducts,
  enableCreemTopup,
  onCreemProductSelect,
  enableWaffoTopup,
  waffoPayMethods,
  waffoMinTopup,
  onWaffoMethodSelect,
  onWaffoMethodChange,
  enableWaffoPancakeTopup,
  showRedemptionCode = true,
}: RechargeFormCardProps) {
  const { t } = useTranslation()
  const [localAmount, setLocalAmount] = useState(topupAmount.toString())
  const [selectedPaymentKey, setSelectedPaymentKey] = useState<string | null>(
    null
  )
  const rechargeEnabled = topUpEnabled ?? topupInfo?.top_up_enabled !== false

  useEffect(() => {
    // Empty string must survive, otherwise the field can never be cleared
    setLocalAmount((prev) =>
      prev === '' && topupAmount === 0 ? prev : topupAmount.toString()
    )
  }, [topupAmount])

  const handleAmountChange = (value: string) => {
    setLocalAmount(value)
    const numValue = Number.parseInt(value) || 0
    if (numValue >= 0) {
      onTopupAmountChange(numValue)
    }
  }

  const hasConfigurableTopup =
    topupInfo?.enable_online_topup ||
    topupInfo?.enable_stripe_topup ||
    enableWaffoTopup ||
    enableWaffoPancakeTopup
  const hasAnyTopup = hasConfigurableTopup || enableCreemTopup
  const hasWaffoPaymentMethods =
    Array.isArray(waffoPayMethods) && waffoPayMethods.length > 0
  const minTopup = getMinTopupAmount(topupInfo)
  const redemptionEnabled =
    showRedemptionCode && topupInfo?.enable_redemption !== false
  const rechargeBonus = topupInfo?.recharge_bonus
  const showRechargeBonus =
    rechargeBonus?.active &&
    rechargeBonus.show_on_topup !== false &&
    rechargeBonus.bonus_rate > 0
  const inviteRanking = topupInfo?.invite_ranking
  const showInviteRanking =
    inviteRanking?.active &&
    inviteRanking.show_on_topup !== false &&
    Array.isArray(inviteRanking.items) &&
    inviteRanking.items.length > 0
  const bonusLabel = getBonusLabel(rechargeBonus?.bonus_rate || 0)
  const configuredPaymentMethods = topupInfo?.pay_methods || emptyPaymentMethods
  const standardPaymentMethods = hasWaffoPaymentMethods
    ? configuredPaymentMethods.filter(
        (method) => method.type !== PAYMENT_TYPES.WAFFO
      )
    : configuredPaymentMethods
  const hasStandardPaymentMethods = standardPaymentMethods.length > 0
  const selectedStandardMethod = standardPaymentMethods.find(
    (method) => method.type === selectedPaymentKey
  )
  const selectedWaffoIndex = selectedPaymentKey?.startsWith('waffo-')
    ? Number.parseInt(selectedPaymentKey.slice('waffo-'.length))
    : -1
  const selectedWaffoMethod =
    selectedWaffoIndex >= 0 ? waffoPayMethods?.[selectedWaffoIndex] : undefined
  let selectedMinimum = 0
  if (selectedStandardMethod) {
    selectedMinimum = Math.max(selectedStandardMethod.min_topup || 0, minTopup)
  } else if (selectedWaffoMethod) {
    selectedMinimum = waffoMinTopup || 0
  }
  const paymentUnavailable =
    (!selectedStandardMethod && !selectedWaffoMethod) ||
    topupAmount < selectedMinimum

  useEffect(() => {
    const standardSelectionExists = standardPaymentMethods.some(
      (method) => method.type === selectedPaymentKey
    )
    const waffoSelectionExists =
      selectedWaffoIndex >= 0 &&
      selectedWaffoIndex < (waffoPayMethods?.length || 0)
    if (standardSelectionExists || waffoSelectionExists) return

    if (standardPaymentMethods[0]) {
      setSelectedPaymentKey(standardPaymentMethods[0].type)
      return
    }

    if (waffoPayMethods?.length) {
      setSelectedPaymentKey('waffo-0')
      return
    }

    setSelectedPaymentKey(null)
  }, [
    selectedPaymentKey,
    selectedWaffoIndex,
    standardPaymentMethods,
    waffoPayMethods,
  ])

  const handlePaymentContinue = () => {
    if (selectedStandardMethod) {
      onPaymentMethodSelect(selectedStandardMethod)
      return
    }

    if (selectedWaffoMethod && selectedWaffoIndex >= 0 && onWaffoMethodSelect) {
      onWaffoMethodSelect(selectedWaffoMethod, selectedWaffoIndex)
    }
  }

  let standardPaymentMethodsContent = null
  if (hasStandardPaymentMethods) {
    standardPaymentMethodsContent = (
      <div className='grid grid-cols-2 gap-1.5 sm:gap-3 lg:grid-cols-3'>
        {standardPaymentMethods.map((method) => {
          const methodMinTopup = Math.max(method.min_topup || 0, minTopup)
          const disabled = methodMinTopup > topupAmount
          let disabledReason: string | undefined
          let disabledLabel: string | undefined
          if (disabled) {
            disabledReason = t('Minimum topup amount: {{amount}}', {
              amount: formatNumber(methodMinTopup),
            })
            disabledLabel = `${t('Minimum:')} ${formatNumber(methodMinTopup)}`
          }
          const paymentIcon =
            paymentLoading === method.type ? (
              <Loader2 className='h-4 w-4 animate-spin' />
            ) : (
              getPaymentIcon(method.type, 'h-4 w-4', method.icon, method.name)
            )

          const button = (
            <Button
              key={method.type}
              variant='outline'
              onClick={() => {
                setSelectedPaymentKey(method.type)
                onPaymentMethodChange?.(method)
              }}
              disabled={disabled || !!paymentLoading}
              aria-pressed={selectedPaymentKey === method.type}
              title={disabledReason}
              aria-label={
                disabledReason
                  ? `${method.name}. ${disabledReason}`
                  : method.name
              }
              className={cn(
                'min-h-16 min-w-0 justify-start gap-3 rounded-lg px-3 py-2.5 text-left',
                selectedPaymentKey === method.type &&
                  'border-primary bg-primary/5 text-foreground ring-primary/15 ring-2'
              )}
            >
              {paymentIcon}
              <span className='flex min-w-0 flex-1 flex-col items-start gap-0.5'>
                <span className='max-w-full truncate'>{method.name}</span>
                {disabledLabel && (
                  <span className='text-muted-foreground max-w-full truncate text-[11px] leading-4 font-normal'>
                    {disabledLabel}
                  </span>
                )}
              </span>
              {selectedPaymentKey === method.type && (
                <Check className='text-primary size-4' aria-hidden='true' />
              )}
            </Button>
          )

          return disabled ? (
            <TooltipProvider key={method.type}>
              <Tooltip>
                <TooltipTrigger render={button} />
                <TooltipContent>{disabledReason}</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          ) : (
            button
          )
        })}
      </div>
    )
  } else if (!hasWaffoPaymentMethods) {
    standardPaymentMethodsContent = (
      <Alert>
        <AlertDescription>
          {t('No payment methods available. Please contact administrator.')}
        </AlertDescription>
      </Alert>
    )
  }

  if (loading) {
    return (
      <Card
        data-card-hover='false'
        className='wallet-recharge-card gap-0 overflow-hidden py-0'
      >
        <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
          <Skeleton className='h-6 w-32' />
          <Skeleton className='mt-2 h-4 w-48' />
        </CardHeader>
        <CardContent className='space-y-4 p-3 sm:space-y-6 sm:p-5'>
          <div className='space-y-4 sm:space-y-6'>
            {/* Preset Amounts Skeleton */}
            <div className='space-y-3'>
              <Skeleton className='h-3 w-16' />
              <div className='grid grid-cols-2 gap-3 sm:grid-cols-4'>
                {presetSkeletonKeys.map((key) => (
                  <Skeleton key={key} className='h-[72px] rounded-lg' />
                ))}
              </div>
            </div>

            {/* Custom Amount Input Skeleton */}
            <div className='space-y-3'>
              <Skeleton className='h-3 w-28' />
              <Skeleton className='h-[42px] w-full' />
            </div>

            {/* Payment Methods Skeleton */}
            <div className='space-y-3'>
              <Skeleton className='h-3 w-32' />
              <div className='flex flex-wrap gap-3'>
                {paymentSkeletonKeys.map((key) => (
                  <Skeleton key={key} className='h-10 w-24 rounded-lg' />
                ))}
              </div>
            </div>
          </div>

          {/* Redemption Code Section Skeleton */}
          <div className='space-y-3 border-t pt-8'>
            <Skeleton className='h-3 w-24' />
            <div className='flex gap-2'>
              <Skeleton className='h-10 flex-1' />
              <Skeleton className='h-10 w-20' />
            </div>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!rechargeEnabled) {
    return (
      <Card className='wallet-recharge-card gap-0 py-0 shadow-sm'>
        <CardHeader className='border-b p-5 sm:p-6'>
          <div className='flex items-start gap-3'>
            <span className='bg-primary/10 text-primary flex size-9 shrink-0 items-center justify-center rounded-lg'>
              <WalletCards className='size-4' />
            </span>
            <div className='min-w-0'>
              <CardTitle>{t('Online Recharge')}</CardTitle>
              <CardDescription className='mt-1'>
                {t('Choose an amount and payment method')}
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className='p-5 sm:p-6'>
          <Alert variant='destructive'>
            <AlertDescription>
              {t(
                'Recharge is locked for this account. Please join the official group and contact an administrator to enable recharge access.'
              )}
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className='wallet-recharge-card gap-0 py-0 shadow-sm'>
      <CardHeader className='wallet-recharge-header border-b p-5 sm:p-6'>
        <div className='min-w-0'>
          <CardTitle>{t('Online Recharge')}</CardTitle>
        </div>
      </CardHeader>
      <CardContent className='wallet-recharge-content space-y-5 p-5 sm:space-y-6 sm:p-6'>
        {/* Online Topup Section */}
        {hasAnyTopup ? (
          <div className='space-y-4 sm:space-y-6'>
            {hasConfigurableTopup && (
              <>
                {(showRechargeBonus || showInviteRanking) && (
                  <div className='grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(260px,0.72fr)]'>
                    {showRechargeBonus && (
                      <div className='border-primary/20 bg-primary/5 rounded-lg border p-3'>
                        <div className='flex items-start justify-between gap-3'>
                          <div className='min-w-0 space-y-1'>
                            <div className='flex items-center gap-2'>
                              <Gift className='text-primary h-4 w-4' />
                              <p className='text-sm font-semibold'>
                                {rechargeBonus.title ||
                                  t('Limited-time recharge bonus')}
                              </p>
                            </div>
                            <p className='text-muted-foreground text-xs leading-5'>
                              {rechargeBonus.description ||
                                t(
                                  'Recharge at least {{amount}} and receive {{bonus}} extra balance.',
                                  {
                                    amount: formatNumber(
                                      rechargeBonus.min_amount || minTopup
                                    ),
                                    bonus: bonusLabel,
                                  }
                                )}
                            </p>
                            {(rechargeBonus.start_time ||
                              rechargeBonus.end_time) && (
                              <p className='text-muted-foreground text-[11px]'>
                                {[
                                  rechargeBonus.start_time
                                    ? formatCampaignDate(
                                        rechargeBonus.start_time
                                      )
                                    : '',
                                  rechargeBonus.end_time
                                    ? formatCampaignDate(rechargeBonus.end_time)
                                    : '',
                                ]
                                  .filter(Boolean)
                                  .join(' - ')}
                              </p>
                            )}
                          </div>
                          {bonusLabel && (
                            <span className='bg-primary text-primary-foreground shrink-0 rounded-md px-2 py-1 text-xs font-semibold'>
                              {bonusLabel}
                            </span>
                          )}
                        </div>
                      </div>
                    )}
                    {showInviteRanking && (
                      <div className='rounded-lg border p-3'>
                        <div className='mb-2 flex items-center gap-2'>
                          <Trophy className='h-4 w-4 text-amber-500' />
                          <p className='text-sm font-semibold'>
                            {inviteRanking.title || t('Invite Leaderboard')}
                          </p>
                        </div>
                        <div className='space-y-1.5'>
                          {inviteRanking.items.slice(0, 5).map((item) => (
                            <div
                              key={`${item.rank}-${item.user_id}`}
                              className='flex items-center justify-between gap-3 text-xs'
                            >
                              <div className='flex min-w-0 items-center gap-2'>
                                <span className='bg-muted flex h-5 w-5 shrink-0 items-center justify-center rounded text-[11px] font-semibold'>
                                  {item.rank}
                                </span>
                                <span className='truncate'>
                                  {item.display_name}
                                </span>
                              </div>
                              <span className='text-muted-foreground shrink-0'>
                                {t('{{count}} invites', {
                                  count: item.invite_count,
                                })}
                              </span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}

                {presetAmounts.length > 0 && (
                  <div className='wallet-preset-amount-section space-y-2.5 sm:space-y-3'>
                    <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                      {t('Recharge Amount')}
                    </Label>
                    <div className='wallet-amount-grid grid grid-cols-2 gap-2 sm:gap-3 md:grid-cols-4'>
                      {presetAmounts.map((preset) => {
                        const discount =
                          preset.discount ??
                          resolveAmountDiscount(
                            preset.value,
                            topupInfo?.discount
                          )
                        const { actualPrice, savedAmount, hasDiscount } =
                          calculatePresetPricing(
                            preset.value,
                            priceRatio,
                            discount,
                            usdExchangeRate
                          )
                        const hasBonus =
                          showRechargeBonus &&
                          preset.value >= (rechargeBonus.min_amount || 0)
                        return (
                          <Button
                            key={`preset-${preset.value}`}
                            variant='outline'
                            className={cn(
                              'wallet-amount-option flex min-h-16 flex-col items-start rounded-lg px-3 py-2.5 text-left whitespace-normal sm:min-h-[88px] sm:p-4',
                              selectedPreset === preset.value
                                ? 'border-primary bg-primary/5 ring-primary/15 ring-2'
                                : 'border-border'
                            )}
                            onClick={() => onSelectPreset(preset)}
                            aria-pressed={selectedPreset === preset.value}
                          >
                            <div className='flex w-full items-center gap-2'>
                              <span className='wallet-amount-icon bg-primary/10 text-primary flex size-7 shrink-0 items-center justify-center rounded-full'>
                                <CircleDollarSign
                                  className='size-4'
                                  aria-hidden='true'
                                />
                              </span>
                              <div className='text-base font-semibold sm:text-lg'>
                                {formatSystemCurrencyUSD(preset.value)}
                              </div>
                              <div className='ml-auto flex items-center gap-1.5'>
                                {hasDiscount && (
                                  <div className='text-xs font-medium text-green-600'>
                                    {getDiscountLabel(discount)}
                                  </div>
                                )}
                                {hasBonus && rechargeBonus.show_bonus_ratio && (
                                  <div className='text-xs font-medium text-amber-600'>
                                    {bonusLabel}
                                  </div>
                                )}
                              </div>
                            </div>
                            <div className='text-muted-foreground mt-1.5 w-full text-xs sm:mt-2'>
                              <span>{t('Amount to pay')}</span>{' '}
                              {formatLocalPaymentAmount(actualPrice)}
                              {hasDiscount && savedAmount > 0 && (
                                <span className='text-green-600'>
                                  {' • '}
                                  {t('Save')}{' '}
                                  {formatLocalPaymentAmount(savedAmount)}
                                </span>
                              )}
                            </div>
                          </Button>
                        )
                      })}
                    </div>
                  </div>
                )}

                <div className='wallet-custom-amount-section space-y-2.5 sm:space-y-3'>
                  <Label
                    htmlFor='topup-amount'
                    className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
                  >
                    {t('Custom Amount')}
                  </Label>
                  <div className='wallet-custom-amount-row grid grid-cols-[minmax(0,1fr)_minmax(110px,0.55fr)] gap-2 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center'>
                    <div className='wallet-custom-amount-input relative min-w-0'>
                      <span
                        aria-hidden='true'
                        className='text-muted-foreground pointer-events-none absolute inset-y-0 left-3 z-10 flex items-center text-sm font-semibold'
                      >
                        $
                      </span>
                      <Input
                        id='topup-amount'
                        type='number'
                        value={localAmount}
                        onChange={(e) => handleAmountChange(e.target.value)}
                        min={minTopup}
                        placeholder={t('Minimum {{amount}}', {
                          amount: formatNumber(minTopup),
                        })}
                        className='h-9 pl-8 text-base sm:h-10 sm:text-lg'
                      />
                    </div>
                    <div className='wallet-custom-amount-total bg-muted/30 flex min-h-9 items-center justify-end gap-2 rounded-md border px-3 lg:min-w-52'>
                      {calculating ? (
                        <Skeleton className='h-5 w-16' />
                      ) : (
                        <span className='text-muted-foreground text-sm font-medium'>
                          {t('Amount to pay')}{' '}
                          {formatLocalPaymentAmount(paymentAmount)}
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                <div className='wallet-payment-layout'>
                  <div className='wallet-payment-methods space-y-2.5 sm:space-y-3'>
                    <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                      {t('Payment Method')}
                    </Label>
                    {standardPaymentMethodsContent}
                    {enableWaffoTopup &&
                      hasWaffoPaymentMethods &&
                      onWaffoMethodSelect && (
                        <div className='wallet-waffo-methods space-y-2.5 sm:space-y-3'>
                          <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                            {t('Waffo Payment')}
                          </Label>
                          <div className='wallet-payment-grid grid grid-cols-2 gap-2 sm:gap-3 lg:grid-cols-3'>
                            {waffoPayMethods?.map((method, index) => {
                              const loadingKey = `waffo-${index}`
                              const waffoMin = waffoMinTopup || 0
                              const belowMin = waffoMin > topupAmount
                              const disabledReason = belowMin
                                ? t('Minimum topup amount: {{amount}}', {
                                    amount: formatNumber(waffoMin),
                                  })
                                : undefined
                              const disabledLabel = belowMin
                                ? `${t('Minimum:')} ${formatNumber(waffoMin)}`
                                : undefined

                              let methodIcon = getPaymentIcon('waffo')
                              if (paymentLoading === loadingKey) {
                                methodIcon = (
                                  <Loader2 className='h-4 w-4 animate-spin' />
                                )
                              } else if (method.icon) {
                                methodIcon = (
                                  <img
                                    src={method.icon}
                                    alt={method.name}
                                    className='h-4 w-4 object-contain'
                                  />
                                )
                              }

                              const button = (
                                <Button
                                  key={`${method.name}-${method.payMethodType || index}`}
                                  variant='outline'
                                  onClick={() => {
                                    setSelectedPaymentKey(loadingKey)
                                    onWaffoMethodChange?.(method, index)
                                  }}
                                  disabled={belowMin || !!paymentLoading}
                                  aria-pressed={
                                    selectedPaymentKey === loadingKey
                                  }
                                  title={disabledReason}
                                  aria-label={
                                    disabledReason
                                      ? `${method.name}. ${disabledReason}`
                                      : method.name
                                  }
                                  className={cn(
                                    'min-h-16 min-w-0 justify-start gap-3 rounded-lg px-3 py-2.5 text-left',
                                    selectedPaymentKey === loadingKey &&
                                      'border-primary bg-primary/5 text-foreground ring-primary/15 ring-2'
                                  )}
                                >
                                  {methodIcon}
                                  <span className='flex min-w-0 flex-1 flex-col items-start gap-0.5'>
                                    <span className='max-w-full truncate'>
                                      {method.name}
                                    </span>
                                    {disabledLabel && (
                                      <span className='text-muted-foreground max-w-full truncate text-[11px] leading-4 font-normal'>
                                        {disabledLabel}
                                      </span>
                                    )}
                                  </span>
                                  {selectedPaymentKey === loadingKey && (
                                    <Check
                                      className='text-primary size-4'
                                      aria-hidden='true'
                                    />
                                  )}
                                </Button>
                              )

                              return belowMin ? (
                                <TooltipProvider
                                  key={`${method.name}-${method.payMethodType || index}`}
                                >
                                  <Tooltip>
                                    <TooltipTrigger render={button} />
                                    <TooltipContent>
                                      {disabledReason}
                                    </TooltipContent>
                                  </Tooltip>
                                </TooltipProvider>
                              ) : (
                                button
                              )
                            })}
                          </div>
                        </div>
                      )}
                  </div>

                  <div className='wallet-payment-assurances'>
                    <div className='grid gap-3 sm:grid-cols-3 lg:grid-cols-1'>
                      <div className='text-muted-foreground flex items-center gap-2'>
                        <ShieldCheck
                          className='text-success size-4'
                          aria-hidden='true'
                        />
                        <span className='min-w-0'>
                          <strong className='text-foreground block text-sm font-medium'>
                            {t('Secure payment')}
                          </strong>
                          <span className='block text-xs leading-5'>
                            {t(
                              'Payment details are handled by secure provider channels.'
                            )}
                          </span>
                        </span>
                      </div>
                      <div className='text-muted-foreground flex items-center gap-2'>
                        <Zap
                          className='text-warning size-4'
                          aria-hidden='true'
                        />
                        <span className='min-w-0'>
                          <strong className='text-foreground block text-sm font-medium'>
                            {t('Instant credit')}
                          </strong>
                          <span className='block text-xs leading-5'>
                            {t(
                              'Successful payments are credited automatically.'
                            )}
                          </span>
                        </span>
                      </div>
                      <div className='text-muted-foreground flex items-center gap-2'>
                        <Check
                          className='text-info size-4'
                          aria-hidden='true'
                        />
                        <span className='min-w-0'>
                          <strong className='text-foreground block text-sm font-medium'>
                            {t('Clear billing records')}
                          </strong>
                          <span className='block text-xs leading-5'>
                            {t(
                              'Clear usage records keep every charge easy to verify.'
                            )}
                          </span>
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                <div className='wallet-payment-footer space-y-3 border-t pt-5 sm:pt-6'>
                  <div className='wallet-pay-row'>
                    <Button
                      size='lg'
                      className='wallet-pay-button h-11 w-full text-sm shadow-sm'
                      onClick={handlePaymentContinue}
                      disabled={
                        paymentUnavailable || !!paymentLoading || calculating
                      }
                    >
                      {paymentLoading ? (
                        <Loader2 className='size-4 animate-spin' />
                      ) : (
                        <ShieldCheck className='size-4' />
                      )}
                      {t('Pay {{amount}} now', {
                        amount: formatLocalPaymentAmount(paymentAmount),
                      })}
                      <ArrowRight data-icon='inline-end' />
                    </Button>
                    <div className='wallet-pay-security text-muted-foreground flex items-center gap-2 text-xs'>
                      <CheckCircle2
                        className='text-success size-4 shrink-0'
                        aria-hidden='true'
                      />
                      <span>
                        <strong className='text-foreground block font-medium'>
                          {t('Secure payment')}
                        </strong>
                        <span>
                          {t(
                            'Payment details are handled by secure provider channels.'
                          )}
                        </span>
                      </span>
                    </div>
                  </div>
                </div>
              </>
            )}
          </div>
        ) : (
          <Alert>
            <AlertDescription>
              {t(
                'Online topup is not enabled. Please use redemption code or contact administrator.'
              )}
            </AlertDescription>
          </Alert>
        )}

        {/* Creem Products Section */}
        {enableCreemTopup &&
          Array.isArray(creemProducts) &&
          creemProducts.length > 0 &&
          onCreemProductSelect && (
            <div className='wallet-extra-payment space-y-2.5 border-t pt-4 sm:space-y-3 sm:pt-6'>
              <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                {t('Creem Payment')}
              </Label>
              <CreemProductsSection
                products={creemProducts}
                onProductSelect={onCreemProductSelect}
              />
            </div>
          )}

        {/* Redemption Code Section. The wallet reference surface keeps this
            flow on the dedicated CDK page, while the reusable card can still
            expose it to callers that explicitly opt in. */}
        {showRedemptionCode &&
          (redemptionEnabled ? (
            <div className='wallet-redemption-section space-y-2.5 border-t pt-4 sm:space-y-3 sm:pt-6'>
              <div className='flex items-center gap-2'>
                <Gift className='text-muted-foreground h-4 w-4' />
                <Label
                  htmlFor='redemption-code'
                  className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
                >
                  {t('Have a Code?')}
                </Label>
              </div>
              <div className='grid grid-cols-[minmax(0,1fr)_auto] gap-2'>
                <Input
                  id='redemption-code'
                  value={redemptionCode}
                  onChange={(e) => onRedemptionCodeChange(e.target.value)}
                  placeholder={t('Enter your redemption code')}
                  className='h-9 min-w-0'
                />
                <Button
                  onClick={onRedeem}
                  disabled={redeeming}
                  variant='outline'
                  className='h-9 px-4'
                >
                  {redeeming && (
                    <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                  )}
                  {t('Redeem')}
                </Button>
              </div>
              {topupLink && (
                <p className='text-muted-foreground text-xs'>
                  {t('Need a redemption code?')}{' '}
                  <a
                    href={topupLink}
                    target='_blank'
                    rel='noopener noreferrer'
                    className='inline-flex items-center gap-1 underline-offset-4 hover:underline'
                  >
                    {t('Get one here')}
                    <ExternalLink className='h-3 w-3' />
                  </a>
                </p>
              )}
            </div>
          ) : (
            <Alert className='border-t'>
              <AlertDescription>
                {t(
                  'Redemption codes are disabled until the administrator confirms compliance terms.'
                )}
              </AlertDescription>
            </Alert>
          ))}
      </CardContent>
    </Card>
  )
}
