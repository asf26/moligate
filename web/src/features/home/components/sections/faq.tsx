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

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'

const faqDefinitions = [
  {
    question: 'How can I use Claude in China?',
    answer:
      'Create an account, generate an API key, and point Claude Code or the Anthropic SDK at the gateway base URL. The request is routed through an available domestic-compatible provider.',
  },
  {
    question: 'What is a Claude proxy or relay?',
    answer:
      'A relay exposes a compatible API endpoint between your application and the upstream model provider. Your integration keeps its existing SDK while the gateway handles routing and billing.',
  },
  {
    question: 'How is a relay different from the official API?',
    answer:
      'The gateway keeps the request and response shape compatible, while adding unified provider access, usage accounting, and payment options for this deployment.',
  },
  {
    question: 'Can Claude Code be used domestically?',
    answer:
      'Yes. Set ANTHROPIC_BASE_URL and ANTHROPIC_API_KEY, then run Claude Code normally. No source-code changes are required.',
  },
  {
    question: 'How is gateway pricing calculated?',
    answer:
      'Choose a monthly plan for a predictable limit or use balance billing by token. The console records model, usage, latency, and charge for each request.',
  },
  {
    question: 'How do I call GPT or OpenAI models?',
    answer:
      'Use the OpenAI-compatible base URL with the same SDK and API key. Select the model name in your request; the gateway handles provider routing.',
  },
]

export function FAQ() {
  const { t } = useTranslation()
  const defaultExpandedQuestion = faqDefinitions[0]?.question ?? ''

  return (
    <section
      id='faq'
      className='home-reference-section home-reference-faq-section px-4 pb-24'
    >
      <div className='mx-auto max-w-3xl'>
        <div className='text-center'>
          <p className='home-reference-kicker'>{t('FAQ')}</p>
          <h2 className='home-reference-heading mt-3'>
            {t('Common questions')}
          </h2>
        </div>
        <Accordion
          multiple
          defaultValue={[defaultExpandedQuestion]}
          className='home-reference-faq mt-10'
        >
          {faqDefinitions.map((item) => (
            <AccordionItem
              key={item.question}
              value={item.question}
              className='home-reference-faq-item'
            >
              <AccordionTrigger className='home-reference-faq-trigger hover:no-underline'>
                {t(item.question)}
              </AccordionTrigger>
              <AccordionContent className='home-reference-faq-content'>
                <p>{t(item.answer)}</p>
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      </div>
    </section>
  )
}
