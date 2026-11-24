import { describe, expect, it } from 'vitest'
import { checkLocale, placeholders } from '@/scripts/check-messages.mjs'

describe('placeholders', () => {
  it('finds arguments and tags, also inside plural options', () => {
    expect(placeholders('Hello {name}')).toEqual({ args: 'name', tags: '' })
    expect(
      placeholders('{count, plural, =0 {none} one {# <b>item</b> of {total}} other {# items}}')
    ).toEqual({ args: 'count,total', tags: 'b' })
    expect(placeholders('{n, number, ::percent} {when, date, short}')).toEqual({
      args: 'n,when',
      tags: '',
    })
  })

  it('reads apostrophes like ICU', () => {
    expect(placeholders("Don't stop")).toEqual({ args: '', tags: '' })
    expect(placeholders("Use '{braces}' and it''s {x}")).toEqual({ args: 'x', tags: '' })
  })

  it('throws on a message ICU cannot read', () => {
    expect(() => placeholders('Hello {name')).toThrow()
    expect(() => placeholders('Hello }')).toThrow()
  })
})

describe('checkLocale', () => {
  const en = { a: { title: 'Title', count: '{count} left' }, b: 'B' }

  it('passes a complete translation', () => {
    expect(checkLocale(en, { a: { title: 'Titel', count: 'nog {count}' }, b: 'B' })).toEqual([])
  })

  it('reports missing, extra and broken keys', () => {
    expect(checkLocale(en, { a: { title: 'Titel', count: 'nog {aantal}' }, c: 'C' })).toEqual([
      'a.count has arguments {aantal}, en.json {count}',
      'missing key b',
      'extra key c',
    ])
  })
})
