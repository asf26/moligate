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
import { useTranslation } from 'react-i18next'

import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

interface IntegrationExample {
  id: string
  label: string
  language: string
  lines: string[]
}

const integrationExamples: IntegrationExample[] = [
  {
    id: 'claude-code',
    label: 'Claude Code',
    language: 'shell',
    lines: [
      'export ANTHROPIC_BASE_URL=https://your-gateway.example.com',
      'export ANTHROPIC_API_KEY=sk_your_key_here',
      '',
      'claude --bare',
    ],
  },
  {
    id: 'anthropic',
    label: 'Anthropic SDK',
    language: 'python',
    lines: [
      'from anthropic import Anthropic',
      '',
      'client = Anthropic(',
      '  base_url="https://your-gateway.example.com",',
      '  api_key="sk_your_key_here",',
      ')',
    ],
  },
  {
    id: 'openai',
    label: 'OpenAI SDK',
    language: 'python',
    lines: [
      'from openai import OpenAI',
      '',
      'client = OpenAI(',
      '  base_url="https://your-gateway.example.com/v1",',
      '  api_key="sk_your_key_here",',
      ')',
    ],
  },
  {
    id: 'curl',
    label: 'curl',
    language: 'shell',
    lines: [
      'curl https://your-gateway.example.com/v1/messages \\',
      '  -H "x-api-key: sk_your_key_here" \\',
      '  -H "content-type: application/json" \\',
      '  -d \'{"model":"your-model","messages":[]}\'',
    ],
  },
]

export function HowItWorks() {
  const { t } = useTranslation()

  return (
    <section
      id='integration'
      className='home-reference-section home-reference-integration px-4 pb-24'
    >
      <div className='mx-auto max-w-4xl'>
        <div className='text-center'>
          <p className='home-reference-kicker'>{t('5-minute integration')}</p>
          <h2 className='home-reference-heading mt-3'>
            {t('Call it just like the official Claude API')}
          </h2>
          <p className='text-muted-foreground mx-auto mt-4 max-w-2xl text-sm leading-5'>
            {t(
              'Point base_url to your gateway and replace api_key with your access key. Everything else stays the same.'
            )}
          </p>
        </div>

        <Tabs defaultValue='claude-code' className='home-reference-code mt-10'>
          <TabsList
            variant='line'
            className='home-reference-code-tabs h-auto w-full justify-start overflow-x-auto rounded-none border-b px-4 py-0'
          >
            {integrationExamples.map((example) => (
              <TabsTrigger
                key={example.id}
                value={example.id}
                className='home-reference-code-trigger flex-none px-3'
              >
                {t(example.label)}
              </TabsTrigger>
            ))}
          </TabsList>
          {integrationExamples.map((example) => (
            <TabsContent
              key={example.id}
              value={example.id}
              className='home-reference-code-content relative m-0 min-h-44 px-5 py-5 sm:px-6'
            >
              <span className='home-reference-code-language'>
                {example.language}
              </span>
              <pre className='overflow-x-auto font-mono text-xs leading-7 sm:text-sm'>
                <code>{example.lines.join('\n')}</code>
              </pre>
            </TabsContent>
          ))}
        </Tabs>
      </div>
    </section>
  )
}
