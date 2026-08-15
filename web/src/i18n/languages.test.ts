import { describe, expect, test } from 'vitest'

import { normalizeInterfaceLanguage } from './languages'

describe('normalizeInterfaceLanguage', () => {
  test('maps legacy Chinese language tags to the supported zh resource', () => {
    expect(normalizeInterfaceLanguage('zhCN')).toBe('zhCN')
    expect(normalizeInterfaceLanguage('zh-CN')).toBe('zhCN')
    expect(normalizeInterfaceLanguage('zh_Hans_CN')).toBe('zhCN')
    expect(normalizeInterfaceLanguage('zhTW')).toBe('zhTW')
    expect(normalizeInterfaceLanguage('zh-TW')).toBe('zhTW')
    expect(normalizeInterfaceLanguage('zh_Hant_TW')).toBe('zhTW')
  })

  test('keeps supported language-only tags and falls back to English', () => {
    expect(normalizeInterfaceLanguage('fr')).toBe('fr')
    expect(normalizeInterfaceLanguage('fr-FR')).toBe('fr')
    expect(normalizeInterfaceLanguage('ja')).toBe('ja')
    expect(normalizeInterfaceLanguage('not-supported')).toBe('en')
  })
})
