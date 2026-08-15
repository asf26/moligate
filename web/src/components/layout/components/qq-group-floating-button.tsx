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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useTheme } from '@/context/theme-provider'
import { useSystemConfig } from '@/hooks/use-system-config'
import { cn } from '@/lib/utils'

export function QQGroupQRCodePanel() {
  const { t } = useTranslation()
  const { qqGroup, wechatGroup, loading } = useSystemConfig()
  const { resolvedTheme } = useTheme()
  const [failedQrcodeUrls, setFailedQrcodeUrls] = useState<
    Record<string, boolean>
  >({})

  const lightQrcodeUrl = qqGroup?.qrcodeUrlLight?.trim() ?? ''
  const darkQrcodeUrl = qqGroup?.qrcodeUrlDark?.trim() ?? ''
  const qrcodeUrl =
    resolvedTheme === 'dark'
      ? darkQrcodeUrl || lightQrcodeUrl
      : lightQrcodeUrl || darkQrcodeUrl
  const groupNumber = qqGroup?.number?.trim() ?? ''
  const wechatLightQrcodeUrl = wechatGroup?.qrcodeUrlLight?.trim() ?? ''
  const wechatDarkQrcodeUrl = wechatGroup?.qrcodeUrlDark?.trim() ?? ''
  const wechatQrcodeUrl =
    resolvedTheme === 'dark'
      ? wechatDarkQrcodeUrl || wechatLightQrcodeUrl
      : wechatLightQrcodeUrl || wechatDarkQrcodeUrl

  const groups = [
    {
      key: 'wechat',
      title: t('Official WeChat Group'),
      shortLabel: t('WeChat'),
      qrcodeUrl: wechatQrcodeUrl,
      enabled: wechatGroup?.enabled === true,
      alt: t('WeChat group QR code'),
      groupNumber: '',
    },
    {
      key: 'qq',
      title: t('Official QQ Group'),
      shortLabel: 'QQ',
      qrcodeUrl,
      enabled: qqGroup?.enabled === true,
      alt: t('QQ group QR code'),
      groupNumber,
    },
  ].filter((group) => group.enabled && group.qrcodeUrl.length > 0)

  if (loading || groups.length === 0) {
    return null
  }

  return (
    <div className='w-full group-data-[collapsible=icon]:hidden'>
      <Popover>
        <PopoverTrigger
          render={
            <Button
              variant='ghost'
              className='text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground mx-auto h-auto w-[88%] max-w-[10.75rem] justify-start gap-2 rounded-lg px-2 py-2'
              aria-label={t('Open community QR codes')}
            >
              <span className='bg-sidebar-primary text-sidebar-primary-foreground flex size-7 shrink-0 items-center justify-center rounded-md text-xs font-semibold'>
                QR
              </span>
              <span className='flex min-w-0 flex-1 flex-col items-start gap-0.5'>
                <span className='truncate text-xs font-medium'>
                  {t('Community groups')}
                </span>
                <span className='text-muted-foreground truncate text-[11px]'>
                  {groups.map((group) => group.shortLabel).join(' / ')}
                </span>
              </span>
            </Button>
          }
        />
        <PopoverContent
          side='right'
          align='end'
          sideOffset={10}
          className='w-[15.5rem] gap-3 p-3'
        >
          <Tabs defaultValue={groups[0]?.key} className='gap-3'>
            {groups.length > 1 ? (
              <TabsList className='grid w-full grid-cols-2'>
                {groups.map((group) => (
                  <TabsTrigger key={group.key} value={group.key}>
                    {group.shortLabel}
                  </TabsTrigger>
                ))}
              </TabsList>
            ) : null}

            {groups.map((group) => {
              const imageFailed = failedQrcodeUrls[group.qrcodeUrl] === true

              return (
                <TabsContent
                  key={group.key}
                  value={group.key}
                  className='flex flex-col gap-2'
                >
                  <p className='text-center text-sm font-medium'>
                    {group.title}
                  </p>
                  <div className='bg-muted grid aspect-square w-full place-items-center overflow-hidden rounded-md'>
                    {imageFailed ? (
                      <p className='text-muted-foreground px-2 text-center text-xs'>
                        {t('QR code failed to load')}
                      </p>
                    ) : (
                      <img
                        src={group.qrcodeUrl}
                        alt={group.alt}
                        className='size-full object-contain'
                        loading='lazy'
                        onError={() =>
                          setFailedQrcodeUrls((current) => ({
                            ...current,
                            [group.qrcodeUrl]: true,
                          }))
                        }
                      />
                    )}
                  </div>

                  {group.groupNumber ? (
                    <div className='flex min-w-0 items-center justify-center gap-1 px-1 py-0.5'>
                      <p className='min-w-0 truncate text-center text-xs leading-4'>
                        <span className='text-muted-foreground'>
                          {t('QQ Group:')}
                        </span>
                        <span className='ml-1 font-medium'>
                          {group.groupNumber}
                        </span>
                      </p>
                      <CopyButton
                        value={group.groupNumber}
                        variant='ghost'
                        size='icon'
                        className={cn(
                          '-mr-1 size-6 shrink-0 rounded',
                          'text-muted-foreground hover:text-foreground'
                        )}
                        tooltip={t('Copy QQ group number')}
                        aria-label={t('Copy QQ group number')}
                      />
                    </div>
                  ) : null}
                </TabsContent>
              )
            })}
          </Tabs>
        </PopoverContent>
      </Popover>
    </div>
  )
}
