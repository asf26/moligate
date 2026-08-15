import { describe, expect, test } from 'vitest'

import { normalizeIntlLocales } from './intl-locale'

describe('normalizeIntlLocales', () => {
  test('repairs compact legacy locale tags before they reach Intl', () => {
    expect(normalizeIntlLocales('zhCN')).toBe('zh-CN')
    expect(normalizeIntlLocales('zh_CN')).toBe('zh-CN')
    expect(normalizeIntlLocales('zhHansCN')).toBe('zh-Hans-CN')
    expect(normalizeIntlLocales('enUS')).toBe('en-US')
  })

  test('drops invalid locale tags so Intl uses the runtime default', () => {
    expect(normalizeIntlLocales('not a locale')).toBeUndefined()
    expect(() => {
      new Intl.NumberFormat(normalizeIntlLocales('not a locale')).format(1)
    }).not.toThrow()
  })

  test('normalizes locale arrays and removes invalid entries', () => {
    expect(normalizeIntlLocales(['zhCN', 'en_us', 'bad tag'])).toEqual([
      'zh-CN',
      'en-US',
    ])
  })
})
