import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { normalizeInterfaceLanguage } from './languages'

describe('normalizeInterfaceLanguage', () => {
  test('maps legacy Chinese language tags to the supported zh resource', () => {
    assert.equal(normalizeInterfaceLanguage('zhCN'), 'zh')
    assert.equal(normalizeInterfaceLanguage('zh-CN'), 'zh')
    assert.equal(normalizeInterfaceLanguage('zh_Hans_CN'), 'zh')
  })

  test('keeps supported language-only tags and falls back to English', () => {
    assert.equal(normalizeInterfaceLanguage('fr'), 'fr')
    assert.equal(normalizeInterfaceLanguage('ja'), 'ja')
    assert.equal(normalizeInterfaceLanguage('not-supported'), 'en')
  })
})
