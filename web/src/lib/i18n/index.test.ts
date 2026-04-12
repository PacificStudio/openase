import { describe, expect, it } from 'vitest'

import { localeLabel, pageTitle, parseLocale, translate } from './index'

describe('i18n helpers', () => {
  it('parses supported locales from language tags', () => {
    expect(parseLocale('en-US')).toBe('en')
    expect(parseLocale('zh-CN')).toBe('zh')
  })

  it('falls back to English for unknown locales', () => {
    expect(parseLocale('fr-FR')).toBe('en')
    expect(parseLocale(undefined)).toBe('en')
  })

  it('translates labels in the active display locale', () => {
    expect(localeLabel('zh', 'en')).toBe('Chinese')
    expect(localeLabel('en', 'zh')).toBe('英文')
  })

  it('builds translated page titles', () => {
    expect(pageTitle(translate('zh', 'dashboard.pageTitle'), 'zh')).toBe('工作台 - OpenASE')
  })
})
