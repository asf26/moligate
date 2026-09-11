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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertCircle,
  Building2,
  ChevronLeft,
  ChevronRight,
  Database,
  Download,
  FileSpreadsheet,
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
} from 'lucide-react'
import { useEffect, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'

import {
  createEnterpriseBillingAccount,
  deleteEnterpriseBillingAccount,
  exportEnterpriseBillingCsv,
  getEnterpriseBillingAccounts,
  getEnterpriseBillingReport,
  updateEnterpriseBillingAccount,
} from './api'
import type {
  EnterpriseBillingAccount,
  EnterpriseBillingAccountPayload,
  EnterpriseBillingReport,
  EnterpriseBillingSource,
} from './types'

const PAGE_SIZE = 50
const DEFAULT_RULE = 'p * 2.5 + c * 15'
const EMPTY_ACCOUNTS: EnterpriseBillingAccount[] = []

type AccountForm = {
  name: string
  code: string
  source: EnterpriseBillingSource
  usernames: string
  pricing_rule: string
  enabled: boolean
}

function currentMonth() {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}

function emptyForm(): AccountForm {
  return {
    name: '',
    code: '',
    source: 'local',
    usernames: '',
    pricing_rule: DEFAULT_RULE,
    enabled: true,
  }
}

function formFromAccount(account: EnterpriseBillingAccount): AccountForm {
  return {
    name: account.name,
    code: account.code,
    source: account.source,
    usernames: account.usernames.join('\n'),
    pricing_rule: account.pricing_rule,
    enabled: account.enabled,
  }
}

function toPayload(form: AccountForm): EnterpriseBillingAccountPayload {
  return {
    name: form.name,
    code: form.code,
    source: form.source,
    usernames: form.usernames
      .split(/[\n,]/)
      .map((value) => value.trim())
      .filter(Boolean),
    pricing_rule: form.pricing_rule,
    enabled: form.enabled,
  }
}

function formatNumber(value: number) {
  return new Intl.NumberFormat().format(value || 0)
}

function formatPrice(value: number, currency: string) {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: currency || 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 6,
  }).format(value || 0)
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}

function AccountFormDialog({
  open,
  account,
  onOpenChange,
  onSubmit,
  pending,
}: {
  open: boolean
  account: EnterpriseBillingAccount | null
  onOpenChange: (open: boolean) => void
  onSubmit: (payload: EnterpriseBillingAccountPayload) => void
  pending: boolean
}) {
  const { t } = useTranslation()
  const [form, setForm] = useState<AccountForm>(emptyForm)

  useEffect(() => {
    setForm(account ? formFromAccount(account) : emptyForm())
  }, [account, open])

  const update = <K extends keyof AccountForm>(key: K, value: AccountForm[K]) =>
    setForm((previous) => ({ ...previous, [key]: value }))

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[calc(100vh-2rem)] overflow-y-auto sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>
            {account
              ? t('Edit enterprise account')
              : t('Add enterprise account')}
          </DialogTitle>
          <DialogDescription>
            {t('Connect a usage source and define the contract pricing rule.')}
          </DialogDescription>
        </DialogHeader>
        <form
          className='grid gap-4'
          onSubmit={(event) => {
            event.preventDefault()
            onSubmit(toPayload(form))
          }}
        >
          <div className='grid gap-2 sm:grid-cols-2'>
            <div className='grid gap-2'>
              <Label htmlFor='enterprise-name'>{t('Company name')}</Label>
              <Input
                id='enterprise-name'
                value={form.name}
                onChange={(event) => update('name', event.target.value)}
                placeholder={t('e.g. Acme AI')}
                required
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='enterprise-code'>{t('Account code')}</Label>
              <Input
                id='enterprise-code'
                value={form.code}
                onChange={(event) => update('code', event.target.value)}
                placeholder={t('e.g. acme-ai')}
                required
              />
            </div>
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='enterprise-source'>{t('Usage source')}</Label>
            <NativeSelect
              id='enterprise-source'
              className='w-full'
              value={form.source}
              onChange={(event) =>
                update('source', event.target.value as EnterpriseBillingSource)
              }
            >
              <NativeSelectOption value='local'>
                {t('This gateway log database')}
              </NativeSelectOption>
              <NativeSelectOption value='s10'>s10</NativeSelectOption>
              <NativeSelectOption value='moligate'>
                moligate.com
              </NativeSelectOption>
            </NativeSelect>
            <p className='text-muted-foreground text-xs'>
              {t('Remote sources use server-side DSN environment variables.')}
            </p>
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='enterprise-usernames'>{t('Usage accounts')}</Label>
            <Textarea
              id='enterprise-usernames'
              value={form.usernames}
              onChange={(event) => update('usernames', event.target.value)}
              placeholder={t('One username per line')}
              rows={4}
              required
            />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='enterprise-pricing-rule'>
              {t('Contract pricing rule')}
            </Label>
            <Textarea
              id='enterprise-pricing-rule'
              value={form.pricing_rule}
              onChange={(event) => update('pricing_rule', event.target.value)}
              placeholder={t('Example: p * 2.5 + c * 15')}
              rows={4}
              className='font-mono text-xs'
              required
            />
            <p className='text-muted-foreground text-xs'>
              {t(
                'Prices are USD per 1M tokens. Available variables: p, c, len.'
              )}
            </p>
          </div>
          <label className='flex items-center gap-2 text-sm'>
            <Input
              type='checkbox'
              className='size-4'
              checked={form.enabled}
              onChange={(event) => update('enabled', event.target.checked)}
            />
            {t('Enabled')}
          </label>
          <DialogFooter>
            <DialogClose render={<Button type='button' variant='outline' />}>
              {t('Cancel')}
            </DialogClose>
            <Button type='submit' disabled={pending}>
              {pending ? t('Saving...') : t('Save account')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function SummaryCards({ report }: { report: EnterpriseBillingReport }) {
  const { t } = useTranslation()
  const cards = [
    [t('Usage records'), formatNumber(report.summary.raw_count)],
    [t('Prompt tokens'), formatNumber(report.summary.prompt_tokens)],
    [t('Completion tokens'), formatNumber(report.summary.completion_tokens)],
    [
      t('Customer invoice'),
      formatPrice(report.summary.total_price, report.summary.currency),
    ],
  ]
  return (
    <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
      {cards.map(([label, value]) => (
        <div key={label} className='bg-muted/30 rounded-lg border p-4'>
          <p className='text-muted-foreground text-xs'>{label}</p>
          <p className='mt-1 text-lg font-semibold tabular-nums'>{value}</p>
        </div>
      ))}
    </div>
  )
}

export function RawRowsTable({ report }: { report: EnterpriseBillingReport }) {
  const { t } = useTranslation()
  return (
    <Table className='min-w-[1120px]'>
      <TableHeader>
        <TableRow>
          <TableHead>{t('ID')}</TableHead>
          <TableHead>{t('Time')}</TableHead>
          <TableHead>{t('Type')}</TableHead>
          <TableHead>{t('Username')}</TableHead>
          <TableHead>{t('Token')}</TableHead>
          <TableHead>{t('Model')}</TableHead>
          <TableHead>{t('Quota')}</TableHead>
          <TableHead>{t('Prompt Tokens')}</TableHead>
          <TableHead>{t('Completion Tokens')}</TableHead>
          <TableHead>{t('Use Time')}</TableHead>
          <TableHead>{t('Channel ID')}</TableHead>
          <TableHead>{t('Channel Name')}</TableHead>
          <TableHead>{t('Group')}</TableHead>
          <TableHead>{t('Request ID')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {report.raw_rows.length === 0 ? (
          <TableRow>
            <TableCell
              colSpan={14}
              className='text-muted-foreground h-24 text-center'
            >
              {t('No usage records for this month')}
            </TableCell>
          </TableRow>
        ) : (
          report.raw_rows.map((row) => (
            <TableRow key={`${row.id}-${row.request_id}`}>
              <TableCell>{row.id}</TableCell>
              <TableCell>{row.time}</TableCell>
              <TableCell>{row.type}</TableCell>
              <TableCell>{row.username}</TableCell>
              <TableCell>{row.token}</TableCell>
              <TableCell>{row.model}</TableCell>
              <TableCell>{formatNumber(row.quota)}</TableCell>
              <TableCell>{formatNumber(row.prompt_tokens)}</TableCell>
              <TableCell>{formatNumber(row.completion_tokens)}</TableCell>
              <TableCell>{row.use_time}s</TableCell>
              <TableCell>{row.channel_id}</TableCell>
              <TableCell>{row.channel_name || '-'}</TableCell>
              <TableCell>{row.group || '-'}</TableCell>
              <TableCell className='max-w-56 truncate' title={row.request_id}>
                {row.request_id || '-'}
              </TableCell>
            </TableRow>
          ))
        )}
      </TableBody>
    </Table>
  )
}

export function CustomerRowsTable({
  report,
}: {
  report: EnterpriseBillingReport
}) {
  const { t } = useTranslation()
  return (
    <Table className='min-w-[760px]'>
      <TableHeader>
        <TableRow>
          <TableHead>{t('Order ID')}</TableHead>
          <TableHead>{t('Time')}</TableHead>
          <TableHead>{t('Type')}</TableHead>
          <TableHead>{t('Model name')}</TableHead>
          <TableHead>{t('Use Time')}</TableHead>
          <TableHead>{t('Request ID')}</TableHead>
          <TableHead className='text-right'>{t('Price')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {report.customer_rows.length === 0 ? (
          <TableRow>
            <TableCell
              colSpan={7}
              className='text-muted-foreground h-24 text-center'
            >
              {t('No invoice records for this month')}
            </TableCell>
          </TableRow>
        ) : (
          report.customer_rows.map((row) => (
            <TableRow key={`${row.order_id}-${row.request_id}`}>
              <TableCell>{row.order_id}</TableCell>
              <TableCell>{row.time}</TableCell>
              <TableCell>
                <Badge variant='outline'>{row.model_type}</Badge>
              </TableCell>
              <TableCell>{row.model_name}</TableCell>
              <TableCell>{row.use_time}s</TableCell>
              <TableCell className='max-w-56 truncate' title={row.request_id}>
                {row.request_id || '-'}
              </TableCell>
              <TableCell className='text-right font-medium tabular-nums'>
                {formatPrice(row.price, report.summary.currency)}
              </TableCell>
            </TableRow>
          ))
        )}
      </TableBody>
    </Table>
  )
}

export function EnterpriseBilling() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedAccountId, setSelectedAccountId] = useState(0)
  const [month, setMonth] = useState(currentMonth)
  const [page, setPage] = useState(1)
  const [view, setView] = useState<'customer' | 'raw'>('customer')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingAccount, setEditingAccount] =
    useState<EnterpriseBillingAccount | null>(null)

  const accountsQuery = useQuery({
    queryKey: ['enterprise-billing-accounts'],
    queryFn: getEnterpriseBillingAccounts,
  })
  const accounts = accountsQuery.data?.data ?? EMPTY_ACCOUNTS
  const selectedAccount = accounts.find(
    (account) => account.id === selectedAccountId
  )
  const firstAccountId = accounts[0]?.id ?? 0
  useEffect(() => {
    if (selectedAccountId === 0 && firstAccountId > 0) {
      setSelectedAccountId(firstAccountId)
    }
  }, [firstAccountId, selectedAccountId])

  const reportQuery = useQuery({
    queryKey: ['enterprise-billing-report', selectedAccountId, month, page],
    queryFn: () =>
      getEnterpriseBillingReport({
        account_id: selectedAccountId,
        month,
        p: page,
        page_size: PAGE_SIZE,
      }),
    enabled:
      selectedAccountId > 0 &&
      month.length > 0 &&
      Boolean(selectedAccount?.source_configured),
  })
  const report = reportQuery.data?.data
  const totalPages = Math.max(1, Math.ceil((report?.total || 0) / PAGE_SIZE))

  const saveMutation = useMutation({
    mutationFn: (payload: EnterpriseBillingAccountPayload) =>
      editingAccount
        ? updateEnterpriseBillingAccount(editingAccount.id, payload)
        : createEnterpriseBillingAccount(payload),
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to save enterprise account'))
        return
      }
      toast.success(t('Enterprise account saved'))
      setDialogOpen(false)
      setEditingAccount(null)
      queryClient.invalidateQueries({
        queryKey: ['enterprise-billing-accounts'],
      })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const deleteMutation = useMutation({
    mutationFn: deleteEnterpriseBillingAccount,
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to delete enterprise account'))
        return
      }
      toast.success(t('Enterprise account deleted'))
      setSelectedAccountId(0)
      queryClient.invalidateQueries({
        queryKey: ['enterprise-billing-accounts'],
      })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  useEffect(() => {
    setPage(1)
  }, [selectedAccountId, month])

  const exportRows = async (kind: 'raw' | 'customer') => {
    if (!selectedAccountId) return
    try {
      const result = await exportEnterpriseBillingCsv(
        { account_id: selectedAccountId, month },
        kind
      )
      downloadBlob(result.blob, result.filename)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('Export failed'))
    }
  }

  let reportContent: ReactNode = null
  if (!selectedAccount) {
    reportContent = (
      <div className='text-muted-foreground rounded-lg border border-dashed p-10 text-center text-sm'>
        <Database className='mx-auto mb-2 size-5' />
        {t('Select an enterprise account to view its report.')}
      </div>
    )
  } else if (selectedAccount.source_configured && reportQuery.isLoading) {
    reportContent = (
      <div className='text-muted-foreground p-10 text-center text-sm'>
        {t('Loading report...')}
      </div>
    )
  } else if (
    selectedAccount.source_configured &&
    (reportQuery.isError || reportQuery.data?.success === false)
  ) {
    reportContent = (
      <Alert variant='destructive'>
        <AlertCircle />
        <AlertTitle>{t('Failed to load report')}</AlertTitle>
        <AlertDescription>
          {reportQuery.data?.message ||
            (reportQuery.error instanceof Error
              ? reportQuery.error.message
              : '') ||
            t('Try again later.')}
        </AlertDescription>
      </Alert>
    )
  } else if (report) {
    reportContent = (
      <>
        <SummaryCards report={report} />
        <div className='flex flex-wrap items-center justify-between gap-3 border-b pb-3'>
          <div className='flex gap-1 rounded-lg border p-1'>
            <Button
              variant={view === 'customer' ? 'secondary' : 'ghost'}
              size='sm'
              aria-pressed={view === 'customer'}
              onClick={() => setView('customer')}
            >
              {t('Customer invoice')}
            </Button>
            <Button
              variant={view === 'raw' ? 'secondary' : 'ghost'}
              size='sm'
              aria-pressed={view === 'raw'}
              onClick={() => setView('raw')}
            >
              {t('Raw usage')}
            </Button>
          </div>
          <div className='flex flex-wrap gap-2'>
            <Button
              variant='outline'
              size='sm'
              onClick={() => exportRows('customer')}
            >
              <Download data-icon='inline-start' />
              {t('Export invoice')}
            </Button>
            <Button
              variant='outline'
              size='sm'
              onClick={() => exportRows('raw')}
            >
              <Download data-icon='inline-start' />
              {t('Export raw data')}
            </Button>
          </div>
        </div>
        <div className='min-w-0 overflow-x-auto rounded-lg border'>
          {view === 'customer' ? (
            <CustomerRowsTable report={report} />
          ) : (
            <RawRowsTable report={report} />
          )}
        </div>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <p className='text-muted-foreground text-sm'>
            {t('Showing page {{page}} of {{total}} ({{count}} records)', {
              page,
              total: totalPages,
              count: report.total,
            })}
          </p>
          <div className='flex gap-2'>
            <Button
              variant='outline'
              size='sm'
              disabled={page <= 1 || reportQuery.isFetching}
              onClick={() => setPage((current) => Math.max(1, current - 1))}
            >
              <ChevronLeft data-icon='inline-start' />
              {t('Previous')}
            </Button>
            <Button
              variant='outline'
              size='sm'
              disabled={page >= totalPages || reportQuery.isFetching}
              onClick={() => setPage((current) => current + 1)}
            >
              {t('Next')}
              <ChevronRight data-icon='inline-end' />
            </Button>
          </div>
        </div>
      </>
    )
  }

  let accountsContent: ReactNode
  if (accountsQuery.isLoading) {
    accountsContent = (
      <div className='text-muted-foreground p-8 text-center text-sm'>
        {t('Loading...')}
      </div>
    )
  } else if (accountsQuery.isError || accountsQuery.data?.success === false) {
    accountsContent = (
      <Alert variant='destructive'>
        <AlertCircle />
        <AlertTitle>{t('Failed to load')}</AlertTitle>
        <AlertDescription className='flex flex-wrap items-center gap-2'>
          <span>
            {accountsQuery.data?.message ||
              (accountsQuery.error instanceof Error
                ? accountsQuery.error.message
                : '') ||
              t('Try again later.')}
          </span>
          <Button
            variant='outline'
            size='sm'
            onClick={() => accountsQuery.refetch()}
          >
            <RefreshCw data-icon='inline-start' />
            {t('Retry')}
          </Button>
        </AlertDescription>
      </Alert>
    )
  } else if (accounts.length === 0) {
    accountsContent = (
      <div className='text-muted-foreground rounded-lg border border-dashed p-8 text-center text-sm'>
        {t('No enterprise accounts configured')}
      </div>
    )
  } else {
    accountsContent = (
      <Table className='min-w-[780px]'>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Company')}</TableHead>
            <TableHead>{t('Code')}</TableHead>
            <TableHead>{t('Source')}</TableHead>
            <TableHead>{t('Usage accounts')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead className='text-right'>{t('Actions')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {accounts.map((account) => (
            <TableRow
              key={account.id}
              data-state={
                selectedAccountId === account.id ? 'selected' : undefined
              }
              className='cursor-pointer'
              onClick={() => setSelectedAccountId(account.id)}
            >
              <TableCell className='font-medium'>{account.name}</TableCell>
              <TableCell className='font-mono text-xs'>
                {account.code}
              </TableCell>
              <TableCell>{account.source}</TableCell>
              <TableCell>{account.usernames.length}</TableCell>
              <TableCell>
                <Badge variant={account.enabled ? 'default' : 'secondary'}>
                  {account.enabled ? t('Enabled') : t('Disabled')}
                </Badge>
              </TableCell>
              <TableCell className='text-right'>
                <div className='flex justify-end gap-1'>
                  <Button
                    variant='ghost'
                    size='icon-sm'
                    aria-label={t('Edit')}
                    onClick={(event) => {
                      event.stopPropagation()
                      setEditingAccount(account)
                      setDialogOpen(true)
                    }}
                  >
                    <Pencil data-icon='inline-start' />
                  </Button>
                  <Button
                    variant='ghost'
                    size='icon-sm'
                    aria-label={t('Delete')}
                    onClick={(event) => {
                      event.stopPropagation()
                      if (
                        window.confirm(t('Delete this enterprise account?'))
                      ) {
                        deleteMutation.mutate(account.id)
                      }
                    }}
                  >
                    <Trash2 data-icon='inline-start' />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    )
  }

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Enterprise Billing')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t(
            'Group usage accounts, calculate monthly invoices, and export billing data.'
          )}
        </SectionPageLayout.Description>
        <SectionPageLayout.Actions>
          <Button
            variant='outline'
            size='sm'
            onClick={() => accountsQuery.refetch()}
            disabled={accountsQuery.isFetching}
          >
            <RefreshCw
              data-icon='inline-start'
              className={accountsQuery.isFetching ? 'animate-spin' : ''}
            />
            {t('Refresh')}
          </Button>
          <Button
            size='sm'
            onClick={() => {
              setEditingAccount(null)
              setDialogOpen(true)
            }}
          >
            <Plus data-icon='inline-start' />
            {t('Add account')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='grid min-w-0 gap-6'>
            <Card>
              <CardHeader>
                <CardTitle className='flex items-center gap-2 text-base'>
                  <Building2 className='text-primary size-4' />
                  {t('Enterprise accounts')}
                </CardTitle>
              </CardHeader>
              <CardContent className='grid min-w-0 gap-4'>
                {accountsContent}
              </CardContent>
            </Card>

            <Card>
              <CardHeader className='gap-4 sm:flex-row sm:items-center sm:justify-between'>
                <div>
                  <CardTitle className='flex items-center gap-2 text-base'>
                    <FileSpreadsheet className='text-primary size-4' />
                    {t('Monthly billing report')}
                  </CardTitle>
                  <p className='text-muted-foreground mt-1 text-sm'>
                    {t(
                      'Choose an account and month to load usage and customer invoice rows.'
                    )}
                  </p>
                </div>
                <div className='flex flex-wrap gap-2'>
                  <NativeSelect
                    aria-label={t('Enterprise account')}
                    value={selectedAccountId || ''}
                    onChange={(event) =>
                      setSelectedAccountId(Number(event.target.value))
                    }
                    className='min-w-44'
                    disabled={accounts.length === 0}
                  >
                    <NativeSelectOption value=''>
                      {t('Select an account')}
                    </NativeSelectOption>
                    {accounts.map((account) => (
                      <NativeSelectOption key={account.id} value={account.id}>
                        {account.name}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                  <Input
                    aria-label={t('Billing month')}
                    type='month'
                    value={month}
                    onChange={(event) => setMonth(event.target.value)}
                    className='w-36'
                  />
                </div>
              </CardHeader>
              <CardContent className='grid min-w-0 gap-4'>
                {selectedAccount && !selectedAccount.source_configured ? (
                  <Alert variant='destructive'>
                    <AlertCircle />
                    <AlertTitle>
                      {t('Usage source is not configured')}
                    </AlertTitle>
                    <AlertDescription>
                      {t(
                        'Configure the source DSN on the server before loading this account.'
                      )}
                    </AlertDescription>
                  </Alert>
                ) : null}
                {reportContent}
              </CardContent>
            </Card>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>
      <AccountFormDialog
        open={dialogOpen}
        account={editingAccount}
        onOpenChange={(open) => {
          setDialogOpen(open)
          if (!open) setEditingAccount(null)
        }}
        onSubmit={(payload) => saveMutation.mutate(payload)}
        pending={saveMutation.isPending}
      />
    </>
  )
}
