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
import { Download, Loader2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'

type ExportField = {
  key: string
  labelKey: string
}

const EXPORT_FIELDS: ExportField[] = [
  { key: 'id', labelKey: 'ID' },
  { key: 'created_at', labelKey: 'Time' },
  { key: 'type', labelKey: 'Type' },
  { key: 'content', labelKey: 'Content' },
  { key: 'user_id', labelKey: 'User ID' },
  { key: 'username', labelKey: 'Username' },
  { key: 'token_id', labelKey: 'Token ID' },
  { key: 'token_name', labelKey: 'Token' },
  { key: 'model_name', labelKey: 'Model' },
  { key: 'quota', labelKey: 'Quota' },
  { key: 'prompt_tokens', labelKey: 'Prompt Tokens' },
  { key: 'completion_tokens', labelKey: 'Completion Tokens' },
  { key: 'use_time', labelKey: 'Use Time' },
  { key: 'is_stream', labelKey: 'Stream' },
  { key: 'channel', labelKey: 'Channel ID' },
  { key: 'channel_name', labelKey: 'Channel Name' },
  { key: 'group', labelKey: 'Group' },
  { key: 'ip', labelKey: 'IP' },
  { key: 'request_id', labelKey: 'Request ID' },
  { key: 'upstream_request_id', labelKey: 'Upstream Request ID' },
  { key: 'other', labelKey: 'Other' },
]

const DEFAULT_EXPORT_FIELDS = [
  'id',
  'created_at',
  'type',
  'username',
  'token_name',
  'model_name',
  'quota',
  'prompt_tokens',
  'completion_tokens',
  'use_time',
  'channel',
  'channel_name',
  'group',
  'request_id',
  'upstream_request_id',
]

interface UsageLogsExportDialogProps {
  disabled?: boolean
  onExport: (fields: string[]) => Promise<void>
}

export function UsageLogsExportDialog(props: UsageLogsExportDialogProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [isExporting, setIsExporting] = useState(false)
  const [selectedFields, setSelectedFields] = useState<string[]>(
    DEFAULT_EXPORT_FIELDS
  )
  const selectedSet = new Set(selectedFields)

  const toggleField = (fieldKey: string, checked: boolean) => {
    if (checked) {
      setSelectedFields((current) =>
        current.includes(fieldKey) ? current : [...current, fieldKey]
      )
      return
    }
    setSelectedFields((current) => current.filter((key) => key !== fieldKey))
  }

  const handleExport = async () => {
    if (selectedFields.length === 0) {
      return
    }
    setIsExporting(true)
    try {
      await props.onExport(selectedFields)
      setOpen(false)
    } finally {
      setIsExporting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger
        render={
          <Button variant='outline' disabled={props.disabled}>
            <Download />
            {t('Export CSV')}
          </Button>
        }
      />
      <DialogContent className='sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>{t('Export CSV')}</DialogTitle>
          <DialogDescription>
            {t('Export logs using the current filters and selected fields.')}
          </DialogDescription>
        </DialogHeader>

        <div className='flex flex-wrap items-center gap-2'>
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() =>
              setSelectedFields(EXPORT_FIELDS.map((field) => field.key))
            }
          >
            {t('Select All')}
          </Button>
          <Button
            type='button'
            variant='ghost'
            size='sm'
            onClick={() => setSelectedFields(DEFAULT_EXPORT_FIELDS)}
          >
            {t('Default Fields')}
          </Button>
          <Button
            type='button'
            variant='ghost'
            size='sm'
            onClick={() => setSelectedFields([])}
          >
            {t('Clear')}
          </Button>
        </div>

        <div className='grid max-h-[340px] gap-2 overflow-y-auto pr-1 sm:grid-cols-2'>
          {EXPORT_FIELDS.map((field) => (
            <label
              key={field.key}
              className='border-border/70 hover:bg-muted/40 flex cursor-pointer items-center gap-2 rounded-md border px-3 py-2 text-sm'
            >
              <Checkbox
                checked={selectedSet.has(field.key)}
                onCheckedChange={(checked) =>
                  toggleField(field.key, checked === true)
                }
              />
              <span>{t(field.labelKey)}</span>
            </label>
          ))}
        </div>

        <DialogFooter>
          <Button
            variant='outline'
            type='button'
            onClick={() => setOpen(false)}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='button'
            disabled={isExporting || selectedFields.length === 0}
            onClick={handleExport}
          >
            {isExporting && <Loader2 className='animate-spin' />}
            {t('Download CSV')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
