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
import { Ban, Eye, Plus, RotateCcw, Tags, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { MultiSelect } from '@/components/multi-select'
import { DataTableRowActionMenu, StaticDataTable } from '@/components/data-table'
import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { StatusBadge } from '@/components/status-badge'
import { TableId } from '@/components/table-id'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
} from '@/components/ui/dropdown-menu'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet'
import { Switch } from '@/components/ui/switch'
import { formatQuota } from '@/lib/format'

import {
  getAdminPlans,
  getUserSubscriptions,
  createUserSubscription,
  previewSubscriptionUsage,
  invalidateUserSubscription,
  deleteUserSubscription,
  resetUserSubscriptionsByPlan,
  updateUserSubscriptionGroups,
  getGroups,
} from '../../api'
import { formatTimestamp } from '../../lib'
import type {
  PlanRecord,
  SubscriptionUsagePreview,
  UserSubscriptionRecord,
} from '../../types'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: { id: number; username?: string } | null
  onSuccess?: () => void
}

function SubscriptionStatusBadge(props: {
  sub: UserSubscriptionRecord['subscription']
  t: (key: string) => string
}) {
  // eslint-disable-next-line react-hooks/purity
  const now = Date.now() / 1000
  const isExpired = (props.sub.end_time || 0) > 0 && props.sub.end_time < now
  const isActive = props.sub.status === 'active' && !isExpired
  if (isActive) {
    return (
      <StatusBadge
        label={props.t('Active')}
        variant='success'
        copyable={false}
      />
    )
  }
  if (props.sub.status === 'cancelled') {
    return (
      <StatusBadge
        label={props.t('Invalidated')}
        variant='neutral'
        copyable={false}
      />
    )
  }
  return (
    <StatusBadge
      label={props.t('Expired')}
      variant='neutral'
      copyable={false}
    />
  )
}

export function UserSubscriptionsDialog(props: Props) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [plans, setPlans] = useState<PlanRecord[]>([])
  const [subs, setSubs] = useState<UserSubscriptionRecord[]>([])
  const [groupOptions, setGroupOptions] = useState<string[]>([])
  const [selectedPlanId, setSelectedPlanId] = useState<string>('')
  const [effectiveStart, setEffectiveStart] = useState('')
  const [importUsage, setImportUsage] = useState(false)
  const [clearWallet, setClearWallet] = useState(false)
  const [previewing, setPreviewing] = useState(false)
  const [preview, setPreview] = useState<SubscriptionUsagePreview | null>(null)
  const [confirmOverLimit, setConfirmOverLimit] = useState(false)
  const [resetting, setResetting] = useState(false)
  const [advanceResetTime, setAdvanceResetTime] = useState(true)
  const [resetAction, setResetAction] = useState<{
    planId: number
    planTitle: string
  } | null>(null)
  const [confirmAction, setConfirmAction] = useState<{
    type: 'invalidate' | 'delete'
    subId: number
  } | null>(null)
  const [groupAction, setGroupAction] = useState<{
    subId: number
    groups: string[]
  } | null>(null)

  const planTitleMap = useMemo(() => {
    const map = new Map<number, string>()
    plans.forEach((p) => {
      if (p.plan.id) map.set(p.plan.id, p.plan.title || `#${p.plan.id}`)
    })
    return map
  }, [plans])

  const loadData = useCallback(async () => {
    if (!props.user?.id) return
    setLoading(true)
    try {
      const [plansRes, subsRes, groupsRes] = await Promise.all([
        getAdminPlans(),
        getUserSubscriptions(props.user.id),
        getGroups(),
      ])
      if (plansRes.success) setPlans(plansRes.data || [])
      if (subsRes.success) setSubs(subsRes.data || [])
      if (groupsRes.success) setGroupOptions(groupsRes.data || [])
    } catch {
      toast.error(t('Loading failed'))
    } finally {
      setLoading(false)
    }
  }, [props.user?.id, t])

  useEffect(() => {
    if (props.open && props.user?.id) {
      setSelectedPlanId('')
      const now = new Date()
      setEffectiveStart(
        new Date(now.getTime() - now.getTimezoneOffset() * 60_000)
          .toISOString()
          .slice(0, 16)
      )
      setImportUsage(false)
      setClearWallet(false)
      setPreview(null)
      loadData()
    }
  }, [props.open, props.user?.id, loadData])

  const effectiveStartTime = Math.floor(
    new Date(effectiveStart).getTime() / 1000
  )

  const handlePreview = async () => {
    if (!props.user?.id || !selectedPlanId || !effectiveStartTime) return
    setPreviewing(true)
    try {
      const res = await previewSubscriptionUsage({
        user_id: props.user.id,
        plan_id: Number(selectedPlanId),
        effective_start_time: effectiveStartTime,
      })
      if (res.success && res.data) {
        setPreview(res.data)
        setConfirmOverLimit(false)
      } else {
        toast.error(res.message || t('Preview failed'))
      }
    } catch {
      toast.error(t('Preview failed'))
    } finally {
      setPreviewing(false)
    }
  }

  const handleCreate = async () => {
    if (!props.user?.id || !selectedPlanId) {
      toast.error(t('Please select a subscription plan'))
      return
    }
    if (importUsage && !preview) {
      await handlePreview()
      return
    }
    if (importUsage && preview?.exceeds_any_limit && !confirmOverLimit) {
      setConfirmOverLimit(true)
      return
    }
    setCreating(true)
    try {
      const res = await createUserSubscription(props.user.id, {
        plan_id: Number(selectedPlanId),
        effective_start_time: effectiveStartTime,
        import_usage: importUsage,
        clear_wallet: clearWallet,
        confirm_over_limit: confirmOverLimit,
      })
      if (res.success) {
        toast.success(res.data?.message || t('Added successfully'))
        setSelectedPlanId('')
        setPreview(null)
        await loadData()
        props.onSuccess?.()
      }
    } catch {
      toast.error(t('Request failed'))
    } finally {
      setCreating(false)
    }
  }

  const handleConfirmAction = async () => {
    if (!confirmAction) return
    try {
      if (confirmAction.type === 'invalidate') {
        const res = await invalidateUserSubscription(confirmAction.subId)
        if (res.success) {
          toast.success(res.data?.message || t('Has been invalidated'))
          await loadData()
          props.onSuccess?.()
        }
      } else {
        const res = await deleteUserSubscription(confirmAction.subId)
        if (res.success) {
          toast.success(t('Deleted'))
          await loadData()
          props.onSuccess?.()
        }
      }
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setConfirmAction(null)
    }
  }

  const handleResetConfirm = async () => {
    if (!props.user?.id || !resetAction) return
    setResetting(true)
    try {
      const res = await resetUserSubscriptionsByPlan(props.user.id, {
        plan_id: resetAction.planId,
        advance_reset_time: advanceResetTime,
      })
      if (res.success) {
        toast.success(
          t('Reset {{count}} active subscriptions', {
            count: res.data?.reset_count || 0,
          })
        )
        await loadData()
        props.onSuccess?.()
      }
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setResetting(false)
      setResetAction(null)
    }
  }

  const handleGroupsConfirm = async () => {
    if (!groupAction) return
    try {
      const res = await updateUserSubscriptionGroups(
        groupAction.subId,
        groupAction.groups
      )
      if (res.success) {
        toast.success(t('Subscription groups updated'))
        await loadData()
      } else {
        toast.error(res.message || t('Operation failed'))
      }
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setGroupAction(null)
    }
  }

  return (
    <>
      <Sheet open={props.open} onOpenChange={props.onOpenChange}>
        <SheetContent className={sideDrawerContentClassName('sm:max-w-2xl')}>
          <SheetHeader className={sideDrawerHeaderClassName()}>
            <SheetTitle>{t('User Subscription Management')}</SheetTitle>
            <SheetDescription>
              {props.user?.username || '-'} (ID: {props.user?.id || '-'})
            </SheetDescription>
          </SheetHeader>

          <div className={sideDrawerFormClassName()}>
            <div className='grid gap-3 border-b pb-4'>
              <div className='flex gap-2'>
              <Select
                items={plans.map((p) => ({
                  value: String(p.plan.id),
                  label: (
                    <>
                      {p.plan.title}($
                      {Number(p.plan.price_amount || 0).toFixed(2)})
                    </>
                  ),
                }))}
                value={selectedPlanId}
                onValueChange={(v) => v !== null && setSelectedPlanId(v)}
              >
                <SelectTrigger className='flex-1'>
                  <SelectValue placeholder={t('Select subscription plan')} />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {plans.map((p) => (
                      <SelectItem key={p.plan.id} value={String(p.plan.id)}>
                        {p.plan.title} ($
                        {Number(p.plan.price_amount || 0).toFixed(2)})
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              <Button
                onClick={handleCreate}
                disabled={creating || !selectedPlanId}
              >
                <Plus className='mr-1 h-4 w-4' />
                {t('Add subscription')}
              </Button>
              </div>

              <div className='grid gap-3 sm:grid-cols-2'>
                <label className='grid gap-1.5 text-sm'>
                  <span className='font-medium'>{t('Effective start time')}</span>
                  <Input
                    type='datetime-local'
                    value={effectiveStart}
                    max={new Date().toISOString().slice(0, 16)}
                    onChange={(event) => {
                      setEffectiveStart(event.target.value)
                      setPreview(null)
                    }}
                  />
                </label>
                <div className='grid gap-2'>
                  <label className='flex items-center justify-between gap-3 text-sm'>
                    <span>{t('Import historical usage')}</span>
                    <Switch
                      checked={importUsage}
                      onCheckedChange={(checked) => {
                        setImportUsage(!!checked)
                        setPreview(null)
                      }}
                    />
                  </label>
                  <label className='flex items-center justify-between gap-3 text-sm'>
                    <span>{t('Clear wallet balance after binding')}</span>
                    <Switch
                      checked={clearWallet}
                      onCheckedChange={(checked) => setClearWallet(!!checked)}
                    />
                  </label>
                </div>
              </div>

              {importUsage && (
                <div className='flex items-center justify-between gap-3'>
                  <p className='text-muted-foreground text-xs'>
                    {t('Usage is calculated from consume logs in the selected plan groups')}
                  </p>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    disabled={previewing || !selectedPlanId || !effectiveStartTime}
                    onClick={handlePreview}
                  >
                    <Eye className='mr-1 h-4 w-4' />
                    {t('Preview usage')}
                  </Button>
                </div>
              )}

              {importUsage && preview && (
                <div className='grid grid-cols-2 gap-x-4 gap-y-2 rounded-md border px-3 py-2 text-sm sm:grid-cols-4'>
                  {[
                    [t('Total'), preview.amount_used, preview.amount_total, preview.exceeds_amount_total],
                    [t('Daily'), preview.daily_used, preview.daily_amount, preview.exceeds_daily_amount],
                    [t('Weekly'), preview.weekly_used, preview.weekly_amount, preview.exceeds_weekly_amount],
                    [t('Monthly'), preview.monthly_used, preview.monthly_amount, preview.exceeds_monthly_amount],
                  ].map(([label, used, limit, exceeded]) => (
                    <div key={String(label)}>
                      <div className='text-muted-foreground text-xs'>{label}</div>
                      <div className={exceeded ? 'text-destructive font-medium' : 'font-medium'}>
                        {formatQuota(Number(used))} / {Number(limit) > 0 ? formatQuota(Number(limit)) : t('Unlimited')}
                      </div>
                    </div>
                  ))}
                  {!preview.consume_log_enabled && (
                    <p className='text-destructive col-span-full text-xs'>
                      {t('Consume logging is disabled; imported usage may be incomplete')}
                    </p>
                  )}
                </div>
              )}
            </div>

            <StaticDataTable
              data={loading ? [] : subs}
              getRowKey={(record) => record.subscription.id}
              emptyClassName={loading ? 'py-8' : 'text-muted-foreground py-8'}
              emptyContent={
                loading ? t('Loading...') : t('No subscription records')
              }
              columns={[
                {
                  id: 'id',
                  header: t('ID'),
                  cell: (record) => <TableId value={record.subscription.id} />,
                },
                {
                  id: 'plan',
                  header: t('Plan'),
                  cell: (record) => {
                    const sub = record.subscription

                    return (
                      <div>
                        <div className='font-medium'>
                          {planTitleMap.get(sub.plan_id) || `#${sub.plan_id}`}
                        </div>
                        <div className='text-muted-foreground text-sm'>
                          {t('Source')}: {sub.source || '-'}
                        </div>
                      </div>
                    )
                  },
                },
                {
                  id: 'status',
                  header: t('Status'),
                  cell: (record) => (
                    <SubscriptionStatusBadge sub={record.subscription} t={t} />
                  ),
                },
                {
                  id: 'validity',
                  header: t('Validity'),
                  cell: (record) => {
                    const sub = record.subscription

                    return (
                      <div className='text-sm'>
                        <div>
                          {t('Start')}: {formatTimestamp(sub.start_time)}
                        </div>
                        <div>
                          {t('End')}: {formatTimestamp(sub.end_time)}
                        </div>
                      </div>
                    )
                  },
                },
                {
                  id: 'quota',
                  header: t('Total Quota'),
                  cell: (record) => {
                    const sub = record.subscription
                    const total = Number(sub.amount_total || 0)
                    const used = Number(sub.amount_used || 0)
                    return total > 0
                      ? `${formatQuota(used)}/${formatQuota(total)}`
                      : t('Unlimited')
                  },
                },
                {
                  id: 'groups',
                  header: t('API key groups'),
                  cell: (record) => {
                    const groups = record.subscription.applicable_groups || []
                    return (
                      <span className='text-muted-foreground text-sm'>
                        {groups.length > 0 ? groups.join(', ') : t('All groups')}
                      </span>
                    )
                  },
                },
                {
                  id: 'actions',
                  header: t('Actions'),
                  className: 'text-right',
                  cellClassName: 'text-right',
                  cell: (record) => {
                    const sub = record.subscription
                    const now = Date.now() / 1000
                    const isExpired =
                      (sub.end_time || 0) > 0 && sub.end_time < now
                    const isActive = sub.status === 'active' && !isExpired

                    return (
                      <DataTableRowActionMenu ariaLabel={t('Actions')}>
                        <DropdownMenuItem
                          onSelect={(event) => {
                            event.preventDefault()
                            setGroupAction({
                              subId: sub.id,
                              groups: sub.applicable_groups || [],
                            })
                          }}
                        >
                          {t('Edit API key groups')}
                          <DropdownMenuShortcut>
                            <Tags size={16} />
                          </DropdownMenuShortcut>
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          disabled={!isActive}
                          onClick={() => {
                            setAdvanceResetTime(true)
                            setResetAction({
                              planId: sub.plan_id,
                              planTitle:
                                planTitleMap.get(sub.plan_id) ||
                                `#${sub.plan_id}`,
                            })
                          }}
                        >
                          {t('Reset quota')}
                          <DropdownMenuShortcut>
                            <RotateCcw size={16} />
                          </DropdownMenuShortcut>
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          disabled={!isActive}
                          onClick={() =>
                            setConfirmAction({
                              type: 'invalidate',
                              subId: sub.id,
                            })
                          }
                        >
                          {t('Invalidate')}
                          <DropdownMenuShortcut>
                            <Ban size={16} />
                          </DropdownMenuShortcut>
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          variant='destructive'
                          onClick={() =>
                            setConfirmAction({
                              type: 'delete',
                              subId: sub.id,
                            })
                          }
                        >
                          {t('Delete')}
                          <DropdownMenuShortcut>
                            <Trash2 size={16} />
                          </DropdownMenuShortcut>
                        </DropdownMenuItem>
                      </DataTableRowActionMenu>
                    )
                  },
                },
              ]}
            />
          </div>
        </SheetContent>
      </Sheet>

      {confirmAction && (
        <ConfirmDialog
          open
          onOpenChange={(v) => !v && setConfirmAction(null)}
          title={
            confirmAction.type === 'invalidate'
              ? t('Confirm invalidate')
              : t('Confirm delete')
          }
          desc={
            confirmAction.type === 'invalidate'
              ? t(
                  'After invalidating, this subscription will be immediately deactivated. Historical records are not affected. Continue?'
                )
              : t(
                  'Deleting will permanently remove this subscription record (including benefit details). Continue?'
                )
          }
          handleConfirm={handleConfirmAction}
          destructive={confirmAction.type === 'delete'}
        />
      )}

      {resetAction && (
        <ConfirmDialog
          open
          onOpenChange={(v) => !v && setResetAction(null)}
          title={t('Reset subscription quota')}
          desc={t('Reset active {{plan}} subscriptions for this user?', {
            plan: resetAction.planTitle,
          })}
          confirmText={t('Reset quota')}
          handleConfirm={handleResetConfirm}
          isLoading={resetting}
        >
          <label className='flex items-center justify-between gap-3 rounded-md border px-3 py-2 text-sm'>
            <span>{t('Advance next reset time')}</span>
            <Switch
              checked={advanceResetTime}
              onCheckedChange={(checked) => setAdvanceResetTime(!!checked)}
              aria-label={t('Advance next reset time')}
            />
          </label>
        </ConfirmDialog>
      )}

      {confirmOverLimit && preview?.exceeds_any_limit && (
        <ConfirmDialog
          open
          onOpenChange={(open) => !open && setConfirmOverLimit(false)}
          title={t('Imported usage exceeds plan limits')}
          desc={t('The subscription will be created with exhausted quota in the exceeded periods. Continue?')}
          confirmText={t('Create anyway')}
          destructive
          handleConfirm={handleCreate}
          isLoading={creating}
        />
      )}

      {groupAction && (
        <ConfirmDialog
          open
          onOpenChange={(open) => !open && setGroupAction(null)}
          title={t('Edit API key groups')}
          desc={t('Leave empty to allow this subscription in every API key group')}
          confirmText={t('Save changes')}
          handleConfirm={handleGroupsConfirm}
        >
          <MultiSelect
            options={[...new Set([...groupOptions, ...groupAction.groups])].map(
              (group) => ({
                value: group,
                label: groupOptions.includes(group)
                  ? group
                  : `${group} (${t('Deleted group')})`,
              })
            )}
            selected={groupAction.groups}
            onChange={(groups) =>
              setGroupAction((current) =>
                current ? { ...current, groups } : current
              )
            }
            placeholder={t('All API key groups')}
          />
        </ConfirmDialog>
      )}
    </>
  )
}
