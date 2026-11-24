import { act, fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CustomCssSettings from '@/components/CustomCssSettings'

const theme = { customCss: '', saveTheme: vi.fn() }
vi.mock('@/contexts/ThemeContext', () => ({ useTheme: () => theme }))

// Types the CSS, saves it and returns the answer of the save
function typeAndSave(css: string) {
  let answer: (problem: string | null) => void = () => {}
  theme.saveTheme.mockReturnValue(new Promise((resolve) => (answer = resolve)))
  render(<CustomCssSettings />)
  fireEvent.change(screen.getByRole('textbox'), { target: { value: css } })
  fireEvent.click(screen.getByRole('button'))
  expect(theme.saveTheme).toHaveBeenCalledWith({ custom_css: css })
  return async () => {
    theme.customCss = css // what the server stored
    await act(async () => answer(null))
  }
}

describe('CustomCssSettings', () => {
  beforeEach(() => {
    theme.customCss = ''
  })

  it('follows the stored CSS after a save', async () => {
    const saved = typeAndSave('a { color: red }')
    await saved()

    expect(screen.getByRole('textbox')).toHaveValue('a { color: red }')
    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('keeps what was typed while the CSS was saving', async () => {
    const saved = typeAndSave('a { color: red }')
    fireEvent.change(screen.getByRole('textbox'), {
      target: { value: 'a { color: red } b { color: blue }' },
    })
    await saved()

    expect(screen.getByRole('textbox')).toHaveValue('a { color: red } b { color: blue }')
    expect(screen.getByRole('button')).toBeEnabled()
  })
})
