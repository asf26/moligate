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
import { useFAQ } from '@/features/dashboard/hooks/use-status-data'
import type { FAQItem } from '@/features/dashboard/types'

const fallbackFaqDefinitions: FAQItem[] = [
  {
    question: 'What is Moligate?',
    answer:
      'Moligate is a unified AI gateway for developers and teams. One account connects multiple model providers with shared API access, usage tracking, and billing.',
  },
  {
    question: 'How do I start using Moligate?',
    answer:
      'Register or sign in, create an API key in the console, choose a supported model, and follow the integration guide. You can also open AI Creation to start without writing code.',
  },
  {
    question: 'Which models can I use with Moligate?',
    answer:
      'Moligate brings together supported OpenAI, Claude, Gemini, and other provider models. The model catalog shows the models currently available to your account.',
  },
  {
    question: 'How do I create and use an API key?',
    answer:
      'Open API Keys in the console, create a key with the required group and permissions, then add it to your SDK or Authorization header. Keep your key private and rotate it when needed.',
  },
  {
    question: 'How are usage and prices calculated?',
    answer:
      "Charges follow the selected model and actual usage. Subscription limits and balance billing are shown in the console, where each request's model, usage, and charge can be reviewed.",
  },
  {
    question: 'How do I call Moligate from my application?',
    answer:
      'Use the API base URL and key shown in the console with an OpenAI-compatible SDK, then select the model in your request. Moligate handles provider routing, usage records, and billing.',
  },
]

export function FAQ() {
  const { t } = useTranslation()
  const { items: configuredFaq } = useFAQ()
  const remoteFaq = configuredFaq.filter(
    (item) => item.question.trim() && item.answer.trim()
  )
  const faqItems = remoteFaq.length > 0 ? remoteFaq : fallbackFaqDefinitions
  const usesConfiguredFaq = remoteFaq.length > 0
  const defaultExpandedQuestion = faqItems[0]?.question ?? ''

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
          {faqItems.map((item) => {
            const question = usesConfiguredFaq
              ? item.question
              : t(item.question)
            const answer = usesConfiguredFaq ? item.answer : t(item.answer)

            return (
              <AccordionItem
                key={item.id ?? item.question}
                value={item.question}
                className='home-reference-faq-item'
              >
                <AccordionTrigger className='home-reference-faq-trigger hover:no-underline'>
                  {question}
                </AccordionTrigger>
                <AccordionContent className='home-reference-faq-content'>
                  <p className='whitespace-pre-line'>{answer}</p>
                </AccordionContent>
              </AccordionItem>
            )
          })}
        </Accordion>
      </div>
    </section>
  )
}
