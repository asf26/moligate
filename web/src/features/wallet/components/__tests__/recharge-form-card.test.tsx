import { render, screen } from '@testing-library/react'
import i18next from 'i18next'
import { beforeAll, describe, expect, test, vi } from 'vitest'

import type { TopupInfo } from '../../types'
import { RechargeFormCard } from '../recharge-form-card'

vi.mock('../wallet-contact-support-button', () => ({
  WalletContactSupportButton: () => null,
}))

const baseTopupInfo: TopupInfo = {
  enable_online_topup: true,
  enable_stripe_topup: false,
  enable_creem_topup: false,
  enable_waffo_topup: false,
  enable_waffo_pancake_topup: false,
  enable_redemption: true,
  pay_methods: [{ name: 'WeChat', type: 'wechat' }],
  min_topup: 10,
  stripe_min_topup: 10,
  amount_options: [10],
  discount: {},
}

function renderRecharge(topUpEnabled: boolean) {
  return render(
    <RechargeFormCard
      topupInfo={{ ...baseTopupInfo, top_up_enabled: topUpEnabled }}
      topUpEnabled={topUpEnabled}
      presetAmounts={[]}
      selectedPreset={null}
      onSelectPreset={() => undefined}
      topupAmount={10}
      onTopupAmountChange={() => undefined}
      paymentAmount={0}
      calculating={false}
      onPaymentMethodSelect={() => undefined}
      paymentLoading={null}
      redemptionCode=''
      onRedemptionCodeChange={() => undefined}
      onRedeem={() => undefined}
      redeeming={false}
    />
  )
}

describe('recharge permission gate', () => {
  beforeAll(() => {
    i18next.addResourceBundle('en', 'translation', {
      'Add Funds': 'Add Funds',
      'Choose an amount and payment method':
        'Choose an amount and payment method',
      'Recharge is locked for this account. Please join the official group and contact an administrator to enable recharge access.':
        'Recharge is locked for this account. Please join the official group and contact an administrator to enable recharge access.',
      Amount: 'Amount',
      'Custom Amount': 'Custom Amount',
      'Payment Method': 'Payment Method',
      'Have a Code?': 'Have a Code?',
      'Enter your redemption code': 'Enter your redemption code',
      Redeem: 'Redeem',
    })
  })

  test('hides payment and redemption controls when recharge is disabled', () => {
    renderRecharge(false)

    expect(
      screen.getByText(
        'Recharge is locked for this account. Please join the official group and contact an administrator to enable recharge access.'
      )
    ).toBeInTheDocument()
    expect(screen.queryByLabelText('Custom Amount')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Have a Code?')).not.toBeInTheDocument()
  })

  test('renders the existing recharge form when permission is enabled', () => {
    renderRecharge(true)

    expect(screen.getByLabelText('Custom Amount')).toBeInTheDocument()
    expect(screen.getByLabelText('Have a Code?')).toBeInTheDocument()
  })
})
