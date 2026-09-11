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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import { createVideoAccount, updateVideoAccount } from '../api'
import type { VideoAccount, VideoAccountPayload } from '../types'

type VideoAccountDialogProps = {
  open: boolean
  account?: VideoAccount | null
  onOpenChange: (open: boolean) => void
}

type FormState = {
  name: string
  apiKey: string
  status: number
  groups: string
  proxy: string
  remark: string
}

function getInitialForm(account?: VideoAccount | null): FormState {
  return {
    name: account?.name ?? '',
    apiKey: '',
    status: account?.status ?? 1,
    groups: account?.groups.join(', ') ?? 'default',
    proxy: account?.proxy ?? '',
    remark: account?.remark ?? '',
  }
}

export function VideoAccountDialog(props: VideoAccountDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [form, setForm] = useState<FormState>(() =>
    getInitialForm(props.account)
  )
  const isEditing = Boolean(props.account)

  const mutation = useMutation({
    mutationFn: async (payload: VideoAccountPayload) => {
      if (props.account) {
        return updateVideoAccount(props.account.id, payload)
      }
      return createVideoAccount(payload)
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to save'))
        return
      }
      toast.success(
        t(
          isEditing
            ? 'Video account updated successfully'
            : 'Video account created successfully'
        )
      )
      queryClient.invalidateQueries({ queryKey: ['video-accounts'] })
      props.onOpenChange(false)
    },
    onError: () => toast.error(t('Failed to save')),
  })

  const updateField = <K extends keyof FormState>(
    key: K,
    value: FormState[K]
  ) => {
    setForm((current) => ({ ...current, [key]: value }))
  }

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const name = form.name.trim()
    const apiKey = form.apiKey.trim()
    if (!name) {
      toast.error(t('Name is required'))
      return
    }
    if (!isEditing && !apiKey) {
      toast.error(t('API key is required'))
      return
    }

    const groups = form.groups
      .split(/[\n,]/)
      .map((group) => group.trim())
      .filter(Boolean)
    const payload: VideoAccountPayload = {
      name,
      status: form.status,
      groups: groups.length > 0 ? groups : ['default'],
      proxy: form.proxy.trim() || undefined,
      remark: form.remark.trim() || undefined,
      base_url: 'https://video.ctmoai.com',
    }
    if (apiKey) payload.api_key = apiKey
    mutation.mutate(payload)
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>
            {t(isEditing ? 'Edit Video Account' : 'Add Video Account')}
          </DialogTitle>
          <DialogDescription>
            {t(
              'Manage dedicated CTMOAI video accounts and key-scoped model catalogs.'
            )}
          </DialogDescription>
        </DialogHeader>

        <form className='flex flex-col gap-5' onSubmit={handleSubmit}>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor='video-account-name'>{t('Name')}</FieldLabel>
              <Input
                id='video-account-name'
                value={form.name}
                onChange={(event) => updateField('name', event.target.value)}
                placeholder={t('Account holder name')}
                autoComplete='off'
                required
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='video-account-api-key'>
                {t('API Key')}
              </FieldLabel>
              <Input
                id='video-account-api-key'
                type='password'
                value={form.apiKey}
                onChange={(event) => updateField('apiKey', event.target.value)}
                placeholder={
                  isEditing
                    ? props.account?.api_key_masked ||
                      t('Leave empty to keep current values unchanged.')
                    : t('Enter the CTMOAI API key')
                }
                autoComplete='new-password'
                required={!isEditing}
              />
              <FieldDescription>
                {t(
                  'CTMOAI API keys stay on the server and are never exposed to users.'
                )}
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor='video-account-base-url'>
                {t('Base URL')}
              </FieldLabel>
              <Input
                id='video-account-base-url'
                value='https://video.ctmoai.com'
                readOnly
                aria-readonly='true'
                className='bg-muted/40'
              />
              <FieldDescription>
                {t(
                  'The CTMOAI video gateway URL is fixed for this account type.'
                )}
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor='video-account-groups'>
                {t('Groups')}
              </FieldLabel>
              <Input
                id='video-account-groups'
                value={form.groups}
                onChange={(event) => updateField('groups', event.target.value)}
                placeholder={t('Groups (comma-separated)')}
                autoComplete='off'
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='video-account-proxy'>
                {t('Proxy Address')}
              </FieldLabel>
              <Input
                id='video-account-proxy'
                value={form.proxy}
                onChange={(event) => updateField('proxy', event.target.value)}
                placeholder='https://user:pass@proxy.example.com:8080'
                autoComplete='off'
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='video-account-remark'>
                {t('Remark')}
              </FieldLabel>
              <Textarea
                id='video-account-remark'
                value={form.remark}
                onChange={(event) => updateField('remark', event.target.value)}
                placeholder={t('Additional information')}
                rows={3}
              />
            </Field>

            <Field orientation='horizontal'>
              <FieldLabel htmlFor='video-account-status'>
                {t('Enabled')}
              </FieldLabel>
              <Switch
                id='video-account-status'
                checked={form.status === 1}
                onCheckedChange={(checked) =>
                  updateField('status', checked ? 1 : 0)
                }
                aria-label={t('Enabled')}
              />
            </Field>
          </FieldGroup>

          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => props.onOpenChange(false)}
              disabled={mutation.isPending}
            >
              {t('Cancel')}
            </Button>
            <Button type='submit' disabled={mutation.isPending}>
              {mutation.isPending && <Spinner data-icon='inline-start' />}
              {t(mutation.isPending ? 'Saving...' : 'Save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
