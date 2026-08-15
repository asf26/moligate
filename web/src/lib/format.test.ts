import { describe, expect, test } from 'vitest'

import { formatTimestampRelative } from './format'

describe('formatTimestampRelative', () => {
  test('normalizes compact interface language codes before using Intl', () => {
    expect(() => {
      formatTimestampRelative(Date.now(), 'milliseconds', 'zhCN')
    }).not.toThrow()
  })
})
