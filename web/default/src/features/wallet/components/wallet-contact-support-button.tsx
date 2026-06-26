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
import { MessageCircle, Video } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { useTheme } from '@/context/theme-provider'
import { useSystemConfig } from '@/hooks/use-system-config'
import { cn } from '@/lib/utils'

export function WalletContactSupportButton() {
  const { t } = useTranslation()
  const { qqGroup, wechatGroup, loading } = useSystemConfig()
  const { resolvedTheme } = useTheme()
  const [failedQrcodeUrls, setFailedQrcodeUrls] = useState<
    Record<string, boolean>
  >({})

  const qqLightQrcodeUrl = qqGroup?.qrcodeUrlLight?.trim() ?? ''
  const qqDarkQrcodeUrl = qqGroup?.qrcodeUrlDark?.trim() ?? ''
  const qqQrcodeUrl =
    resolvedTheme === 'dark'
      ? qqDarkQrcodeUrl || qqLightQrcodeUrl
      : qqLightQrcodeUrl || qqDarkQrcodeUrl
  const wechatLightQrcodeUrl = wechatGroup?.qrcodeUrlLight?.trim() ?? ''
  const wechatDarkQrcodeUrl = wechatGroup?.qrcodeUrlDark?.trim() ?? ''
  const wechatQrcodeUrl =
    resolvedTheme === 'dark'
      ? wechatDarkQrcodeUrl || wechatLightQrcodeUrl
      : wechatLightQrcodeUrl || wechatDarkQrcodeUrl

  const contacts = [
    {
      key: 'wechat',
      title: t('Official WeChat Group'),
      qrcodeUrl: wechatQrcodeUrl,
      enabled: wechatGroup?.enabled === true,
      alt: t('WeChat group QR code'),
      groupNumber: '',
    },
    {
      key: 'qq',
      title: t('Official QQ Group'),
      qrcodeUrl: qqQrcodeUrl,
      enabled: qqGroup?.enabled === true,
      alt: t('QQ group QR code'),
      groupNumber: qqGroup?.number?.trim() ?? '',
    },
  ].filter((contact) => contact.enabled && contact.qrcodeUrl.length > 0)

  if (loading || contacts.length === 0) {
    return null
  }

  return (
    <Dialog>
      <DialogTrigger
        render={
          <Button
            size='sm'
            className='w-full border-0 bg-zinc-950 px-3 text-white shadow-lg shadow-cyan-500/15 hover:bg-zinc-900 sm:w-auto dark:bg-zinc-50 dark:text-zinc-950 dark:hover:bg-white'
          >
            <Video data-icon='inline-start' />
            {t('Contact support')}
          </Button>
        }
      />
      <DialogContent
        className={cn(contacts.length === 1 ? 'sm:max-w-sm' : 'sm:max-w-xl')}
      >
        <DialogHeader>
          <div className='flex items-center gap-2'>
            <span className='bg-muted flex size-8 items-center justify-center rounded-lg'>
              <MessageCircle className='h-4 w-4' />
            </span>
            <div>
              <DialogTitle>{t('Contact support')}</DialogTitle>
              <DialogDescription>
                {t('Scan a QR code to join the official support community.')}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div
          className={cn(
            'grid grid-cols-1 gap-4',
            contacts.length > 1 && 'sm:grid-cols-2'
          )}
        >
          {contacts.map((contact) => {
            const imageFailed = failedQrcodeUrls[contact.qrcodeUrl] === true

            return (
              <div key={contact.key} className='flex min-w-0 flex-col gap-3'>
                <p className='text-center text-sm font-medium'>
                  {contact.title}
                </p>
                <div className='bg-muted mx-auto grid aspect-square w-full max-w-52 place-items-center overflow-hidden rounded-lg border p-2'>
                  {imageFailed ? (
                    <p className='text-muted-foreground px-2 text-center text-xs'>
                      {t('QR code failed to load')}
                    </p>
                  ) : (
                    <img
                      src={contact.qrcodeUrl}
                      alt={contact.alt}
                      className='size-full object-contain'
                      loading='lazy'
                      onError={() =>
                        setFailedQrcodeUrls((current) => ({
                          ...current,
                          [contact.qrcodeUrl]: true,
                        }))
                      }
                    />
                  )}
                </div>

                {contact.groupNumber ? (
                  <div className='flex min-w-0 items-center justify-center gap-1'>
                    <p className='min-w-0 truncate text-center text-xs leading-4'>
                      <span className='text-muted-foreground'>
                        {t('QQ Group:')}
                      </span>
                      <span className='ml-1 font-medium'>
                        {contact.groupNumber}
                      </span>
                    </p>
                    <CopyButton
                      value={contact.groupNumber}
                      variant='ghost'
                      size='icon'
                      className='text-muted-foreground hover:text-foreground -mr-1 size-6 shrink-0 rounded'
                      tooltip={t('Copy QQ group number')}
                      aria-label={t('Copy QQ group number')}
                    />
                  </div>
                ) : null}
              </div>
            )
          })}
        </div>
      </DialogContent>
    </Dialog>
  )
}
