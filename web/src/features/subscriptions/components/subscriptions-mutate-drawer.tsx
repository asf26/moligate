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
import { zodResolver } from '@hookform/resolvers/zod'
import {
  CalendarClock,
  CreditCard,
  Gift,
  ListChecks,
  Plus,
  RefreshCw,
  Settings2,
  Sparkles,
  Tag,
  Trash2,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { useFieldArray, useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  SideDrawerSection,
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
  sideDrawerSwitchItemClassName,
} from '@/components/drawer-layout'
import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { IconBadge } from '@/components/ui/icon-badge'
import { Input } from '@/components/ui/input'
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
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { getCurrencyDisplay, getCurrencyLabel } from '@/lib/currency'

import {
  createPlan,
  updatePlan,
  getGroups,
  createWaffoPancakeSubscriptionProduct,
  listWaffoPancakeSubscriptionProductOptions,
} from '../api'
import { getDurationUnitOptions, getResetPeriodOptions } from '../constants'
import {
  getPlanFormSchema,
  PLAN_FORM_DEFAULTS,
  planToFormValues,
  formValuesToPlanPayload,
  emptyBonusResource,
  type PlanFormValues,
} from '../lib'
import type { PlanRecord } from '../types'
import { useSubscriptions } from './subscriptions-provider'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: PlanRecord
}

export function SubscriptionsMutateDrawer({
  open,
  onOpenChange,
  currentRow,
}: Props) {
  const { t } = useTranslation()
  const isEdit = !!currentRow?.plan?.id
  const { triggerRefresh } = useSubscriptions()
  const { meta: currencyMeta } = getCurrencyDisplay()
  const tokensOnly = currencyMeta.kind === 'tokens'
  const currencyLabel = getCurrencyLabel()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [groupOptions, setGroupOptions] = useState<string[]>([])
  const [creatingPancakeProduct, setCreatingPancakeProduct] = useState(false)
  const [pancakeProducts, setPancakeProducts] = useState<
    { id: string; name: string; status: string }[]
  >([])

  const schema = getPlanFormSchema(t)
  const form = useForm<PlanFormValues>({
    resolver: zodResolver(schema) as unknown as Resolver<PlanFormValues>,
    defaultValues: PLAN_FORM_DEFAULTS,
  })
  const bonusResources = useFieldArray({
    control: form.control,
    name: 'bonus_resources',
  })
  const imagePackConfigured = bonusResources.fields.some(
    (resource) =>
      resource.resource_type === 'image_count' &&
      resource.resource_key?.trim().toLowerCase() === 'gpt-image-2'
  )

  useEffect(() => {
    if (open) {
      if (currentRow?.plan) {
        form.reset(planToFormValues(currentRow.plan))
      } else {
        form.reset(PLAN_FORM_DEFAULTS)
      }
      getGroups()
        .then((res) => {
          if (res.success) setGroupOptions(res.data || [])
        })
        .catch(() => {})
      // Best-effort — empty list still lets the operator use "+ Create".
      listWaffoPancakeSubscriptionProductOptions()
        .then((res) => {
          if (
            res.message === 'success' &&
            typeof res.data === 'object' &&
            res.data &&
            Array.isArray((res.data as { products?: unknown }).products)
          ) {
            setPancakeProducts(
              (res.data as { products: typeof pancakeProducts }).products
            )
          } else {
            setPancakeProducts([])
          }
        })
        .catch(() => setPancakeProducts([]))
    }
  }, [open, currentRow, form])

  const durationUnit = form.watch('duration_unit')
  const resetPeriod = form.watch('quota_reset_period')
  // Gate "+ Create on Pancake" on the same checks the mint handler runs.
  const watchedTitle = form.watch('title')
  const watchedPrice = form.watch('price_amount')
  const pancakeCreateReady =
    typeof watchedTitle === 'string' &&
    watchedTitle.trim().length > 0 &&
    Number(watchedPrice ?? 0) > 0

  const onSubmit = async (values: PlanFormValues) => {
    setIsSubmitting(true)
    try {
      const payload = formValuesToPlanPayload(values)
      if (isEdit && currentRow?.plan?.id) {
        const res = await updatePlan(currentRow.plan.id, payload)
        if (res.success) {
          toast.success(t('Update succeeded'))
          onOpenChange(false)
          triggerRefresh()
        }
      } else {
        const res = await createPlan(payload)
        if (res.success) {
          toast.success(t('Create succeeded'))
          onOpenChange(false)
          triggerRefresh()
        }
      }
    } catch {
      toast.error(t('Request failed'))
    } finally {
      setIsSubmitting(false)
    }
  }

  // Mints a Pancake OnetimeProduct (not SubscriptionProduct — see
  // controller) using persisted creds + the form's title/price, then
  // pins the returned PROD_ ID into the form field.
  const handleCreatePancakeProduct = async () => {
    const title = form.getValues('title').trim()
    const priceAmount = Number(form.getValues('price_amount') || 0)
    if (!title) {
      toast.error(t('Plan title is required'))
      return
    }
    if (priceAmount <= 0) {
      toast.error(t('Plan price must be greater than zero'))
      return
    }
    setCreatingPancakeProduct(true)
    try {
      const res = await createWaffoPancakeSubscriptionProduct({
        name: title,
        amount: priceAmount.toFixed(2),
      })
      if (
        res.message === 'success' &&
        typeof res.data === 'object' &&
        res.data
      ) {
        const created = res.data as { product_id: string; product_name: string }
        form.setValue('waffo_pancake_product_id', created.product_id, {
          shouldDirty: true,
        })
        // Refetch from GraphQL so the dropdown reflects authoritative state.
        try {
          const refresh = await listWaffoPancakeSubscriptionProductOptions()
          if (
            refresh.message === 'success' &&
            typeof refresh.data === 'object' &&
            refresh.data &&
            Array.isArray((refresh.data as { products?: unknown }).products)
          ) {
            setPancakeProducts(
              (refresh.data as { products: typeof pancakeProducts }).products
            )
          }
        } catch {
          // Best-effort — form value already points at the new product;
          // raw-ID fallback covers the missing label.
        }
        toast.success(
          `${t('Waffo Pancake product created')}: ${created.product_id}`
        )
      } else {
        const reason = typeof res.data === 'string' ? res.data : undefined
        toast.error(
          reason
            ? `${t('Waffo Pancake product creation failed')}: ${reason}`
            : t('Waffo Pancake product creation failed')
        )
      }
    } catch (err) {
      toast.error(
        `${t('Waffo Pancake product creation failed')}: ${err instanceof Error ? err.message : String(err)}`
      )
    } finally {
      setCreatingPancakeProduct(false)
    }
  }

  const durationUnitOpts = getDurationUnitOptions(t)
  const resetPeriodOpts = getResetPeriodOptions(t)

  return (
    <Sheet
      open={open}
      onOpenChange={(v) => {
        onOpenChange(v)
        if (!v) {
          form.reset()
        }
      }}
    >
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[600px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>
            {isEdit ? t('Update plan info') : t('Create new subscription plan')}
          </SheetTitle>
          <SheetDescription>
            {isEdit
              ? t('Modify existing subscription plan configuration')
              : t(
                  'Fill in the following info to create a new subscription plan'
                )}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form
            id='subscription-form'
            onSubmit={form.handleSubmit(onSubmit)}
            className={sideDrawerFormClassName()}
          >
            {/* Basic Info */}
            <SideDrawerSection>
              <h3 className='flex items-center gap-2 text-sm font-medium'>
                <IconBadge tone='info' size='xs'>
                  <Settings2 />
                </IconBadge>
                {t('Basic Info')}
              </h3>

              <FormField
                control={form.control}
                name='title'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Plan Title')}</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder={t('e.g. Basic Plan')} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='subtitle'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Plan Subtitle')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        placeholder={t('e.g. Suitable for light usage')}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='applicable_groups'
                  render={({ field }) => (
                    <FormItem className='sm:col-span-2'>
                      <FormLabel>{t('Applicable API key groups')}</FormLabel>
                      <FormControl>
                        <MultiSelect
                          options={[
                            ...new Set([
                              ...groupOptions,
                              ...(field.value || []),
                            ]),
                          ].map((group) => ({
                            label: groupOptions.includes(group)
                              ? group
                              : `${group} (${t('Deleted group')})`,
                            value: group,
                          }))}
                          selected={field.value || []}
                          onChange={field.onChange}
                          placeholder={t('All API key groups')}
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          'Leave empty to allow this subscription in every API key group'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='price_amount'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Plan Price')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type='number'
                          step='0.01'
                          min={0}
                          max={100}
                          onChange={(e) =>
                            field.onChange(
                              Number.parseFloat(e.target.value) || 0
                            )
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          'Amount the user pays to purchase this plan; the actual currency depends on the payment gateway.'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='billing_ratio'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Subscription usage rate')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type='number'
                          step='0.01'
                          min={0}
                          max={100}
                          onChange={(event) =>
                            field.onChange(
                              Number.parseFloat(event.target.value) || 0
                            )
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          'Fixed multiplier used by purchased subscriptions; 0 uses the family default.'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='total_amount'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('Quota ({{currency}})', { currency: currencyLabel })}
                      </FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type='number'
                          min={0}
                          step={tokensOnly ? 1 : 0.01}
                          placeholder={
                            tokensOnly
                              ? t('Enter quota in tokens')
                              : t('Enter quota in {{currency}}', {
                                  currency: currencyLabel,
                                })
                          }
                          onChange={(e) =>
                            field.onChange(
                              Number.parseFloat(e.target.value) || 0
                            )
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          'Total quota across the subscription lifetime. It only resets when the total quota reset below is configured. 0 means unlimited.'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              <div className='grid grid-cols-1 gap-3 sm:grid-cols-3'>
                {(
                  [
                    ['daily_amount', 'Daily Quota'],
                    ['weekly_amount', 'Weekly Quota'],
                    ['monthly_amount', 'Monthly Quota'],
                  ] as const
                ).map(([name, label]) => (
                  <FormField
                    key={name}
                    control={form.control}
                    name={name}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t(label)}</FormLabel>
                        <FormControl>
                          <Input
                            {...field}
                            type='number'
                            min={0}
                            step={tokensOnly ? 1 : 0.01}
                            onChange={(event) =>
                              field.onChange(
                                Number.parseFloat(event.target.value) || 0
                              )
                            }
                          />
                        </FormControl>
                        <FormDescription>
                          {t('0 means unlimited')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                ))}
              </div>

              <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='upgrade_group'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Upgrade Group')}</FormLabel>
                      <Select
                        items={[
                          { value: '__none__', label: t('No Upgrade') },
                          ...groupOptions.map((g) => ({ value: g, label: g })),
                        ]}
                        onValueChange={(v) =>
                          field.onChange(v === '__none__' ? '' : v)
                        }
                        value={field.value || ''}
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder={t('No Upgrade')} />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent alignItemWithTrigger={false}>
                          <SelectGroup>
                            <SelectItem value='__none__'>
                              {t('No Upgrade')}
                            </SelectItem>
                            {groupOptions.map((g) => (
                              <SelectItem key={g} value={g}>
                                {g}
                              </SelectItem>
                            ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='downgrade_group'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Downgrade Group')}</FormLabel>
                      <Select
                        items={[
                          {
                            value: '__none__',
                            label: t('Downgrade to pre-purchase group'),
                          },
                          ...groupOptions.map((g) => ({ value: g, label: g })),
                        ]}
                        onValueChange={(v) =>
                          field.onChange(v === '__none__' ? '' : v)
                        }
                        value={field.value || ''}
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue
                              placeholder={t('Downgrade to pre-purchase group')}
                            />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent alignItemWithTrigger={false}>
                          <SelectGroup>
                            <SelectItem value='__none__'>
                              {t('Downgrade to pre-purchase group')}
                            </SelectItem>
                            {groupOptions.map((g) => (
                              <SelectItem key={g} value={g}>
                                {g}
                              </SelectItem>
                            ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <FormDescription>
                        {t(
                          'Downgrade to this group after the subscription expires'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='max_purchase_per_user'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Purchase Limit')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type='number'
                          min={0}
                          onChange={(e) =>
                            field.onChange(
                              Number.parseInt(e.target.value, 10) || 0
                            )
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        {t('0 means unlimited')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              <FormField
                control={form.control}
                name='sort_order'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Sort Order')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        type='number'
                        onChange={(e) =>
                          field.onChange(
                            Number.parseInt(e.target.value, 10) || 0
                          )
                        }
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <div className='flex flex-col gap-3'>
                <FormField
                  control={form.control}
                  name='enabled'
                  render={({ field }) => (
                    <FormItem className={sideDrawerSwitchItemClassName()}>
                      <FormLabel className='!mt-0'>
                        {t('Enabled Status')}
                      </FormLabel>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='allow_balance_pay'
                  render={({ field }) => (
                    <FormItem className={sideDrawerSwitchItemClassName()}>
                      <FormLabel className='!mt-0'>
                        {t('Allow balance redemption')}
                      </FormLabel>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='allow_wallet_overflow'
                  render={({ field }) => (
                    <FormItem className={sideDrawerSwitchItemClassName()}>
                      <FormLabel className='!mt-0'>
                        {t('Allow wallet balance after quota used up')}
                      </FormLabel>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
              </div>
            </SideDrawerSection>

            {/* Model Package Settings */}
            <SideDrawerSection>
              <h3 className='flex items-center gap-2 text-sm font-medium'>
                <IconBadge tone='chart-2' size='xs'>
                  <Sparkles />
                </IconBadge>
                {t('Model package settings')}
              </h3>

              <FormField
                control={form.control}
                name='model_family'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Model family')}</FormLabel>
                    <Select
                      items={[
                        {
                          value: '__auto__',
                          label: t('Automatically infer from plan metadata'),
                        },
                        { value: 'ccmax', label: t('CC Max') },
                        { value: 'kiro-claude', label: t('Kiro Claude') },
                        { value: 'gpt', label: 'GPT' },
                        { value: 'gemini', label: 'Gemini' },
                        { value: 'chinese', label: t('Chinese models') },
                        { value: 'gpt-image', label: t('GPT Image') },
                        { value: 'banana', label: t('Nano Banana') },
                      ]}
                      value={field.value || '__auto__'}
                      onValueChange={(value) =>
                        field.onChange(value === '__auto__' ? '' : value)
                      }
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          <SelectItem value='__auto__'>
                            {t('Automatically infer from plan metadata')}
                          </SelectItem>
                          <SelectItem value='ccmax'>{t('CC Max')}</SelectItem>
                          <SelectItem value='kiro-claude'>
                            {t('Kiro Claude')}
                          </SelectItem>
                          <SelectItem value='gpt'>GPT</SelectItem>
                          <SelectItem value='gemini'>Gemini</SelectItem>
                          <SelectItem value='chinese'>
                            {t('Chinese models')}
                          </SelectItem>
                          <SelectItem value='gpt-image'>
                            {t('GPT Image')}
                          </SelectItem>
                          <SelectItem value='banana'>
                            {t('Nano Banana')}
                          </SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      {t(
                        'Choose the model series shown on the subscription page.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='included_models'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Included models')}</FormLabel>
                    <FormControl>
                      <Textarea
                        value={(field.value || []).join('\n')}
                        onChange={(event) =>
                          field.onChange([
                            ...new Set(
                              event.target.value
                                .split(/[\n,]+/)
                                .map((model) => model.trim())
                                .filter(Boolean)
                            ),
                          ])
                        }
                        placeholder={t(
                          'One model per line. Leave empty to use the family catalog.'
                        )}
                        rows={4}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Only these model names are included in this tier. Leave empty to use the selected family catalog.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SideDrawerSection>

            {/* Bonus resources */}
            <SideDrawerSection>
              <div className='flex items-start justify-between gap-3'>
                <div>
                  <h3 className='flex items-center gap-2 text-sm font-medium'>
                    <IconBadge tone='success' size='xs'>
                      <Gift />
                    </IconBadge>
                    {t('Bonus resources')}
                  </h3>
                  <p className='text-muted-foreground mt-1 text-xs leading-5'>
                    {t(
                      'Gift model quota or a fixed number of gpt-image-2 generations with this plan.'
                    )}
                  </p>
                </div>
                <div className='flex shrink-0 gap-2'>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={() => bonusResources.append(emptyBonusResource())}
                  >
                    <Plus data-icon='inline-start' />
                    {t('Add resource')}
                  </Button>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    disabled={imagePackConfigured}
                    onClick={() =>
                      bonusResources.append({
                        resource_key: 'gpt-image-2',
                        resource_type: 'image_count',
                        model_name: 'gpt-image-2',
                        display_name: '',
                        amount: 10,
                      })
                    }
                  >
                    <Sparkles data-icon='inline-start' />
                    {t('Add image pack')}
                  </Button>
                </div>
              </div>

              {bonusResources.fields.length === 0 ? (
                <p className='text-muted-foreground rounded-md border border-dashed p-3 text-xs'>
                  {t('No bonus resources configured')}
                </p>
              ) : (
                <div className='space-y-3'>
                  {bonusResources.fields.map((resource, index) => (
                    <div
                      key={resource.id}
                      className='bg-muted/20 rounded-md border p-3'
                    >
                      <div className='mb-3 flex items-center justify-between gap-2'>
                        <span className='text-xs font-medium'>
                          {t('Bonus resource {{number}}', {
                            number: index + 1,
                          })}
                        </span>
                        <Button
                          type='button'
                          variant='ghost'
                          size='icon'
                          className='size-7'
                          aria-label={t('Remove resource')}
                          onClick={() => bonusResources.remove(index)}
                        >
                          <Trash2 className='size-3.5' />
                        </Button>
                      </div>
                      <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
                        <FormField
                          control={form.control}
                          name={`bonus_resources.${index}.model_name`}
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>{t('Model name')}</FormLabel>
                              <FormControl>
                                <Input
                                  {...field}
                                  placeholder='gpt-image-2 / grok-4'
                                />
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                        <FormField
                          control={form.control}
                          name={`bonus_resources.${index}.resource_key`}
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>{t('Resource key')}</FormLabel>
                              <FormControl>
                                <Input
                                  {...field}
                                  placeholder='gpt-image-2 / grok'
                                />
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                        <FormField
                          control={form.control}
                          name={`bonus_resources.${index}.resource_type`}
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>{t('Resource type')}</FormLabel>
                              <Select
                                items={[
                                  {
                                    value: 'quota',
                                    label: t('Quota units'),
                                  },
                                  {
                                    value: 'image_count',
                                    label: t('Image generations'),
                                  },
                                ]}
                                value={field.value}
                                onValueChange={field.onChange}
                              >
                                <FormControl>
                                  <SelectTrigger>
                                    <SelectValue />
                                  </SelectTrigger>
                                </FormControl>
                                <SelectContent alignItemWithTrigger={false}>
                                  <SelectGroup>
                                    <SelectItem value='quota'>
                                      {t('Quota units')}
                                    </SelectItem>
                                    <SelectItem value='image_count'>
                                      {t('Image generations')}
                                    </SelectItem>
                                  </SelectGroup>
                                </SelectContent>
                              </Select>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                        <FormField
                          control={form.control}
                          name={`bonus_resources.${index}.amount`}
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>{t('Amount')}</FormLabel>
                              <FormControl>
                                <Input
                                  {...field}
                                  type='number'
                                  min={1}
                                  onChange={(event) =>
                                    field.onChange(
                                      Number.parseInt(event.target.value, 10) ||
                                        0
                                    )
                                  }
                                />
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </div>
                      <FormField
                        control={form.control}
                        name={`bonus_resources.${index}.display_name`}
                        render={({ field }) => (
                          <FormItem className='mt-3'>
                            <FormLabel>{t('Display name')}</FormLabel>
                            <FormControl>
                              <Input
                                {...field}
                                placeholder={t(
                                  'Optional label shown to customers'
                                )}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </div>
                  ))}
                </div>
              )}
            </SideDrawerSection>

            {/* Public Card Presentation */}
            <SideDrawerSection>
              <h3 className='flex items-center gap-2 text-sm font-medium'>
                <IconBadge tone='warning' size='xs'>
                  <Tag />
                </IconBadge>
                {t('Public card presentation')}
              </h3>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Configure the badge and benefit copy shown on the public subscription card. Leave benefits empty to use the default model-package copy.'
                )}
              </p>

              <FormField
                control={form.control}
                name='badge_text'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Card badge')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        maxLength={64}
                        placeholder={t('e.g. New or Limited offer')}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Optional short label displayed above this plan. The purchase limit badge is generated separately.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='is_recommended'
                render={({ field }) => (
                  <FormItem className={sideDrawerSwitchItemClassName()}>
                    <div>
                      <FormLabel className='!mt-0'>
                        {t('Mark as recommended')}
                      </FormLabel>
                      <FormDescription>
                        {t(
                          'Shows the recommendation badge and emphasis on this plan.'
                        )}
                      </FormDescription>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='benefits'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className='flex items-center gap-1.5'>
                      <ListChecks className='size-3.5' aria-hidden='true' />
                      {t('Plan benefits')}
                    </FormLabel>
                    <FormControl>
                      <Textarea
                        value={(field.value || []).join('\n')}
                        maxLength={3000}
                        onChange={(event) =>
                          field.onChange(
                            event.target.value
                              .split('\n')
                              .map((benefit) => benefit.trim())
                              .filter(Boolean)
                          )
                        }
                        placeholder={t(
                          'One benefit per line. Leave empty to use the default copy.'
                        )}
                        rows={6}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Use short customer-facing sentences. Up to 12 benefits are shown in the order entered.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <p className='text-muted-foreground text-xs leading-5'>
                {t(
                  'Price, estimated credit, plan validity, model count, and availability are calculated from the price, quota, model, and purchase-limit settings above.'
                )}
              </p>
            </SideDrawerSection>

            {/* Duration Settings */}
            <SideDrawerSection>
              <h3 className='flex items-center gap-2 text-sm font-medium'>
                <IconBadge tone='chart-4' size='xs'>
                  <CalendarClock />
                </IconBadge>
                {t('Duration Settings')}
              </h3>

              <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='duration_unit'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Duration Unit')}</FormLabel>
                      <Select
                        items={durationUnitOpts.map((o) => ({
                          value: o.value,
                          label: o.label,
                        }))}
                        onValueChange={field.onChange}
                        value={field.value}
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent alignItemWithTrigger={false}>
                          <SelectGroup>
                            {durationUnitOpts.map((o) => (
                              <SelectItem key={o.value} value={o.value}>
                                {o.label}
                              </SelectItem>
                            ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {durationUnit === 'custom' ? (
                  <FormField
                    control={form.control}
                    name='custom_seconds'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Custom Seconds')}</FormLabel>
                        <FormControl>
                          <Input
                            {...field}
                            type='number'
                            min={1}
                            onChange={(e) =>
                              field.onChange(
                                Number.parseInt(e.target.value, 10) || 0
                              )
                            }
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                ) : (
                  <FormField
                    control={form.control}
                    name='duration_value'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Duration Value')}</FormLabel>
                        <FormControl>
                          <Input
                            {...field}
                            type='number'
                            min={1}
                            onChange={(e) =>
                              field.onChange(
                                Number.parseInt(e.target.value, 10) || 0
                              )
                            }
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
              </div>
            </SideDrawerSection>

            {/* Total Quota Reset */}
            <SideDrawerSection>
              <h3 className='flex items-center gap-2 text-sm font-medium'>
                <IconBadge tone='success' size='xs'>
                  <RefreshCw />
                </IconBadge>
                {t('Total Quota Reset')}
              </h3>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Controls only the total quota above. Daily, weekly, and monthly quotas reset independently.'
                )}
              </p>

              <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='quota_reset_period'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Reset Cycle')}</FormLabel>
                      <Select
                        items={resetPeriodOpts.map((o) => ({
                          value: o.value,
                          label: o.label,
                        }))}
                        onValueChange={field.onChange}
                        value={field.value}
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent alignItemWithTrigger={false}>
                          <SelectGroup>
                            {resetPeriodOpts.map((o) => (
                              <SelectItem key={o.value} value={o.value}>
                                {o.label}
                              </SelectItem>
                            ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='quota_reset_custom_seconds'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Custom Seconds')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type='number'
                          min={0}
                          disabled={resetPeriod !== 'custom'}
                          onChange={(e) =>
                            field.onChange(
                              Number.parseInt(e.target.value, 10) || 0
                            )
                          }
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </SideDrawerSection>

            {/* Payment Config */}
            <SideDrawerSection>
              <h3 className='flex items-center gap-2 text-sm font-medium'>
                <IconBadge tone='warning' size='xs'>
                  <CreditCard />
                </IconBadge>
                {t('Third-party Payment Config')}
              </h3>

              <FormField
                control={form.control}
                name='stripe_price_id'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Stripe Price ID</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder='price_...' />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='creem_product_id'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Creem Product ID</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder='prod_...' />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='waffo_pancake_product_id'
                render={({ field }) => {
                  // Raw-ID fallback for IDs not yet in the catalog.
                  const items = pancakeProducts.map((p) => ({
                    value: p.id,
                    label: `${p.name} (${p.id})`,
                  }))
                  if (
                    field.value &&
                    !pancakeProducts.some((p) => p.id === field.value)
                  ) {
                    items.push({ value: field.value, label: field.value })
                  }
                  return (
                    <FormItem>
                      <FormLabel>Waffo Pancake Product ID</FormLabel>
                      <div className='flex gap-2'>
                        <Select
                          items={items}
                          value={field.value || ''}
                          onValueChange={(v) => field.onChange(v)}
                          disabled={items.length === 0}
                        >
                          <SelectTrigger className='w-full flex-1'>
                            <SelectValue placeholder={t('Select a product')} />
                          </SelectTrigger>
                          <SelectContent>
                            {items.map((item) => (
                              <SelectItem key={item.value} value={item.value}>
                                {item.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <Button
                          type='button'
                          variant='outline'
                          onClick={handleCreatePancakeProduct}
                          disabled={
                            creatingPancakeProduct || !pancakeCreateReady
                          }
                          className='shrink-0'
                        >
                          {creatingPancakeProduct
                            ? t('Creating...')
                            : `+ ${t('Create')}`}
                        </Button>
                      </div>
                      <FormDescription>
                        {t(
                          'Creates a Pancake product in the saved store using this plan’s title and price. Requires Waffo Pancake to be fully configured in Payment settings first.'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )
                }}
              />
            </SideDrawerSection>
          </form>
        </Form>
        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose render={<Button variant='outline' />}>
            {t('Close')}
          </SheetClose>
          <Button
            form='subscription-form'
            type='submit'
            disabled={isSubmitting}
          >
            {isSubmitting ? t('Saving...') : t('Save changes')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
