import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import ApiTokensPage from '@/app/(dashboard)/settings/api-tokens/page'
import { api } from '@/lib/api'
import type { ApiToken } from '@/types'

vi.mock('@/lib/api', () => ({
  api: {
    getApiTokens: vi.fn(),
    createApiToken: vi.fn(),
    deleteApiToken: vi.fn(),
  },
}))

const mockToken: ApiToken = {
  id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1',
  name: 'Hearth',
  token_prefix: 'nimbus_abcde',
  read_only: true,
  last_used_at: null,
  created_at: '2026-07-28T10:00:00Z',
}

describe('ApiTokensPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders tokens from the API', async () => {
    vi.mocked(api.getApiTokens).mockResolvedValue({ data: [mockToken] })

    render(<ApiTokensPage />)

    await waitFor(() => {
      expect(screen.getByText('Hearth')).toBeInTheDocument()
    })
    expect(screen.getByText(/nimbus_abcde/)).toBeInTheDocument()
    // Badge on the token row (the Toggle label is "Read-only")
    expect(screen.getByText('read-only')).toBeInTheDocument()
  })

  it('shows empty state when there are no tokens', async () => {
    vi.mocked(api.getApiTokens).mockResolvedValue({ data: [] })

    render(<ApiTokensPage />)

    await waitFor(() => {
      expect(screen.getByText(/no api tokens yet/i)).toBeInTheDocument()
    })
  })

  it('creates a token and reveals the plaintext once', async () => {
    const user = userEvent.setup()
    vi.mocked(api.getApiTokens).mockResolvedValue({ data: [] })
    vi.mocked(api.createApiToken).mockResolvedValue({
      data: {
        token: 'nimbus_plaintextsecret',
        api_token: { ...mockToken, name: 'New Token' },
      },
    })

    render(<ApiTokensPage />)

    await waitFor(() => {
      expect(screen.getByText(/no api tokens yet/i)).toBeInTheDocument()
    })

    await user.type(screen.getByPlaceholderText(/token name/i), 'New Token')
    await user.click(screen.getByRole('button', { name: /create token/i }))

    await waitFor(() => {
      expect(api.createApiToken).toHaveBeenCalledWith({ name: 'New Token', read_only: true })
    })
    expect(screen.getByText('nimbus_plaintextsecret')).toBeInTheDocument()
    expect(screen.getByText(/won't be able to see this token again/i)).toBeInTheDocument()
  })

  it('revokes a token', async () => {
    const user = userEvent.setup()
    vi.mocked(api.getApiTokens).mockResolvedValue({ data: [mockToken] })
    vi.mocked(api.deleteApiToken).mockResolvedValue({})
    vi.stubGlobal('confirm', vi.fn(() => true))

    render(<ApiTokensPage />)

    await waitFor(() => {
      expect(screen.getByText('Hearth')).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /revoke/i }))

    await waitFor(() => {
      expect(api.deleteApiToken).toHaveBeenCalledWith(mockToken.id)
    })
    expect(screen.queryByText('Hearth')).not.toBeInTheDocument()
  })
})
