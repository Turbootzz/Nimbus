import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
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
  scopes: ['read:services', 'read:metrics'],
  read_only: true,
  last_used_at: null,
  expires_at: null,
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
    // Scope badges on the token row; the same scopes also label the form's checkboxes
    const row = screen.getByText('Hearth').closest('div')!
    expect(within(row).getByText('read:services')).toBeInTheDocument()
    expect(within(row).getByText('read:metrics')).toBeInTheDocument()
    expect(within(row).queryByText('mcp')).not.toBeInTheDocument()
    expect(screen.queryByText('expired')).not.toBeInTheDocument()
  })

  it('marks an expired token', async () => {
    vi.mocked(api.getApiTokens).mockResolvedValue({
      data: [{ ...mockToken, expires_at: '2020-01-01T00:00:00Z' }],
    })

    render(<ApiTokensPage />)

    await waitFor(() => {
      expect(screen.getByText('expired')).toBeInTheDocument()
    })
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
      // Read scopes and no expiry by default
      expect(api.createApiToken).toHaveBeenCalledWith({
        name: 'New Token',
        scopes: ['read:services', 'read:metrics', 'read:widgets'],
        expires_at: undefined,
      })
    })
    expect(screen.getByText('nimbus_plaintextsecret')).toBeInTheDocument()
    expect(screen.getByText(/won't be able to see this token again/i)).toBeInTheDocument()
  })

  it('creates a token with the chosen scopes and expiry', async () => {
    const user = userEvent.setup()
    vi.mocked(api.getApiTokens).mockResolvedValue({ data: [] })
    vi.mocked(api.createApiToken).mockResolvedValue({
      data: { token: 'nimbus_plaintextsecret', api_token: mockToken },
    })

    render(<ApiTokensPage />)

    await waitFor(() => {
      expect(screen.getByText(/no api tokens yet/i)).toBeInTheDocument()
    })

    await user.type(screen.getByLabelText(/token name/i), 'Assistant')
    await user.click(screen.getByRole('checkbox', { name: /read metrics/i }))
    await user.click(screen.getByRole('checkbox', { name: /read widgets/i }))
    await user.click(screen.getByRole('checkbox', { name: /mcp/i }))
    await user.selectOptions(screen.getByLabelText(/expires/i), '30')
    const before = Date.now()
    await user.click(screen.getByRole('button', { name: /create token/i }))

    await waitFor(() => {
      expect(api.createApiToken).toHaveBeenCalled()
    })
    const request = vi.mocked(api.createApiToken).mock.calls[0][0]
    expect(request.name).toBe('Assistant')
    expect(request.scopes).toEqual(['read:services', 'mcp'])
    const expiresIn = new Date(request.expires_at!).getTime() - before
    const thirtyDays = 30 * 24 * 60 * 60 * 1000
    expect(expiresIn).toBeGreaterThanOrEqual(thirtyDays)
    expect(expiresIn).toBeLessThan(thirtyDays + 60_000)
  })

  it('cannot create a token without scopes', async () => {
    const user = userEvent.setup()
    vi.mocked(api.getApiTokens).mockResolvedValue({ data: [] })

    render(<ApiTokensPage />)

    await waitFor(() => {
      expect(screen.getByText(/no api tokens yet/i)).toBeInTheDocument()
    })

    await user.type(screen.getByLabelText(/token name/i), 'Nothing')
    for (const name of [/read services/i, /read metrics/i, /read widgets/i]) {
      await user.click(screen.getByRole('checkbox', { name }))
    }

    expect(screen.getByText(/select at least one scope/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /create token/i })).toBeDisabled()
  })

  it('shows an error when loading tokens fails', async () => {
    vi.mocked(api.getApiTokens).mockResolvedValue({
      error: { message: 'Something went wrong' },
    })

    render(<ApiTokensPage />)

    await waitFor(() => {
      expect(screen.getByText('Something went wrong')).toBeInTheDocument()
    })
  })

  it('shows an error when revoking fails and keeps the token listed', async () => {
    const user = userEvent.setup()
    vi.mocked(api.getApiTokens).mockResolvedValue({ data: [mockToken] })
    vi.mocked(api.deleteApiToken).mockResolvedValue({
      error: { message: 'Failed to delete token' },
    })
    vi.stubGlobal(
      'confirm',
      vi.fn(() => true)
    )

    render(<ApiTokensPage />)

    await waitFor(() => {
      expect(screen.getByText('Hearth')).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /revoke/i }))

    await waitFor(() => {
      expect(screen.getByText('Failed to delete token')).toBeInTheDocument()
    })
    expect(screen.getByText('Hearth')).toBeInTheDocument()
  })

  it('revokes a token', async () => {
    const user = userEvent.setup()
    vi.mocked(api.getApiTokens).mockResolvedValue({ data: [mockToken] })
    vi.mocked(api.deleteApiToken).mockResolvedValue({})
    vi.stubGlobal(
      'confirm',
      vi.fn(() => true)
    )

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
