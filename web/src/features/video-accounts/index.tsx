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
import {
  Add01Icon,
  Delete02Icon,
  PencilEdit01Icon,
  Refresh01Icon,
  Search01Icon,
  Video01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from '@/components/ui/input-group'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatTimestampToDate } from '@/lib/format'

import { getVideoAccounts, syncVideoAccount } from './api'
import { VideoAccountDeleteDialog } from './components/video-account-delete-dialog'
import { VideoAccountDialog } from './components/video-account-dialog'
import type { VideoAccount, VideoModel } from './types'

type StatusFilter = 'all' | '1' | '0'

const PAGE_SIZE = 20

function modelLabel(model: VideoModel) {
  return model.display_name?.trim() || model.id
}

function modelPricing(model: VideoModel) {
  const amount = model.pricing?.amount
  if (typeof amount !== 'number' || !Number.isFinite(amount)) return null
  const currency = model.pricing?.currency?.trim() || 'CNY'
  const mode = model.pricing?.mode?.trim()
  const suffix = mode ? ` / ${mode.replaceAll('_', ' ')}` : ''
  return `${amount} ${currency}${suffix}`
}

function AccountModelBadges(props: { account: VideoAccount }) {
  const { t } = useTranslation()
  const models = props.account.models ?? []
  if (models.length === 0) {
    return <span className='text-muted-foreground text-xs'>{t('No data')}</span>
  }
  const visibleModels = models.slice(0, 3)
  return (
    <div className='flex max-w-[360px] flex-wrap gap-1'>
      {visibleModels.map((model) => (
        <Badge
          key={model.id}
          variant='outline'
          title={modelPricing(model) ?? undefined}
        >
          {modelLabel(model)}
        </Badge>
      ))}
      {models.length > visibleModels.length && (
        <Badge variant='secondary'>
          +{models.length - visibleModels.length}
        </Badge>
      )}
    </div>
  )
}

function VideoAccountRows(props: {
  accounts: VideoAccount[]
  isSyncing: number | null
  onEdit: (account: VideoAccount) => void
  onDelete: (account: VideoAccount) => void
  onSync: (account: VideoAccount) => void
}) {
  const { t } = useTranslation()
  if (props.accounts.length === 0) {
    return (
      <TableRow>
        <TableCell colSpan={7} className='h-32 text-center'>
          <span className='text-muted-foreground'>{t('No data')}</span>
        </TableCell>
      </TableRow>
    )
  }

  return props.accounts.map((account) => (
    <TableRow key={account.id}>
      <TableCell>
        <div className='flex min-w-[180px] flex-col gap-1'>
          <span className='font-medium'>{account.name}</span>
          <div className='text-muted-foreground flex max-w-[260px] items-center gap-1 text-xs'>
            <span>{t('Token ID')}:</span>
            <code className='truncate'>{account.token_id}</code>
          </div>
          <div className='text-muted-foreground flex items-center gap-1 text-xs'>
            <span>{t('API key')}:</span>
            <code>{account.api_key_masked || '••••'}</code>
          </div>
          {account.remark && (
            <span className='text-muted-foreground max-w-[220px] truncate text-xs'>
              {account.remark}
            </span>
          )}
        </div>
      </TableCell>
      <TableCell>
        <StatusBadge
          label={account.status === 1 ? t('Enabled') : t('Disabled')}
          variant={account.status === 1 ? 'success' : 'neutral'}
          showDot
          copyable={false}
        />
      </TableCell>
      <TableCell>
        <div className='flex max-w-[200px] flex-wrap gap-1'>
          {account.groups.map((group) => (
            <Badge key={group} variant='secondary'>
              {group}
            </Badge>
          ))}
        </div>
      </TableCell>
      <TableCell>
        <AccountModelBadges account={account} />
      </TableCell>
      <TableCell>
        <code className='text-muted-foreground text-xs'>
          {account.base_url}
        </code>
      </TableCell>
      <TableCell>
        <span className='text-muted-foreground text-xs'>
          {formatTimestampToDate(account.last_model_sync_at)}
        </span>
      </TableCell>
      <TableCell>
        <div className='flex items-center justify-end gap-1'>
          <Button
            variant='ghost'
            size='icon-sm'
            onClick={() => props.onSync(account)}
            disabled={props.isSyncing === account.id}
            aria-label={t('Sync models')}
            title={t('Sync models')}
          >
            {props.isSyncing === account.id ? (
              <Spinner />
            ) : (
              <HugeiconsIcon icon={Refresh01Icon} strokeWidth={2} />
            )}
          </Button>
          <Button
            variant='ghost'
            size='icon-sm'
            onClick={() => props.onEdit(account)}
            aria-label={t('Edit')}
            title={t('Edit')}
          >
            <HugeiconsIcon icon={PencilEdit01Icon} strokeWidth={2} />
          </Button>
          <Button
            variant='ghost'
            size='icon-sm'
            onClick={() => props.onDelete(account)}
            aria-label={t('Delete')}
            title={t('Delete')}
          >
            <HugeiconsIcon icon={Delete02Icon} strokeWidth={2} />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  ))
}

export function VideoAccounts() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState<StatusFilter>('all')
  const [page, setPage] = useState(1)
  const [dialogAccount, setDialogAccount] = useState<
    VideoAccount | null | undefined
  >(undefined)
  const [deleteAccount, setDeleteAccount] = useState<VideoAccount | null>(null)
  const [syncingId, setSyncingId] = useState<number | null>(null)

  const query = useQuery({
    queryKey: ['video-accounts', page, search, status],
    queryFn: () =>
      getVideoAccounts({
        p: page,
        page_size: PAGE_SIZE,
        search: search.trim() || undefined,
        status: status === 'all' ? undefined : Number(status),
      }),
    placeholderData: (previousData) => previousData,
  })

  const list = query.data?.data
  const accounts = list?.items ?? []
  const pageCount = Math.max(1, Math.ceil((list?.total ?? 0) / PAGE_SIZE))

  const handleSync = async (account: VideoAccount) => {
    setSyncingId(account.id)
    try {
      const result = await syncVideoAccount(account.id)
      if (!result.success) {
        toast.error(result.message || t('Failed to sync models'))
        return
      }
      toast.success(t('Models synced successfully'))
      queryClient.invalidateQueries({ queryKey: ['video-accounts'] })
    } catch {
      toast.error(t('Failed to sync models'))
    } finally {
      setSyncingId(null)
    }
  }

  const statusItems = useMemo(
    () => [
      { value: 'all', label: t('All statuses') },
      { value: '1', label: t('Enabled') },
      { value: '0', label: t('Disabled') },
    ],
    [t]
  )

  let accountContent: ReactNode
  if (query.isLoading) {
    accountContent = (
      <div className='flex flex-col gap-2'>
        {Array.from({ length: 5 }, (_, index) => (
          <Skeleton key={index} className='h-15 w-full' />
        ))}
      </div>
    )
  } else if (query.isError || query.data?.success === false) {
    accountContent = (
      <Empty className='border'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <HugeiconsIcon icon={Video01Icon} strokeWidth={2} />
          </EmptyMedia>
          <EmptyTitle>{t('Failed to load')}</EmptyTitle>
          <EmptyDescription>
            {query.data?.message || t('Refresh the list and try again.')}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  } else if (accounts.length === 0 && !search && status === 'all') {
    accountContent = (
      <Empty className='border'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <HugeiconsIcon icon={Video01Icon} strokeWidth={2} />
          </EmptyMedia>
          <EmptyTitle>{t('No video accounts configured')}</EmptyTitle>
          <EmptyDescription>
            {t('Add a dedicated CTMOAI account to load its model catalog.')}
          </EmptyDescription>
        </EmptyHeader>
        <Button size='sm' onClick={() => setDialogAccount(null)}>
          <HugeiconsIcon
            icon={Add01Icon}
            strokeWidth={2}
            data-icon='inline-start'
          />
          {t('Add Video Account')}
        </Button>
      </Empty>
    )
  } else {
    accountContent = (
      <div className='min-h-0 flex-1 overflow-auto rounded-xl border'>
        <Table>
          <TableHeader className='bg-muted/40 sticky top-0 z-1'>
            <TableRow>
              <TableHead>{t('Name')}</TableHead>
              <TableHead>{t('Status')}</TableHead>
              <TableHead>{t('Groups')}</TableHead>
              <TableHead>{t('Models')}</TableHead>
              <TableHead>{t('Base URL')}</TableHead>
              <TableHead>{t('Last refresh')}</TableHead>
              <TableHead className='text-right'>{t('Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <VideoAccountRows
              accounts={accounts}
              isSyncing={syncingId}
              onEdit={(account) => setDialogAccount(account)}
              onDelete={(account) => setDeleteAccount(account)}
              onSync={handleSync}
            />
          </TableBody>
        </Table>
      </div>
    )
  }

  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>{t('Video Accounts')}</SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t(
            'Manage dedicated CTMOAI video accounts and key-scoped model catalogs.'
          )}
        </SectionPageLayout.Description>
        <SectionPageLayout.Actions>
          <Button size='sm' onClick={() => setDialogAccount(null)}>
            <HugeiconsIcon
              icon={Add01Icon}
              strokeWidth={2}
              data-icon='inline-start'
            />
            {t('Add Video Account')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='flex h-full min-h-0 flex-col gap-3'>
            <div className='flex flex-wrap items-center gap-2'>
              <InputGroup className='w-full max-w-sm'>
                <InputGroupInput
                  value={search}
                  onChange={(event) => {
                    setPage(1)
                    setSearch(event.target.value)
                  }}
                  placeholder={t('Search')}
                  aria-label={t('Search')}
                />
                <InputGroupAddon>
                  <HugeiconsIcon icon={Search01Icon} strokeWidth={2} />
                </InputGroupAddon>
              </InputGroup>
              <Select<string>
                items={statusItems}
                value={status}
                onValueChange={(value) => {
                  if (value === null) return
                  setPage(1)
                  setStatus(value as StatusFilter)
                }}
              >
                <SelectTrigger aria-label={t('Status')}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {statusItems.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              {query.isFetching && (
                <Spinner className='text-muted-foreground' />
              )}
            </div>

            {accountContent}

            {list && list.total > 0 && (
              <div className='text-muted-foreground flex items-center justify-between text-xs'>
                <span>
                  {list.total} {t('Video Accounts').toLowerCase()}
                </span>
                {pageCount > 1 && (
                  <div className='flex items-center gap-2'>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={() =>
                        setPage((current) => Math.max(1, current - 1))
                      }
                      disabled={page <= 1}
                    >
                      {t('Previous')}
                    </Button>
                    <span>
                      {page} / {pageCount}
                    </span>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={() =>
                        setPage((current) => Math.min(pageCount, current + 1))
                      }
                      disabled={page >= pageCount}
                    >
                      {t('Next')}
                    </Button>
                  </div>
                )}
              </div>
            )}
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <VideoAccountDialog
        key={
          dialogAccount === undefined ? 'closed' : (dialogAccount?.id ?? 'new')
        }
        open={dialogAccount !== undefined}
        account={dialogAccount}
        onOpenChange={(open) => !open && setDialogAccount(undefined)}
      />
      <VideoAccountDeleteDialog
        account={deleteAccount}
        onOpenChange={(open) => !open && setDeleteAccount(null)}
      />
    </>
  )
}
