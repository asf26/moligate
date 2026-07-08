import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { formatTimestampRelative } from './format'

describe('formatTimestampRelative', () => {
  test('normalizes compact interface language codes before using Intl', () => {
    assert.doesNotThrow(() => {
      formatTimestampRelative(Date.now(), 'milliseconds', 'zhCN')
    })
  })
})
