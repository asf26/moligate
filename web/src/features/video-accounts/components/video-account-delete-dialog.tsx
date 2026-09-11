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
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'

import { deleteVideoAccount } from '../api'
import type { VideoAccount } from '../types'

type VideoAccountDeleteDialogProps = {
  account: VideoAccount | null
  onOpenChange: (open: boolean) => void
}

export function VideoAccountDeleteDialog(props: VideoAccountDeleteDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: () => {
      if (!props.account) {
        throw new Error('video account is missing')
      }
      return deleteVideoAccount(props.account.id)
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to delete account'))
        return
      }
      toast.success(t('Video account deleted successfully'))
      queryClient.invalidateQueries({ queryKey: ['video-accounts'] })
      props.onOpenChange(false)
    },
    onError: () => toast.error(t('Failed to delete account')),
  })

  return (
    <ConfirmDialog
      open={props.account !== null}
      onOpenChange={props.onOpenChange}
      title={t('Delete Account')}
      desc={
        <>
          {t('Are you sure you want to delete this video account?')}{' '}
          <span className='font-semibold'>{props.account?.name}</span>
          {'. '}
          {t('This action cannot be undone.')}
        </>
      }
      confirmText={mutation.isPending ? t('Deleting...') : t('Delete')}
      destructive
      isLoading={mutation.isPending}
      handleConfirm={() => mutation.mutate()}
    />
  )
}
