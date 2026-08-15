/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { getSelf } from '@/lib/api'

import { SubscriptionPlansCard } from './components/subscription-plans-card'
import { useTopupInfo } from './hooks'
import type { UserWalletData } from './types'

export function SubscriptionPlansPage() {
  const { t } = useTranslation()
  const { topupInfo } = useTopupInfo()
  const [user, setUser] = useState<UserWalletData | null>(null)

  const fetchUser = useCallback(async () => {
    try {
      const response = await getSelf()
      if (response.success && response.data) {
        setUser(response.data as UserWalletData)
      }
    } catch {
      setUser(null)
    }
  }, [])

  useEffect(() => {
    void fetchUser()
  }, [fetchUser])

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Subscription Plans')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t('Subscribe to a plan for model access')}
      </SectionPageLayout.Description>
      <SectionPageLayout.Content>
        <div className='w-full'>
          <SubscriptionPlansCard
            topupInfo={topupInfo}
            userQuota={user?.quota}
            onPurchaseSuccess={fetchUser}
          />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
