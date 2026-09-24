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
import { useState, useEffect, useCallback, useRef } from 'react'

import { SectionPageLayout } from '@/components/layout'
import { getSelf } from '@/lib/api'

import { BillingHistoryDialog } from './components/dialogs/billing-history-dialog'
import { CreemConfirmDialog } from './components/dialogs/creem-confirm-dialog'
import { PaymentConfirmDialog } from './components/dialogs/payment-confirm-dialog'
import { RechargeFormCard } from './components/recharge-form-card'
import { WalletSidePanel } from './components/wallet-side-panel'
import { WalletStatsCard } from './components/wallet-stats-card'
import { PAYMENT_TYPES } from './constants'
import {
  useTopupInfo,
  useBillingHistory,
  usePayment,
  useRedemption,
  useCreemPayment,
  useWaffoPayment,
  useWaffoPancakePayment,
} from './hooks'
import {
  getDefaultPaymentType,
  getMinTopupAmount,
  dispatchSelectedPayment,
  resolveAmountDiscount,
} from './lib'
import type {
  UserWalletData,
  PaymentMethod,
  PresetAmount,
  CreemProduct,
  WaffoPayMethod,
} from './types'

interface WalletProps {
  initialShowHistory?: boolean
}

export function Wallet(props: WalletProps) {
  const [user, setUser] = useState<UserWalletData | null>(null)
  const [userLoading, setUserLoading] = useState(true)
  const [topupAmount, setTopupAmount] = useState(0)
  const [selectedPreset, setSelectedPreset] = useState<number | null>(null)
  const [selectedPaymentMethod, setSelectedPaymentMethod] =
    useState<PaymentMethod>()
  const [selectedWaffoMethodIndex, setSelectedWaffoMethodIndex] = useState<
    number | null
  >(null)
  const [paymentLoading, setPaymentLoading] = useState<string | null>(null)
  const [confirmDialogOpen, setConfirmDialogOpen] = useState(false)
  const [billingDialogOpen, setBillingDialogOpen] = useState(false)
  const [redemptionCode, setRedemptionCode] = useState('')
  const [creemDialogOpen, setCreemDialogOpen] = useState(false)
  const [selectedCreemProduct, setSelectedCreemProduct] =
    useState<CreemProduct | null>(null)

  const { topupInfo, presetAmounts, loading: topupLoading } = useTopupInfo()
  const billingHistory = useBillingHistory({
    initialPageSize: 5,
    userOnly: true,
  })
  const rechargeEnabled = topupInfo?.top_up_enabled !== false

  const {
    amount: paymentAmount,
    calculating,
    processing,
    calculatePaymentAmount,
    processPayment,
  } = usePayment()
  const { redeeming, redeemCode } = useRedemption()
  const { processing: creemProcessing, processCreemPayment } = useCreemPayment()
  const { processing: waffoProcessing, processWaffoPayment } = useWaffoPayment()
  const { processing: pancakeProcessing, processWaffoPancakePayment } =
    useWaffoPancakePayment()

  // Fetch and refresh user data
  const fetchUser = useCallback(async () => {
    try {
      setUserLoading(true)
      const response = await getSelf()
      if (response.success && response.data) {
        setUser(response.data as UserWalletData)
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to fetch user data:', error)
    } finally {
      setUserLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchUser()
  }, [fetchUser])

  useEffect(() => {
    if (props.initialShowHistory) {
      setBillingDialogOpen(true)
      window.history.replaceState({}, '', window.location.pathname)
    }
  }, [props.initialShowHistory])

  // Initialize the form with the first configured preset so the default
  // payment state is visible as soon as the wallet opens.
  const topupAmountInitializedRef = useRef(false)
  useEffect(() => {
    if (
      topupLoading ||
      !topupInfo ||
      !rechargeEnabled ||
      topupAmountInitializedRef.current
    ) {
      return
    }

    topupAmountInitializedRef.current = true
    const minTopup = getMinTopupAmount(topupInfo)
    const initialPreset = presetAmounts[0]
    const initialAmount = initialPreset?.value ?? minTopup
    setTopupAmount(initialAmount)
    setSelectedPreset(initialPreset?.value ?? null)

    // Calculate initial payment amount with default payment type
    const defaultPaymentType = getDefaultPaymentType(topupInfo)
    calculatePaymentAmount(initialAmount, defaultPaymentType)
  }, [
    topupInfo,
    topupLoading,
    rechargeEnabled,
    presetAmounts,
    calculatePaymentAmount,
  ])

  // Get current payment type (selected or default)
  const getCurrentPaymentType = useCallback(() => {
    return selectedPaymentMethod?.type || getDefaultPaymentType(topupInfo)
  }, [selectedPaymentMethod, topupInfo])

  // Handle preset selection
  const handleSelectPreset = (preset: PresetAmount) => {
    if (!rechargeEnabled) return
    setTopupAmount(preset.value)
    setSelectedPreset(preset.value)
    calculatePaymentAmount(preset.value, getCurrentPaymentType())
  }

  // Handle topup amount change
  const handleTopupAmountChange = (amount: number) => {
    if (!rechargeEnabled) return
    setTopupAmount(amount)
    setSelectedPreset(null)
    calculatePaymentAmount(amount, getCurrentPaymentType())
  }

  // Handle payment method selection
  const handlePaymentMethodChange = async (method: PaymentMethod) => {
    if (!rechargeEnabled) return
    setSelectedPaymentMethod(method)
    setSelectedWaffoMethodIndex(null)
    setPaymentLoading(method.type)

    try {
      await calculatePaymentAmount(topupAmount, method.type)
    } finally {
      setPaymentLoading(null)
    }
  }

  const handlePaymentMethodSelect = async (method: PaymentMethod) => {
    if (!rechargeEnabled) return
    const minTopup = getMinTopupAmount(topupInfo)
    if (topupAmount < minTopup) {
      return
    }

    await handlePaymentMethodChange(method)
    setConfirmDialogOpen(true)
  }

  // Handle payment confirmation
  const handlePaymentConfirm = async () => {
    if (!rechargeEnabled || !selectedPaymentMethod) return

    const success = await dispatchSelectedPayment(
      selectedPaymentMethod,
      topupAmount,
      selectedWaffoMethodIndex,
      {
        regular: processPayment,
        waffo: processWaffoPayment,
        waffoPancake: processWaffoPancakePayment,
      }
    )

    if (success) {
      setConfirmDialogOpen(false)
      await fetchUser()
    }
  }

  // Handle redemption
  const handleRedeem = async () => {
    if (!rechargeEnabled || !redemptionCode) return

    const success = await redeemCode(redemptionCode)
    if (success) {
      setRedemptionCode('')
      await fetchUser()
    }
  }

  // Handle Creem product selection
  const handleCreemProductSelect = (product: CreemProduct) => {
    if (!rechargeEnabled) return
    setSelectedCreemProduct(product)
    setCreemDialogOpen(true)
  }

  // Handle Creem payment confirmation
  const handleCreemConfirm = async () => {
    if (!rechargeEnabled || !selectedCreemProduct) return

    const success = await processCreemPayment(selectedCreemProduct.productId)
    if (success) {
      setCreemDialogOpen(false)
      setSelectedCreemProduct(null)
      await fetchUser()
    }
  }

  const handleWaffoMethodChange = async (
    method: WaffoPayMethod,
    index: number
  ) => {
    if (!rechargeEnabled) return
    const loadingKey = `waffo-${index}`
    setSelectedPaymentMethod({
      name: method.name,
      type: PAYMENT_TYPES.WAFFO,
      icon: method.icon,
    })
    setSelectedWaffoMethodIndex(index)
    setPaymentLoading(loadingKey)

    try {
      await calculatePaymentAmount(topupAmount, PAYMENT_TYPES.WAFFO)
    } finally {
      setPaymentLoading(null)
    }
  }

  const handleWaffoMethodSelect = async (
    method: WaffoPayMethod,
    index: number
  ) => {
    if (!rechargeEnabled) return
    await handleWaffoMethodChange(method, index)
    setConfirmDialogOpen(true)
  }

  // Get discount rate for current topup amount
  const getDiscountRate = useCallback(() => {
    return resolveAmountDiscount(topupAmount, topupInfo?.discount)
  }, [topupInfo?.discount, topupAmount])

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Content>
          <div className='wallet-reference-page flex w-full flex-col gap-5 sm:gap-6'>
            <WalletStatsCard
              user={user}
              loading={userLoading}
              topupCount={billingHistory.total}
            />

            <div className='wallet-workspace-grid grid items-start gap-5 @6xl/content:grid-cols-[minmax(0,1.65fr)_minmax(360px,1fr)]'>
              <div
                id='wallet-add-funds'
                className='wallet-recharge-column min-w-0 scroll-mt-4'
              >
                <RechargeFormCard
                  topupInfo={topupInfo}
                  topUpEnabled={topupInfo?.top_up_enabled}
                  presetAmounts={presetAmounts}
                  selectedPreset={selectedPreset}
                  onSelectPreset={handleSelectPreset}
                  topupAmount={topupAmount}
                  onTopupAmountChange={handleTopupAmountChange}
                  paymentAmount={paymentAmount}
                  calculating={calculating}
                  onPaymentMethodSelect={handlePaymentMethodSelect}
                  onPaymentMethodChange={handlePaymentMethodChange}
                  paymentLoading={paymentLoading}
                  redemptionCode={redemptionCode}
                  onRedemptionCodeChange={setRedemptionCode}
                  onRedeem={handleRedeem}
                  redeeming={redeeming}
                  topupLink={topupInfo?.topup_link}
                  loading={topupLoading}
                  priceRatio={1}
                  usdExchangeRate={1}
                  creemProducts={topupInfo?.creem_products}
                  enableCreemTopup={topupInfo?.enable_creem_topup}
                  onCreemProductSelect={handleCreemProductSelect}
                  enableWaffoTopup={topupInfo?.enable_waffo_topup}
                  waffoPayMethods={topupInfo?.waffo_pay_methods}
                  waffoMinTopup={topupInfo?.waffo_min_topup}
                  onWaffoMethodSelect={handleWaffoMethodSelect}
                  onWaffoMethodChange={handleWaffoMethodChange}
                  enableWaffoPancakeTopup={
                    topupInfo?.enable_waffo_pancake_topup
                  }
                />
              </div>
              <WalletSidePanel
                onOpenBilling={() => setBillingDialogOpen(true)}
                billing={{
                  records: billingHistory.records,
                  loading: billingHistory.loading,
                  total: billingHistory.total,
                  page: billingHistory.page,
                  pageSize: billingHistory.pageSize,
                  onPageChange: billingHistory.handlePageChange,
                }}
              />
            </div>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <PaymentConfirmDialog
        open={confirmDialogOpen}
        onOpenChange={setConfirmDialogOpen}
        onConfirm={handlePaymentConfirm}
        topupAmount={topupAmount}
        paymentAmount={paymentAmount}
        paymentMethod={selectedPaymentMethod}
        calculating={calculating}
        processing={processing || waffoProcessing || pancakeProcessing}
        discountRate={getDiscountRate()}
      />

      <BillingHistoryDialog
        open={billingDialogOpen}
        onOpenChange={setBillingDialogOpen}
      />

      <CreemConfirmDialog
        open={creemDialogOpen}
        onOpenChange={setCreemDialogOpen}
        onConfirm={handleCreemConfirm}
        product={selectedCreemProduct}
        processing={creemProcessing}
      />
    </>
  )
}
