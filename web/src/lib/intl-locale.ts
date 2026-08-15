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

const COMPACT_LOCALE_ALIASES: Record<string, string> = {
  engb: 'en-GB',
  enus: 'en-US',
  ptbr: 'pt-BR',
  ptpt: 'pt-PT',
  zhcn: 'zh-CN',
  zhhans: 'zh-Hans',
  zhhanscn: 'zh-Hans-CN',
  zhhant: 'zh-Hant',
  zhhanttw: 'zh-Hant-TW',
  zhhk: 'zh-HK',
  zhsg: 'zh-SG',
  zhtw: 'zh-TW',
} as const

function repairCompactLocaleTag(value: string): string {
  const normalized = value.trim().replaceAll('_', '-')
  if (normalized.includes('-')) return normalized

  const alias =
    COMPACT_LOCALE_ALIASES[normalized.toLowerCase().replaceAll('-', '')]
  if (alias) return alias

  const langScriptRegion = normalized.match(
    /^([a-zA-Z]{2,3})([A-Z][a-z]{3})([A-Z]{2}|\d{3})$/
  )
  if (langScriptRegion) {
    return `${langScriptRegion[1]}-${langScriptRegion[2]}-${langScriptRegion[3]}`
  }

  const langScript = normalized.match(/^([a-zA-Z]{2,3})([A-Z][a-z]{3})$/)
  if (langScript) {
    return `${langScript[1]}-${langScript[2]}`
  }

  const langRegion = normalized.match(/^([a-zA-Z]{2,3})([A-Z]{2}|\d{3})$/)
  if (langRegion) {
    return `${langRegion[1]}-${langRegion[2]}`
  }

  return normalized
}

function normalizeIntlLocaleValue(value: string | Intl.Locale) {
  const repaired = repairCompactLocaleTag(String(value))
  if (!repaired) return undefined

  try {
    return Intl.getCanonicalLocales(repaired)[0]
  } catch {
    return undefined
  }
}

export function normalizeIntlLocales(
  locales?: Intl.LocalesArgument
): Intl.LocalesArgument | undefined {
  if (locales == null) return undefined

  const values = Array.isArray(locales) ? locales : [locales]
  const normalized = values
    .map((locale) => normalizeIntlLocaleValue(locale))
    .filter((locale): locale is string => Boolean(locale))

  if (normalized.length === 0) return undefined
  if (Array.isArray(locales)) return normalized
  return normalized[0]
}
