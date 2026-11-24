'use client'

import { useState, useSyncExternalStore } from 'react'
import { useTheme } from '@/contexts/ThemeContext'
import { MAX_CUSTOM_CSS_BYTES, safeMode } from '@/lib/custom-css'

const noSubscription = () => () => {}

export default function CustomCssSettings() {
  const { customCss, saveTheme } = useTheme()
  // Null until the user types: the field then follows what is stored
  const [draft, setDraft] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  // False on the server, so the notice can't mismatch on hydration
  const inSafeMode = useSyncExternalStore(
    noSubscription,
    () => safeMode,
    () => false
  )

  const value = draft ?? customCss
  const bytes = new TextEncoder().encode(value).length
  const tooLarge = bytes > MAX_CUSTOM_CSS_BYTES

  const save = async () => {
    const saved = value
    setSaving(true)
    const problem = await saveTheme({ custom_css: saved })
    setSaving(false)
    setError(problem)
    // Not when the user typed on while it saved
    if (!problem) setDraft((current) => (current === saved ? null : current))
  }

  return (
    <div className="glass-card border-card-border rounded-lg border p-6">
      <h2 className="text-text-primary mb-2 text-xl font-semibold">Custom CSS</h2>
      <p className="text-text-secondary mb-4 text-sm">
        Your own CSS on top of the theme. It only applies to you, on the pages behind the login.
      </p>

      {inSafeMode && (
        <p
          role="status"
          className="border-warning text-text-primary mb-4 rounded-lg border p-3 text-sm"
        >
          Safe mode: your custom CSS is off until you reload without <code>?safe=1</code>.
        </p>
      )}

      <label htmlFor="custom-css" className="sr-only">
        Custom CSS
      </label>
      <textarea
        id="custom-css"
        value={value}
        onChange={(e) => setDraft(e.target.value)}
        rows={10}
        spellCheck={false}
        placeholder=".glass-card { border-radius: 1rem; }"
        aria-describedby="custom-css-help"
        className="border-card-border bg-background text-text-primary focus:ring-primary w-full rounded-lg border px-3 py-2 font-mono text-xs focus:ring-2 focus:outline-none"
      />
      <div className="mt-2 flex items-center justify-between gap-4">
        <span className={`text-xs ${tooLarge ? 'text-error' : 'text-text-muted'}`}>
          {(bytes / 1024).toFixed(1)} / {MAX_CUSTOM_CSS_BYTES / 1024} KB
        </span>
        <button
          type="button"
          onClick={save}
          disabled={saving || tooLarge || draft === null || draft === customCss}
          className="bg-primary hover:bg-primary-hover rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors disabled:opacity-50"
        >
          {saving ? 'Saving...' : 'Save CSS'}
        </button>
      </div>
      {error && (
        <p role="alert" className="text-error mt-2 text-sm">
          {error}
        </p>
      )}
      <p id="custom-css-help" className="text-text-muted mt-3 text-xs">
        CSS only. <code>@import</code>, <code>image-set()</code> and any <code>url()</code> that is
        not an inline <code>data:</code> image are refused, so a theme can never load something from
        another site. If your CSS hides the page, add <code>?safe=1</code> to the address (for
        example <code>/settings/theme?safe=1</code>) to open it without your CSS.
      </p>
    </div>
  )
}
