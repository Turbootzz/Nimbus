import fs from 'fs'
import path from 'path'
import { createTranslator } from 'next-intl'
import { describe, expect, it } from 'vitest'
import { flatten, placeholders } from '@/scripts/check-messages.mjs'
import { languages } from '@/i18n/locale'

const read = (language: string) =>
  JSON.parse(fs.readFileSync(path.join(__dirname, '../../messages', `${language}.json`), 'utf8'))
const keys = Object.keys(flatten(read('en')))

// check-messages compares placeholders; this formats every message with
// next-intl itself, so a message ICU reads differently than meant (a
// quoting apostrophe, a broken plural) fails here
describe.each(languages)('messages/%s.json', (language) => {
  const messages = read(language)
  const flat = flatten(messages) as Record<string, string>
  const t = createTranslator({
    locale: language,
    messages,
    onError: (error) => {
      throw error
    },
  })

  it.each([0, 1, 2, 1234])('formats every message with a count of %i', (count) => {
    for (const key of keys) {
      const { args, tags } = placeholders(flat[key])
      const values: Record<string, string | number | ((chunks: string) => string)> = {}
      for (const arg of args.split(',').filter(Boolean)) {
        values[arg] = new RegExp(`\\{\\s*${arg}\\s*,\\s*plural`).test(flat[key]) ? count : 'X'
      }
      for (const tag of tags.split(',').filter(Boolean)) values[tag] = (chunks) => chunks
      const text: string = t.markup(key, values)
      expect(text, key).not.toMatch(/[{}]|undefined/)
    }
  })
})
