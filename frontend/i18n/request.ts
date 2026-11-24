import { cookies, headers } from 'next/headers'
import { getRequestConfig } from 'next-intl/server'
import { LOCALE_COOKIE, PUBLIC_PAGE_HEADER, defaultLanguage, resolveLocale } from './locale'

// The language of a request: the user's saved choice (a cookie, see
// LocaleSync), else the browser's, else English. No locale in the URL. The
// public status page is not translated and never takes a user's choice.
export default getRequestConfig(async () => {
  const [cookieStore, headerList] = await Promise.all([cookies(), headers()])
  const { language, locale } = headerList.has(PUBLIC_PAGE_HEADER)
    ? { language: defaultLanguage, locale: defaultLanguage }
    : resolveLocale(cookieStore.get(LOCALE_COOKIE)?.value, headerList.get('accept-language'))
  return {
    locale,
    messages: (await import(`../messages/${language}.json`)).default,
  }
})
