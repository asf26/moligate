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
import {
  ArrowRight,
  BookOpen,
  Check,
  CircleDollarSign,
  KeyRound,
  Sparkles,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface CTAProps {
  className?: string
  isAuthenticated?: boolean
  docsUrl?: string
}

export function CTA(props: CTAProps) {
  const { t } = useTranslation()
  const primaryHref = props.isAuthenticated ? '/dashboard' : '/sign-up'
  const primaryLabel = props.isAuthenticated
    ? t('Console')
    : t('Free registration')

  return (
    <section
      className={cn(
        'home-reference-final home-reference-final-cta px-4',
        props.className
      )}
    >
      <div className='home-reference-cta mx-auto max-w-6xl px-6 py-10 sm:px-12 sm:py-14'>
        <AnimateInView
          animation='fade-right'
          className='home-reference-cta-copy relative z-10'
        >
          <div className='home-reference-cta-kicker'>
            <span className='home-reference-cta-icon' aria-hidden='true'>
              <Sparkles />
            </span>
            <p className='home-reference-kicker'>{t('Get started')}</p>
          </div>
          <h2 className='home-reference-heading mt-3'>
            {t('Ready to get started?')}
          </h2>
          <p className='text-muted-foreground mt-3 max-w-xl text-sm leading-6'>
            {t(
              'Free registration, five minutes to your first Claude request. No credit card required.'
            )}
          </p>
          <div
            className='home-reference-cta-benefits'
            aria-label={t('Get started')}
          >
            <span>
              <Check aria-hidden='true' />
              {t('One key, multiple providers')}
            </span>
            <span>
              <Check aria-hidden='true' />
              {t('Subscription + usage, freely combined')}
            </span>
          </div>
        </AnimateInView>
        <AnimateInView
          animation='fade-left'
          className='home-reference-cta-side relative z-10'
          delay={120}
        >
          <div className='home-reference-cta-signal' aria-hidden='true'>
            <span>
              <Zap />
            </span>
            <span>
              <KeyRound />
            </span>
            <span>
              <CircleDollarSign />
            </span>
          </div>
          <div className='home-reference-cta-actions'>
            <Button
              className='home-reference-button home-reference-button-primary home-reference-cta-primary'
              render={<Link to={primaryHref} />}
              nativeButton={false}
            >
              {primaryLabel}
              <ArrowRight data-icon='inline-end' aria-hidden='true' />
            </Button>
            <Button
              variant='outline'
              className='home-reference-button home-reference-button-secondary'
              render={
                <a
                  href={props.docsUrl || 'https://docs.newapi.pro'}
                  target='_blank'
                  rel='noopener noreferrer'
                />
              }
              nativeButton={false}
            >
              <BookOpen data-icon='inline-start' aria-hidden='true' />
              {t('View integration docs')}
            </Button>
          </div>
        </AnimateInView>
      </div>
    </section>
  )
}
