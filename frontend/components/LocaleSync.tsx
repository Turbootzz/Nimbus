'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useTheme } from '@/contexts/ThemeContext'
import { LOCALE_COOKIE } from '@/i18n/locale'

// The language in the cookie; undefined without one (or an emptied one)
function readLocaleCookie(): string | undefined {
  const prefix = `${LOCALE_COOKIE}=`
  const cookie = document.cookie.split('; ').find((c) => c.startsWith(prefix))
  return cookie?.slice(prefix.length) || undefined
}

// Keeps the language cookie that i18n/request.ts reads on the user's saved
// language, and renders the page again when that changes it
export default function LocaleSync() {
  const { language } = useTheme()
  const router = useRouter()

  useEffect(() => {
    // Not known yet (or the preferences could not be loaded)
    if (language === undefined) return
    if (readLocaleCookie() === (language ?? undefined)) return
    document.cookie = language
      ? `${LOCALE_COOKIE}=${language}; path=/; max-age=31536000; samesite=lax`
      : `${LOCALE_COOKIE}=; path=/; max-age=0; samesite=lax`
    router.refresh()
  }, [language, router])

  return null
}
