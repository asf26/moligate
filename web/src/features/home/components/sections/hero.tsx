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
import { Link } from '@tanstack/react-router'
import { ArrowRight, BookOpen, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'
import { cn } from '@/lib/utils'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

const proofPoints = [
  '18+ models',
  'Multi-SDK compatibility',
  'China payments',
  'Full request tracing',
]

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const docsUrl =
    (status?.docs_link as string | undefined) || 'https://docs.newapi.pro'
  const primaryHref = props.isAuthenticated ? '/dashboard' : '/sign-in'

  const docsButton = (
    <Button
      variant='outline'
      className='home-reference-button home-reference-button-secondary'
      render={
        docsUrl.startsWith('http') ? (
          <a href={docsUrl} target='_blank' rel='noopener noreferrer' />
        ) : (
          <Link to={docsUrl} />
        )
      }
      nativeButton={false}
    >
      <BookOpen data-icon='inline-start' aria-hidden='true' />
      {t('View documentation')}
    </Button>
  )

  return (
    <section
      id='top'
      className={cn(
        'home-reference-hero relative overflow-hidden px-4 pt-24 pb-12 sm:pt-28 sm:pb-16',
        props.className
      )}
    >
      <div className='relative z-10 mx-auto max-w-3xl text-center'>
        <div className='home-reference-eyebrow'>
          <Sparkles className='size-3' aria-hidden='true' />
          {t('Claude · GPT direct access · available in China')}
        </div>

        <h1 className='home-reference-title mt-6 text-4xl leading-[1.06] font-semibold text-balance sm:text-6xl'>
          <span className='block'>{t('One-stop AI model gateway')}</span>
          <span className='home-reference-title-accent mt-2 block'>
            {t('Claude · GPT direct access')}
          </span>
        </h1>

        <p className='text-muted-foreground mx-auto mt-6 max-w-2xl text-base leading-7 sm:text-lg'>
          {t(
            'A multi-model API gateway and proxy available in China. One key connects Claude and GPT model families, ready in five minutes.'
          )}
        </p>
        <p className='text-muted-foreground mx-auto mt-2 max-w-2xl text-sm leading-6'>
          {t(
            'Fully compatible with Claude Code, Anthropic SDK and OpenAI SDK. Use them as-is with no code changes.'
          )}
        </p>

        <div className='mt-9 flex flex-col justify-center gap-3 sm:flex-row'>
          <Button
            className='home-reference-button home-reference-button-primary'
            render={<Link to={primaryHref} />}
            nativeButton={false}
          >
            {t('Connect now')}
            <ArrowRight data-icon='inline-end' aria-hidden='true' />
          </Button>
          {docsButton}
        </div>

        <div className='home-reference-proof mt-8 flex flex-wrap items-center justify-center gap-x-5 gap-y-2'>
          {proofPoints.map((point, index) => (
            <span key={point} className='inline-flex items-center gap-5'>
              {index > 0 && (
                <span
                  className='home-reference-proof-divider'
                  aria-hidden='true'
                />
              )}
              {t(point)}
            </span>
          ))}
        </div>
      </div>
    </section>
  )
}
