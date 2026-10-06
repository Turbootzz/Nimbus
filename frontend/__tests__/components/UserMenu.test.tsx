import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import UserMenu from '@/components/UserMenu'
import { api } from '@/lib/api'

vi.mock('next/navigation', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('@/lib/api', () => ({ api: { getCurrentUser: vi.fn(), logout: vi.fn() } }))

const user = (role: 'admin' | 'user') => ({
  data: { id: 'u1', email: 'a@b.test', name: 'Ada', role, provider: 'local' },
})

describe('UserMenu navigation', () => {
  it('lists the main pages for the canvas layout, admin pages for admins only', async () => {
    vi.mocked(api.getCurrentUser).mockResolvedValue(user('admin') as never)
    const { unmount } = render(<UserMenu withNavigation />)
    fireEvent.click(await screen.findByRole('button', { name: /Ada/ }))
    expect(screen.getByRole('link', { name: 'Services' })).toHaveAttribute('href', '/services')
    expect(screen.getByRole('link', { name: 'Users' })).toBeInTheDocument()
    unmount()

    vi.mocked(api.getCurrentUser).mockResolvedValue(user('user') as never)
    render(<UserMenu withNavigation />)
    fireEvent.click(await screen.findByRole('button', { name: /Ada/ }))
    expect(screen.queryByRole('link', { name: 'Users' })).not.toBeInTheDocument()
  })

  it('has no navigation in the classic header', async () => {
    vi.mocked(api.getCurrentUser).mockResolvedValue(user('admin') as never)
    render(<UserMenu />)
    fireEvent.click(await screen.findByRole('button', { name: /Ada/ }))
    expect(screen.queryByRole('link', { name: 'Services' })).not.toBeInTheDocument()
  })
})
