'use client'

import { useTheme } from '@/contexts/ThemeContext'
import { safeMode } from '@/lib/custom-css'

// The user's own CSS, only on the pages behind the login (the dashboard
// layout renders this). The CSS is a text child: React writes it as text,
// never as HTML, so it can't close the element or add a script. The server
// refused anything that loads from another host before storing it.
export default function CustomCss() {
  const { customCss } = useTheme()
  if (!customCss || safeMode) return null
  return <style data-custom-css="">{customCss}</style>
}
