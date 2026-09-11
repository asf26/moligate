/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your
option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import type { SubscriptionPlan } from '../types'

export const MODEL_FAMILY_KEYS = [
  'ccmax',
  'gpt',
  'gemini',
  'chinese',
  'all',
] as const

export type ModelFamilyKey = (typeof MODEL_FAMILY_KEYS)[number]
export type ModelFamilyTone = 'orange' | 'green' | 'amber' | 'blue' | 'slate'

export interface ModelFamilyDefinition {
  key: ModelFamilyKey
  labelKey: string
  descriptionKey: string
  tone: ModelFamilyTone
  models: readonly string[]
  matchers: readonly string[]
}

const CC_MAX_MODELS = [
  'claude-opus-5',
  'claude-opus-4-8',
  'claude-opus-4-7',
  'claude-opus-4-6',
  'claude-sonnet-5',
  'claude-sonnet-4-6',
  'claude-haiku-4-5',
] as const

const GPT_MODELS = [
  'gpt-5.6-sol',
  'gpt-5.6-terra',
  'gpt-5.6-luna',
  'gpt-5.5',
  'gpt-5.4',
  'gpt-5.4-mini',
  'gpt-image-2',
] as const

const GEMINI_MODELS = [
  'gemini-2.5-pro',
  'gemini-2.5-flash',
  'gemini-2.0-flash',
  'gemini-2.0-flash-lite',
  'gemini-3-pro-preview',
  'gemini-3-flash-preview',
  'gemini-3.1-pro-preview',
] as const

const CHINESE_MODELS = [
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
] as const

const ALL_MODELS = [
  ...CC_MAX_MODELS,
  ...GPT_MODELS,
  ...GEMINI_MODELS,
  ...CHINESE_MODELS,
] as const

export const MODEL_FAMILY_DEFINITIONS: readonly ModelFamilyDefinition[] = [
  {
    key: 'ccmax',
    labelKey: 'CC Max',
    descriptionKey: 'Claude Code Max models for agentic development',
    tone: 'orange',
    models: CC_MAX_MODELS,
    matchers: ['ccmax', 'cc max', 'claude', 'anthropic'],
  },
  {
    key: 'gpt',
    labelKey: 'GPT',
    descriptionKey: 'GPT and OpenAI models for text, vision and image work',
    tone: 'green',
    models: GPT_MODELS,
    matchers: ['gpt', 'openai'],
  },
  {
    key: 'gemini',
    labelKey: 'Gemini',
    descriptionKey: 'Gemini models for fast multimodal workloads',
    tone: 'amber',
    models: GEMINI_MODELS,
    matchers: ['gemini', 'google'],
  },
  {
    key: 'chinese',
    labelKey: 'Chinese models',
    descriptionKey: 'DeepSeek, Qwen, GLM, Kimi, Doubao and MiniMax models',
    tone: 'blue',
    models: CHINESE_MODELS,
    matchers: [
      'chinese',
      '国产',
      'deepseek',
      'qwen',
      'glm',
      'kimi',
      'doubao',
      'minimax',
    ],
  },
  {
    key: 'all',
    labelKey: 'All supported models',
    descriptionKey:
      'One subscription with access to every supported model family',
    tone: 'slate',
    models: ALL_MODELS,
    matchers: ['all', 'unlimited', '通用', '全部'],
  },
]

const MODEL_FAMILY_MAP = new Map(
  MODEL_FAMILY_DEFINITIONS.map((family) => [family.key, family])
)

function normalizeFamilyValue(value: string | undefined): string {
  return value?.trim().toLowerCase() || ''
}

export function getModelFamilyDefinition(
  key: string | undefined
): ModelFamilyDefinition {
  const normalized = normalizeFamilyValue(key)
  if (MODEL_FAMILY_KEYS.includes(normalized as ModelFamilyKey)) {
    const family = MODEL_FAMILY_MAP.get(normalized as ModelFamilyKey)
    if (family) return family
  }
  const fallback = MODEL_FAMILY_MAP.get('all')
  if (!fallback) {
    throw new Error('The all-supported model family must be configured')
  }
  return fallback
}

export function getPlanModelFamily(
  plan: SubscriptionPlan
): ModelFamilyDefinition {
  const explicit = getModelFamilyDefinition(plan.model_family)
  if (plan.model_family?.trim()) return explicit

  const searchable = [
    plan.title,
    plan.subtitle,
    ...(plan.applicable_groups || []),
  ]
    .join(' ')
    .toLowerCase()

  return (
    MODEL_FAMILY_DEFINITIONS.find((family) =>
      family.matchers.some((matcher) => searchable.includes(matcher))
    ) || explicit
  )
}

export function getIncludedModels(
  plan: SubscriptionPlan,
  family: ModelFamilyDefinition
): string[] {
  const configured = (plan.included_models || [])
    .map((model) => model.trim())
    .filter(Boolean)

  return configured.length > 0 ? [...new Set(configured)] : [...family.models]
}

export interface ModelFamilyPlanGroup {
  family: ModelFamilyDefinition
  plans: SubscriptionPlan[]
}

export function groupPlansByModelFamily(
  plans: SubscriptionPlan[]
): ModelFamilyPlanGroup[] {
  const grouped = new Map<ModelFamilyKey, SubscriptionPlan[]>()
  for (const plan of plans) {
    const family = getPlanModelFamily(plan)
    const current = grouped.get(family.key) || []
    current.push(plan)
    grouped.set(family.key, current)
  }

  return MODEL_FAMILY_DEFINITIONS.filter((family) =>
    grouped.has(family.key)
  ).map((family) => ({
    family,
    plans: grouped.get(family.key) || [],
  }))
}
