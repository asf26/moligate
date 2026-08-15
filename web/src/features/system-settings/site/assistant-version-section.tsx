import { zodResolver } from '@hookform/resolvers/zod'
import type { Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
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
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

type AssistantVersionFormValues = {
  AssistantVersion: string
  AssistantForceUpdate: boolean
  AssistantReleaseNotes: string
  AssistantMacDownloadURL: string
  AssistantMacSignature: string
  AssistantWinDownloadURL: string
  AssistantWinSignature: string
  AssistantPublishedAt: string
}

type AssistantVersionSectionProps = {
  defaultValues: AssistantVersionFormValues
}

function normalizeValue(value: unknown): string {
  if (value === undefined || value === null) return ''
  return typeof value === 'string' ? value : String(value)
}

export function AssistantVersionSection(props: AssistantVersionSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const setConfig = useSystemConfigStore((state) => state.setConfig)

  const normalizedDefaults: AssistantVersionFormValues = {
    AssistantVersion: normalizeValue(props.defaultValues.AssistantVersion),
    AssistantForceUpdate: props.defaultValues.AssistantForceUpdate === true,
    AssistantReleaseNotes: normalizeValue(
      props.defaultValues.AssistantReleaseNotes
    ),
    AssistantMacDownloadURL: normalizeValue(
      props.defaultValues.AssistantMacDownloadURL
    ),
    AssistantMacSignature: normalizeValue(
      props.defaultValues.AssistantMacSignature
    ),
    AssistantWinDownloadURL: normalizeValue(
      props.defaultValues.AssistantWinDownloadURL
    ),
    AssistantWinSignature: normalizeValue(
      props.defaultValues.AssistantWinSignature
    ),
    AssistantPublishedAt: normalizeValue(
      props.defaultValues.AssistantPublishedAt
    ),
  }

  const assistantVersionSchemaWithI18n = z.object({
    AssistantVersion: z.string(),
    AssistantForceUpdate: z.boolean(),
    AssistantReleaseNotes: z.string(),
    AssistantMacDownloadURL: z
      .string()
      .url({ error: () => t('Please enter a valid download URL') })
      .optional()
      .or(z.literal('')),
    AssistantMacSignature: z.string(),
    AssistantWinDownloadURL: z
      .string()
      .url({ error: () => t('Please enter a valid download URL') })
      .optional()
      .or(z.literal('')),
    AssistantWinSignature: z.string(),
    AssistantPublishedAt: z
      .string()
      .regex(/^\d*$/, { message: t('Please enter a valid Unix timestamp') }),
  })

  const { form, handleSubmit, handleReset, isDirty, isSubmitting } =
    useSettingsForm<AssistantVersionFormValues>({
      resolver: zodResolver(assistantVersionSchemaWithI18n) as Resolver<
        AssistantVersionFormValues,
        unknown,
        AssistantVersionFormValues
      >,
      defaultValues: normalizedDefaults,
      onSubmit: async (data, changedFields) => {
        for (const [key, value] of Object.entries(changedFields)) {
          await updateOption.mutateAsync({
            key,
            value:
              typeof value === 'boolean' ? value : normalizeValue(value).trim(),
          })
        }

        setConfig({
          assistant: {
            version: normalizeValue(data.AssistantVersion).trim(),
            forceUpdate: data.AssistantForceUpdate,
            releaseNotes: normalizeValue(data.AssistantReleaseNotes).trim(),
            macDownloadUrl: normalizeValue(data.AssistantMacDownloadURL).trim(),
            macSignature: normalizeValue(data.AssistantMacSignature).trim(),
            winDownloadUrl: normalizeValue(data.AssistantWinDownloadURL).trim(),
            winSignature: normalizeValue(data.AssistantWinSignature).trim(),
            publishedAt: Number(normalizeValue(data.AssistantPublishedAt)) || 0,
          },
        })
      },
    })

  return (
    <>
      <FormNavigationGuard when={isDirty} />

      <SettingsSection
        title={t('Assistant Version Management')}
        description={t(
          'Configure the assistant version and release package links'
        )}
      >
        <Form {...form}>
          <form onSubmit={handleSubmit} className='flex flex-col gap-5'>
            <SettingsPageFormActions
              onSave={handleSubmit}
              onReset={handleReset}
              isSaving={isSubmitting || updateOption.isPending}
              isResetDisabled={!isDirty}
            />
            <FormDirtyIndicator isDirty={isDirty} />

            <div className='bg-muted/20 rounded-xl border p-4'>
              <div className='mb-4 flex flex-col gap-1'>
                <h4 className='text-sm font-semibold'>
                  {t('Release information')}
                </h4>
                <p className='text-muted-foreground text-xs'>
                  {t('Controls the version returned to assistant clients.')}
                </p>
              </div>
              <div className='grid gap-4 lg:grid-cols-[1fr_180px]'>
                <FormField
                  control={form.control}
                  name='AssistantVersion'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Assistant version')}</FormLabel>
                      <FormControl>
                        <Input
                          placeholder={t('Enter assistant version')}
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>
                        {t('Use SemVer, for example 0.1.1.')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='AssistantPublishedAt'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Published timestamp')}</FormLabel>
                      <FormControl>
                        <Input placeholder={t('1710000000')} {...field} />
                      </FormControl>
                      <FormDescription>{t('Unix timestamp.')}</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </div>

            <FormField
              control={form.control}
              name='AssistantForceUpdate'
              render={({ field }) => (
                <FormItem className='flex flex-row items-center justify-between rounded-lg border p-4'>
                  <div className='flex flex-col gap-0.5'>
                    <FormLabel className='text-base'>
                      {t('Force assistant update')}
                    </FormLabel>
                    <FormDescription>
                      {t('Require clients to update before continuing.')}
                    </FormDescription>
                  </div>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                      disabled={updateOption.isPending}
                    />
                  </FormControl>
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='AssistantReleaseNotes'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Assistant release notes')}</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder={t('Enter assistant release notes')}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Release notes returned by the version check API.')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className='grid gap-4 xl:grid-cols-2'>
              <div className='bg-muted/20 rounded-xl border p-4'>
                <div className='mb-4 flex flex-col gap-1'>
                  <h4 className='text-sm font-semibold'>
                    {t('macOS package')}
                  </h4>
                  <p className='text-muted-foreground text-xs'>
                    {t('Use the macOS updater .app.tar.gz generated by Tauri.')}
                  </p>
                </div>
                <div className='flex flex-col gap-4'>
                  <FormField
                    control={form.control}
                    name='AssistantMacDownloadURL'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Download URL')}</FormLabel>
                        <FormControl>
                          <Input
                            placeholder={t(
                              'https://example.com/assistant.app.tar.gz'
                            )}
                            {...field}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name='AssistantMacSignature'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Signature')}</FormLabel>
                        <FormControl>
                          <Textarea
                            className='min-h-24 font-mono text-xs'
                            placeholder={t('Paste the .sig file content')}
                            {...field}
                          />
                        </FormControl>
                        <FormDescription>
                          {t('Paste signature content, not a signature URL.')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </div>

              <div className='bg-muted/20 rounded-xl border p-4'>
                <div className='mb-4 flex flex-col gap-1'>
                  <h4 className='text-sm font-semibold'>
                    {t('Windows package')}
                  </h4>
                  <p className='text-muted-foreground text-xs'>
                    {t('Use the x64 setup installer generated by Tauri.')}
                  </p>
                </div>
                <div className='flex flex-col gap-4'>
                  <FormField
                    control={form.control}
                    name='AssistantWinDownloadURL'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Download URL')}</FormLabel>
                        <FormControl>
                          <Input
                            placeholder={t(
                              'https://example.com/assistant-setup.exe'
                            )}
                            {...field}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name='AssistantWinSignature'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Signature')}</FormLabel>
                        <FormControl>
                          <Textarea
                            className='min-h-24 font-mono text-xs'
                            placeholder={t('Paste the .sig file content')}
                            {...field}
                          />
                        </FormControl>
                        <FormDescription>
                          {t('Paste signature content, not a signature URL.')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </div>
            </div>
          </form>
        </Form>
      </SettingsSection>
    </>
  )
}
