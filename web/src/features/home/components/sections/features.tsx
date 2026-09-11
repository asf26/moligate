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
import {
  BarChart3,
  CircleDollarSign,
  Globe2,
  KeyRound,
  LifeBuoy,
  PlugZap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'

const featureDefinitions = [
  {
    title: 'Perfect Claude Code compatibility',
    description:
      'Set two environment variables for ANTHROPIC_BASE_URL and use the official CLI with no code changes.',
    icon: PlugZap,
  },
  {
    title: 'One key, multiple providers',
    description:
      'Anthropic, OpenAI and Codex endpoints are available together, with free model switching from one account.',
    icon: KeyRound,
  },
  {
    title: 'Subscription + usage, freely combined',
    description:
      'Use monthly limits first, then continue with token-based balance billing when the plan limit is reached.',
    icon: CircleDollarSign,
  },
  {
    title: 'China-friendly access',
    description:
      'Domestic routes with card, Alipay and WeChat payment support. No extra proxy configuration is needed.',
    icon: Globe2,
  },
  {
    title: 'Transparent billing',
    description:
      'Review tokens, model, latency and every charge for each request directly in the console.',
    icon: BarChart3,
  },
  {
    title: 'Complete support',
    description:
      'Built-in tickets, complete integration guides and full request-id tracing help resolve issues quickly.',
    icon: LifeBuoy,
  },
]

export function Features() {
  const { t } = useTranslation()

  return (
    <section id='features' className='home-reference-section px-4 pb-24'>
      <div className='mx-auto max-w-6xl'>
        <div className='text-center'>
          <p className='home-reference-kicker'>{t('Why choose us')}</p>
          <h2 className='home-reference-heading mt-3'>
            {t('A multi-model AI gateway built for developers')}
          </h2>
        </div>

        <div className='mt-12 grid gap-4 md:grid-cols-2 lg:grid-cols-3'>
          {featureDefinitions.map((feature, index) => {
            const Icon = feature.icon
            return (
              <AnimateInView
                key={feature.title}
                delay={index * 70}
                className='home-reference-feature-card'
              >
                <span className='home-reference-feature-icon'>
                  <Icon aria-hidden='true' />
                </span>
                <h3 className='mt-5 text-base font-semibold'>
                  {t(feature.title)}
                </h3>
                <p className='text-muted-foreground mt-2 text-sm leading-6'>
                  {t(feature.description)}
                </p>
              </AnimateInView>
            )
          })}
        </div>
      </div>
    </section>
  )
}
