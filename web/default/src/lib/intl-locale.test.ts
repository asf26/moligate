import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { normalizeIntlLocales } from './intl-locale'

describe('normalizeIntlLocales', () => {
  test('repairs compact legacy locale tags before they reach Intl', () => {
    assert.equal(normalizeIntlLocales('zhCN'), 'zh-CN')
    assert.equal(normalizeIntlLocales('zh_CN'), 'zh-CN')
    assert.equal(normalizeIntlLocales('zhHansCN'), 'zh-Hans-CN')
    assert.equal(normalizeIntlLocales('enUS'), 'en-US')
  })

  test('drops invalid locale tags so Intl uses the runtime default', () => {
    assert.equal(normalizeIntlLocales('not a locale'), undefined)
    assert.doesNotThrow(() => {
      new Intl.NumberFormat(normalizeIntlLocales('not a locale')).format(1)
    })
  })

  test('normalizes locale arrays and removes invalid entries', () => {
    assert.deepEqual(normalizeIntlLocales(['zhCN', 'en_us', 'bad tag']), [
      'zh-CN',
      'en-US',
    ])
  })
})
