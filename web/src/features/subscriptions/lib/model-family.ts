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
  'kiro-claude',
  'gpt',
  'gemini',
  'chinese',
  'gpt-image',
  'banana',
] as const

export type ModelFamilyKey = (typeof MODEL_FAMILY_KEYS)[number]
export type ModelFamilyTone =
  | 'orange'
  | 'green'
  | 'amber'
  | 'blue'
  | 'slate'
  | 'violet'
  | 'pink'

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
] as const

const KIRO_CLAUDE_MODELS = CC_MAX_MODELS

const GPT_IMAGE_MODELS = [
  'gpt-image-2',
  'gpt-image-2.5-sunburst',
  'gpt-image-2.5-flare',
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
  'DeepSeek-V4.1-Flash',
  'deepseek-v4-pro',
  'deepseek-v4-flash',
  'deepseek-v4-flash-0731',
  'deepseek-v4-pro-0813',
  'glm-5.2',
  'glm-5.3',
  'glm-5.3-flash',
  'kimi-k3',
  'kimi-k2.7-code',
  'kimi-k2.7-code-highspeed',
  'MiniMax-M2.7',
  'MiniMax-M2.7-highspeed',
  'MiniMax-M3',
] as const

const BANANA_MODELS = [
  'gemini-2.0-flash-exp-image-generation',
  'gemini-2.0-flash-exp',
  'gemini-2.5-flash-image',
  'gemini-3-pro-image',
  'gemini-3-pro-image-preview',
  'gemini-3.1-flash-image',
  'gemini-3.1-flash-image-preview',
  'nano-banana-pro-preview',
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
    key: 'kiro-claude',
    labelKey: 'Kiro Claude',
    descriptionKey: 'Kiro Claude models for coding and agent workflows',
    tone: 'slate',
    models: KIRO_CLAUDE_MODELS,
    matchers: ['kiro-claude', 'kiro claude', 'kiro'],
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
    descriptionKey: 'GLM, DeepSeek, Kimi and MiniMax models',
    tone: 'blue',
    models: CHINESE_MODELS,
    matchers: ['chinese', '国产', 'deepseek', 'glm', 'kimi', 'minimax'],
  },
  {
    key: 'gpt-image',
    labelKey: 'GPT Image',
    descriptionKey: 'GPT Image models for image generation',
    tone: 'violet',
    models: GPT_IMAGE_MODELS,
    matchers: ['gpt-image', 'gpt image', 'image generation', '图片生成'],
  },
  {
    key: 'banana',
    labelKey: 'Nano Banana',
    descriptionKey: 'Nano Banana models for image generation',
    tone: 'pink',
    models: BANANA_MODELS,
    matchers: [
      'nano banana',
      'banana',
      '香蕉生图',
      'gemini-2.0-flash-exp-image-generation',
      'gemini-2.0-flash-exp',
      'gemini-2.5-flash-image',
      'gemini-3-pro-image',
      'gemini-3-pro-image-preview',
      'gemini-3.1-flash-image',
      'gemini-3.1-flash-image-preview',
      'nano-banana-pro-preview',
    ],
  },
]

const MODEL_FAMILY_MAP = new Map(
  MODEL_FAMILY_DEFINITIONS.map((family) => [family.key, family])
)

const MODEL_FAMILY_ALIASES: Record<string, ModelFamilyKey> = {
  kiro: 'kiro-claude',
  'kiro claude': 'kiro-claude',
  kiro_claude: 'kiro-claude',
  'gpt image': 'gpt-image',
  gpt_image: 'gpt-image',
  'nano banana': 'banana',
}

function normalizeFamilyValue(value: string | undefined): string {
  return value?.trim().toLowerCase() || ''
}

export function getModelFamilyDefinition(
  key: string | undefined
): ModelFamilyDefinition | undefined {
  const normalized = normalizeFamilyValue(key)
  const resolvedKey = MODEL_FAMILY_ALIASES[normalized] || normalized
  return MODEL_FAMILY_MAP.get(resolvedKey as ModelFamilyKey)
}

function findFamilyByMetadata(
  plan: SubscriptionPlan
): ModelFamilyDefinition | undefined {
  const metadata = [
    plan.title,
    plan.subtitle,
    ...(plan.applicable_groups || []),
  ]
    .join(' ')
    .toLowerCase()

  const matches = MODEL_FAMILY_DEFINITIONS.flatMap((family) =>
    family.matchers
      .filter((matcher) => metadata.includes(matcher))
      .map((matcher) => ({ family, matcher }))
  )
  matches.sort((left, right) => right.matcher.length - left.matcher.length)
  return matches[0]?.family
}

function findFamilyByIncludedModels(
  plan: SubscriptionPlan
): ModelFamilyDefinition | undefined {
  const configured = [
    ...new Set(
      (plan.included_models || [])
        .map((model) => model.trim().toLowerCase())
        .filter(Boolean)
    ),
  ]
  if (configured.length === 0) return undefined

  const matches = MODEL_FAMILY_DEFINITIONS.filter((family) =>
    configured.every((model) =>
      family.models.some((candidate) => candidate.toLowerCase() === model)
    )
  )
  return matches.length === 1 ? matches[0] : undefined
}

export function getPlanModelFamily(
  plan: SubscriptionPlan
): ModelFamilyDefinition | undefined {
  const explicit = getModelFamilyDefinition(plan.model_family)
  if (explicit) return explicit

  // Legacy `all` and unknown values are inferred only when their metadata
  // identifies one concrete family. Otherwise the plan stays hidden instead
  // of being presented as a misleading catch-all or Kiro plan.
  return findFamilyByMetadata(plan) || findFamilyByIncludedModels(plan)
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
    if (!family) continue
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
