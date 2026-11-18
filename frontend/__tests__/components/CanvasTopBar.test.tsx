import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import CanvasTopBar from '@/components/layout/CanvasTopBar'
import UserMenu from '@/components/UserMenu'
import { api } from '@/lib/api'
import { setEditMode, setTopBarWeather, useEditMode } from '@/lib/dashboard-store'
import { weatherPayload } from './widgets/fixtures'

vi.mock('next/navigation', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('@/lib/api', () => ({ api: { getCurrentUser: vi.fn(), logout: vi.fn() } }))
vi.mock('@/components/SearchButton', () => ({ default: () => null }))
vi.mock('@/components/ThemeToggle', () => ({ default: () => null }))

const user = { data: { id: 'u1', email: 'a@b.test', name: 'Ada', role: 'user', provider: 'local' } }

function EditModeProbe() {
  return <p>{useEditMode() ? 'editing' : 'viewing'}</p>
}

afterEach(() => {
  act(() => {
    setEditMode(false)
    setTopBarWeather(null)
  })
})

describe('CanvasTopBar weather chip', () => {
  it('is absent without a weather widget and follows its data', async () => {
    vi.mocked(api.getCurrentUser).mockResolvedValue(user as never)
    render(<CanvasTopBar />)
    await screen.findByRole('button', { name: /Ada/ })
    expect(screen.queryByText(/°/)).not.toBeInTheDocument()

    act(() => setTopBarWeather({ location: 'Utrecht', weather: weatherPayload }))
    expect(screen.getByText(`${Math.round(weatherPayload.temperature)}°`)).toBeInTheDocument()
    expect(screen.getByText('Utrecht')).toBeInTheDocument()

    act(() => setTopBarWeather(null))
    expect(screen.queryByText('Utrecht')).not.toBeInTheDocument()
  })
})

describe('UserMenu on the canvas', () => {
  it('toggles the dashboard edit mode from "Edit layout"', async () => {
    vi.mocked(api.getCurrentUser).mockResolvedValue(user as never)
    render(
      <>
        <UserMenu withNavigation />
        <EditModeProbe />
      </>
    )
    const trigger = await screen.findByRole('button', { name: /Ada/ })
    fireEvent.click(trigger)
    fireEvent.click(screen.getByRole('button', { name: 'Edit layout' }))
    expect(screen.getByText('editing')).toBeInTheDocument()
    // The menu closed and the focus is back on its button
    expect(screen.queryByRole('button', { name: 'Edit layout' })).not.toBeInTheDocument()
    expect(trigger).toHaveFocus()

    fireEvent.click(trigger)
    fireEvent.click(screen.getByRole('button', { name: 'Done editing' }))
    expect(screen.getByText('viewing')).toBeInTheDocument()
  })

  it('has no "Edit layout" in the classic header', async () => {
    vi.mocked(api.getCurrentUser).mockResolvedValue(user as never)
    render(<UserMenu />)
    fireEvent.click(await screen.findByRole('button', { name: /Ada/ }))
    expect(screen.queryByRole('button', { name: 'Edit layout' })).not.toBeInTheDocument()
  })

  it('says whether it is open, and Escape closes it and returns the focus', async () => {
    vi.mocked(api.getCurrentUser).mockResolvedValue(user as never)
    render(<UserMenu withNavigation />)
    const trigger = await screen.findByRole('button', { name: /Ada/ })
    expect(trigger).toHaveAttribute('aria-haspopup', 'true')
    expect(trigger).toHaveAttribute('aria-expanded', 'false')

    fireEvent.click(trigger)
    expect(trigger).toHaveAttribute('aria-expanded', 'true')
    const link = screen.getByRole('link', { name: 'Services' })
    expect(document.getElementById(trigger.getAttribute('aria-controls')!)).toContainElement(link)

    link.focus()
    fireEvent.keyDown(link, { key: 'Escape' })
    expect(screen.queryByRole('link', { name: 'Services' })).not.toBeInTheDocument()
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    expect(trigger).toHaveFocus()
  })
})
