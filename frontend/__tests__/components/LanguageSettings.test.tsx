import { act, fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import LanguageSettings from '@/components/LanguageSettings'
import LocaleSync from '@/components/LocaleSync'
import type { Language } from '@/i18n/locale'

const refresh = vi.fn()
vi.mock('next/navigation', () => ({ useRouter: () => ({ refresh }) }))

const theme: { language: Language | null | undefined; setLanguage: (v: Language | null) => void } =
  { language: undefined, setLanguage: vi.fn() }
vi.mock('@/contexts/ThemeContext', () => ({ useTheme: () => theme }))

// The language in the cookie, if any
const cookie = () => document.cookie.match(/NEXT_LOCALE=(\w+)/)?.[1]

describe('LanguageSettings', () => {
  it('offers automatic first, then each language in its own name', () => {
    theme.language = null
    render(<LanguageSettings />)
    const select = screen.getByLabelText('Language')
    const options = Array.from(select.querySelectorAll('option')).map((o) => o.textContent)
    expect(options).toEqual([
      'Automatic (browser)',
      'English',
      'Nederlands',
      'Deutsch',
      'Français',
      'Español',
    ])
    fireEvent.change(select, { target: { value: 'nl' } })
    expect(theme.setLanguage).toHaveBeenCalledWith('nl')
    fireEvent.change(select, { target: { value: '' } })
    expect(theme.setLanguage).toHaveBeenLastCalledWith(null)
  })
})

describe('LocaleSync', () => {
  beforeEach(() => {
    refresh.mockClear()
    document.cookie = 'NEXT_LOCALE=; path=/; max-age=0'
  })

  it('waits for the preferences', () => {
    theme.language = undefined
    render(<LocaleSync />)
    expect(cookie()).toBeUndefined()
    expect(refresh).not.toHaveBeenCalled()
  })

  it('writes the saved language and renders the page again, once', () => {
    theme.language = 'de'
    const { rerender } = render(<LocaleSync />)
    expect(cookie()).toBe('de')
    expect(refresh).toHaveBeenCalledTimes(1)
    act(() => rerender(<LocaleSync />))
    expect(refresh).toHaveBeenCalledTimes(1)
  })

  it('drops the cookie when the user goes back to automatic', () => {
    document.cookie = 'NEXT_LOCALE=fr; path=/'
    theme.language = null
    render(<LocaleSync />)
    expect(cookie()).toBeUndefined()
    expect(refresh).toHaveBeenCalledTimes(1)
  })

  it('leaves an automatic user without a cookie alone', () => {
    theme.language = null
    render(<LocaleSync />)
    expect(refresh).not.toHaveBeenCalled()
  })
})
