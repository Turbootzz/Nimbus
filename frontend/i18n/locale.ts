// The languages Nimbus is translated into; messages/<language>.json each
export const languages = ['en', 'nl', 'de', 'fr', 'es'] as const
export type Language = (typeof languages)[number]
export const defaultLanguage: Language = 'en'

// Holds the language the signed-in user picked; written by LocaleSync, read
// by i18n/request.ts. No cookie: follow the browser.
export const LOCALE_COOKIE = 'NEXT_LOCALE'

// Set by proxy.ts on the public status page, which never takes a user's
// language (it is the instance's page, not the visitor's)
export const PUBLIC_PAGE_HEADER = 'x-nimbus-public-page'

export function isLanguage(value: unknown): value is Language {
  return languages.includes(value as Language)
}

const primary = (tag: string) => tag.split('-')[0].toLowerCase()

// The browser's language tags, most wanted first (q-values of RFC 9110)
function acceptedTags(header: string | null): string[] {
  return (header ?? '')
    .split(',')
    .map((part) => {
      const [tag, ...params] = part.split(';').map((s) => s.trim())
      const q = params.find((p) => p.startsWith('q='))
      return { tag, q: q ? Number(q.slice(2)) : 1 }
    })
    .filter(({ tag, q }) => tag && tag !== '*' && q > 0)
    .sort((a, b) => b.q - a.q)
    .map(({ tag }) => tag)
}

function canonical(tag: string | undefined): string | undefined {
  if (!tag) return undefined
  try {
    return Intl.getCanonicalLocales(tag)[0]
  } catch {
    return undefined
  }
}

// resolveLocale picks the language (the saved one, else the browser's first
// one Nimbus has, else English) and the locale to format dates and numbers
// with: the browser's regional variant of that language when it has one
// (en-GB keeps a 24-hour clock), else the language itself.
export function resolveLocale(
  saved: string | undefined,
  acceptLanguage: string | null
): { language: Language; locale: string } {
  const tags = acceptedTags(acceptLanguage)
  const language = isLanguage(saved)
    ? saved
    : (tags.map(primary).find(isLanguage) ?? defaultLanguage)
  const locale = canonical(tags.find((tag) => primary(tag) === language)) ?? language
  return { language, locale }
}
