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
import { useQuery } from '@tanstack/react-query'
import { Search, UsersRound } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { AffiliateInviteesTable } from '@/features/affiliate-commissions/components/affiliate-invitees-table'

import { getAdminAffiliateInvitees } from '../api'

export function AffiliateInviteesAdminPanel() {
  const { t } = useTranslation()
  const [promoterInput, setPromoterInput] = useState('')
  const [searchValue, setSearchValue] = useState('')
  const searchIsId = /^\d+$/.test(searchValue)
  const inviteesQuery = useQuery({
    queryKey: ['admin-affiliate-invitees', searchValue],
    queryFn: () =>
      getAdminAffiliateInvitees(
        searchIsId
          ? { promoter_id: Number(searchValue), p: 1, page_size: 50 }
          : { promoter_username: searchValue, p: 1, page_size: 50 }
      ),
    enabled: searchValue.length > 0,
  })

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const value = promoterInput.trim()
    if (value) setSearchValue(value)
  }

  const response = inviteesQuery.data
  const data = response?.success ? response.data : undefined

  return (
    <Card>
      <CardHeader className='gap-3 border-b sm:flex-row sm:items-center sm:justify-between'>
        <div className='flex items-center gap-2'>
          <UsersRound className='text-primary size-4' />
          <div>
            <CardTitle className='text-base'>
              {t('Invitation details')}
            </CardTitle>
            <p className='text-muted-foreground mt-1 text-xs'>
              {t(
                'Search a promoter to see the users they invited and each contribution.'
              )}
            </p>
          </div>
        </div>
        <form className='flex w-full gap-2 sm:w-auto' onSubmit={handleSubmit}>
          <Input
            value={promoterInput}
            onChange={(event) => setPromoterInput(event.target.value)}
            placeholder={t('Enter a promoter user ID or username')}
            aria-label={t('Search promoter')}
            className='sm:w-64'
          />
          <Button type='submit' disabled={!promoterInput.trim()}>
            <Search className='size-4' />
            {t('Search')}
          </Button>
        </form>
      </CardHeader>
      <CardContent className='pt-4'>
        {!searchValue ? (
          <div className='text-muted-foreground flex min-h-28 flex-col items-center justify-center gap-2 text-center text-sm'>
            <Search className='size-5 opacity-60' />
            <span>{t('Select a promoter to view invitation details.')}</span>
          </div>
        ) : response && !response.success ? (
          <div className='text-destructive flex min-h-28 items-center justify-center text-sm'>
            {response.message || t('No invitation details')}
          </div>
        ) : (
          <AffiliateInviteesTable
            items={data?.items || []}
            total={data?.total || 0}
            isLoading={inviteesQuery.isLoading || inviteesQuery.isFetching}
          />
        )}
      </CardContent>
    </Card>
  )
}
