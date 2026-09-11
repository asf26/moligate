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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'
import type { PricingModel } from '@/features/pricing/types'
import { cn } from '@/lib/utils'

interface ModelFamily {
  name: string
  models: string[]
  tone: ModelTone
}

type ModelTone = 'orange' | 'green' | 'amber' | 'blue' | 'slate'

const MAX_MODELS_PER_FAMILY = 8

interface ModelCategory {
  name: string
  tone: ModelTone
  matches: (model: PricingModel) => boolean
}

const CHINESE_MODEL_PATTERN =
  /deepseek|qwen|glm|kimi|doubao|ernie|wenxin|hunyuan|internlm|yi-|mimo|stepfun|moonshot|aliyun|alibaba|zhipu|智谱|阿里|通义|字节|百度|腾讯|讯飞|月之暗面|百川|零一万物|混元|minimax/i

function matchesModel(model: PricingModel, pattern: RegExp): boolean {
  return pattern.test(`${model.model_name} ${model.vendor_name ?? ''}`)
}

const MODEL_CATEGORIES: ModelCategory[] = [
  {
    name: 'Claude',
    tone: 'orange',
    matches: (model) => matchesModel(model, /claude|anthropic/),
  },
  {
    name: 'GPT',
    tone: 'blue',
    matches: (model) => matchesModel(model, /gpt|openai|codex/),
  },
  {
    name: 'Chinese models',
    tone: 'green',
    matches: (model) => matchesModel(model, CHINESE_MODEL_PATTERN),
  },
  {
    name: 'Gemini',
    tone: 'amber',
    matches: (model) => matchesModel(model, /gemini|google|gemma/),
  },
  {
    name: 'Grok',
    tone: 'slate',
    matches: (model) => matchesModel(model, /grok|xai/),
  },
  {
    name: 'Other models',
    tone: 'slate',
    matches: () => true,
  },
]

const fallbackModelFamilies: ModelFamily[] = [
  {
    name: 'Claude',
    tone: 'orange',
    models: [
      'claude-opus-5',
      'claude-opus-4-8',
      'claude-opus-4-7',
      'claude-opus-4-6',
      'claude-sonnet-5',
      'claude-sonnet-4-6',
      'claude-haiku-4-5',
    ],
  },
  {
    name: 'GPT',
    tone: 'blue',
    models: [
      'gpt-5.6-sol',
      'gpt-5.6-terra',
      'gpt-5.6-luna',
      'gpt-5.5',
      'gpt-5.4',
      'gpt-5.4-mini',
      'gpt-image-2',
    ],
  },
  {
    name: 'Chinese models',
    tone: 'green',
    models: [
      'deepseek-v4-pro',
      'deepseek-v4-flash',
      'qwen3.8-max',
      'qwen3.8-flash',
      'qwen3.7-max',
      'qwen3-vl-235b-a22b-thinking',
      'glm-5.3',
      'glm-5.3-flash',
      'kimi-k3',
      'kimi-k2.7-code',
      'kimi-k2.7-code-highspeed',
      'doubao-seed-2.0-pro',
      'doubao-seed-2.0-lite',
      'doubao-seed-2.0-mini',
      'doubao-seed-2.0-code',
      'minimax-m3',
    ],
  },
  {
    name: 'Gemini',
    tone: 'amber',
    models: [
      'gemini-2.5-pro',
      'gemini-2.5-flash',
      'gemini-2.0-flash',
      'gemini-2.0-flash-lite',
      'gemini-3-pro-preview',
      'gemini-3-flash-preview',
      'gemini-3.1-pro-preview',
    ],
  },
]

function buildModelFamilies(models: PricingModel[]): ModelFamily[] {
  const familyModels = new Map<string, string[]>()
  const seenModels = new Set<string>()

  for (const model of models) {
    const modelName = model.model_name.trim()
    if (!modelName || seenModels.has(modelName)) continue

    seenModels.add(modelName)
    const familyName =
      MODEL_CATEGORIES.find((category) => category.matches(model))?.name ??
      'Other models'
    const modelsInFamily = familyModels.get(familyName) ?? []

    if (modelsInFamily.length < MAX_MODELS_PER_FAMILY) {
      modelsInFamily.push(modelName)
      familyModels.set(familyName, modelsInFamily)
    }
  }

  return MODEL_CATEGORIES.flatMap((category) => {
    const modelsInCategory = familyModels.get(category.name)
    return modelsInCategory
      ? [{ name: category.name, models: modelsInCategory, tone: category.tone }]
      : []
  })
}

export function ModelDirectory() {
  const { t } = useTranslation()
  const { models } = usePricingData()
  const remoteModelFamilies = useMemo(
    () => buildModelFamilies(models),
    [models]
  )
  const modelFamilies =
    remoteModelFamilies.length > 0 ? remoteModelFamilies : fallbackModelFamilies

  return (
    <section id='models' className='home-reference-models px-4 pb-16'>
      <h2 className='sr-only'>{t('Models directory')}</h2>
      <div className='home-reference-model-rail mx-auto max-w-5xl'>
        {modelFamilies.map((family) => (
          <div
            key={family.name}
            className='home-reference-model-row grid gap-3 py-5 md:grid-cols-[7rem_1fr] md:items-center md:gap-6'
          >
            <h3 className='home-reference-family-label'>
              <span
                className={cn(
                  'home-reference-family-dot',
                  `home-reference-tone-${family.tone}`
                )}
                aria-hidden='true'
              />
              {t(family.name)}
            </h3>
            <div className='flex flex-wrap gap-2'>
              {family.models.map((model) => (
                <span key={model} className='home-reference-model-chip'>
                  <span
                    className={cn(
                      'home-reference-family-dot',
                      `home-reference-tone-${family.tone}`
                    )}
                    aria-hidden='true'
                  />
                  {model}
                </span>
              ))}
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}
