import { describe, expect, it } from 'vitest'
import { resolveLocale } from '@/i18n/locale'

describe('resolveLocale', () => {
  it('takes the saved language, in the browser regional variant when it has one', () => {
    expect(resolveLocale('nl', 'en-US,en;q=0.9')).toEqual({ language: 'nl', locale: 'nl' })
    expect(resolveLocale('nl', 'en-US,nl-BE;q=0.8')).toEqual({ language: 'nl', locale: 'nl-BE' })
  })

  it('follows the browser without a saved language, by q-value', () => {
    expect(resolveLocale(undefined, 'de-AT,de;q=0.9,en;q=0.8')).toEqual({
      language: 'de',
      locale: 'de-AT',
    })
    expect(resolveLocale(undefined, 'ja;q=0.9,fr-ca;q=0.95')).toEqual({
      language: 'fr',
      locale: 'fr-CA',
    })
    expect(resolveLocale(undefined, 'en-GB,en;q=0.9')).toEqual({ language: 'en', locale: 'en-GB' })
  })

  it('falls back to English', () => {
    expect(resolveLocale(undefined, null)).toEqual({ language: 'en', locale: 'en' })
    expect(resolveLocale(undefined, 'ja,zh;q=0.5,*;q=0.1')).toEqual({
      language: 'en',
      locale: 'en',
    })
    expect(resolveLocale(undefined, 'nl;q=0')).toEqual({ language: 'en', locale: 'en' })
  })

  it('ignores a cookie that is not a language Nimbus has, and broken tags', () => {
    expect(resolveLocale('../../etc/passwd', 'es')).toEqual({ language: 'es', locale: 'es' })
    expect(resolveLocale('xx', null)).toEqual({ language: 'en', locale: 'en' })
    expect(resolveLocale(undefined, 'es-!!')).toEqual({ language: 'es', locale: 'es' })
  })
})
