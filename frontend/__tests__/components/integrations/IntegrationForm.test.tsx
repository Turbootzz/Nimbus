import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import IntegrationForm from '@/components/integrations/IntegrationForm'
import { api } from '@/lib/api'
import { kinds, makeIntegration } from './fixtures'

vi.mock('@/lib/api', () => ({
  api: {
    createIntegration: vi.fn(),
    updateIntegration: vi.fn(),
    testIntegration: vi.fn(),
    testSavedIntegration: vi.fn(),
  },
}))

describe('IntegrationForm', () => {
  beforeEach(() => vi.clearAllMocks())

  it('tests and creates a new integration', async () => {
    vi.mocked(api.testIntegration).mockResolvedValue({ data: { ok: true, latency_ms: 12 } })
    const created = makeIntegration({ id: 'new' })
    vi.mocked(api.createIntegration).mockResolvedValue({ data: created })
    const onSaved = vi.fn()
    render(<IntegrationForm kinds={kinds} onClose={() => {}} onSaved={onSaved} />)

    fireEvent.change(screen.getByLabelText('App'), { target: { value: 'sonarr' } })
    expect(screen.getByLabelText('Name')).toHaveValue('Sonarr')
    expect(screen.getByLabelText('URL')).toHaveAttribute('placeholder', 'http://192.168.1.10:8989')
    fireEvent.change(screen.getByLabelText('URL'), { target: { value: 'http://sonarr.lan' } })
    fireEvent.change(screen.getByLabelText('API key'), { target: { value: 'secret' } })

    fireEvent.click(screen.getByText('Test connection'))
    expect(await screen.findByText('Connected in 12 ms')).toBeInTheDocument()
    const request = {
      kind: 'sonarr',
      name: 'Sonarr',
      base_url: 'http://sonarr.lan',
      auth_type: 'api_key',
      credentials: { api_key: 'secret' },
      verify_tls: true,
      refresh_seconds: 60,
    }
    expect(api.testIntegration).toHaveBeenCalledWith(request)

    fireEvent.click(screen.getByText('Add Integration', { selector: 'button' }))
    await waitFor(() => expect(onSaved).toHaveBeenCalledWith(created))
    expect(api.createIntegration).toHaveBeenCalledWith(request)
  })

  it('shows a failed test and a save error', async () => {
    vi.mocked(api.testIntegration).mockResolvedValue({
      data: { ok: false, error: 'the API key was rejected', latency_ms: 3 },
    })
    vi.mocked(api.createIntegration).mockResolvedValue({
      error: { message: 'Base URL is required' },
    })
    render(<IntegrationForm kinds={kinds} onClose={() => {}} onSaved={() => {}} />)

    fireEvent.click(screen.getByText('Test connection'))
    expect(
      await screen.findByText('Connection failed: the API key was rejected')
    ).toBeInTheDocument()
    fireEvent.submit(screen.getByLabelText('Name').closest('form')!)
    expect(await screen.findByText('Base URL is required')).toBeInTheDocument()
  })

  it('offers the auth types a kind supports', () => {
    render(<IntegrationForm kinds={kinds} onClose={() => {}} onSaved={() => {}} />)
    fireEvent.change(screen.getByLabelText('App'), { target: { value: 'multi' } })
    expect(screen.getByLabelText('Username')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Sign in with'), { target: { value: 'token' } })
    expect(screen.getByLabelText('Token')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Sign in with'), { target: { value: 'none' } })
    expect(screen.queryByLabelText('Token')).not.toBeInTheDocument()
  })

  it('keeps saved credentials when editing without typing new ones', async () => {
    const integration = makeIntegration()
    vi.mocked(api.updateIntegration).mockResolvedValue({ data: integration })
    vi.mocked(api.testSavedIntegration).mockResolvedValue({ data: { ok: true, latency_ms: 5 } })
    const onSaved = vi.fn()
    render(
      <IntegrationForm
        kinds={kinds}
        integration={integration}
        onClose={() => {}}
        onSaved={onSaved}
      />
    )

    expect(screen.getByLabelText('App')).toBeDisabled()
    expect(screen.getByLabelText('API key')).toHaveAttribute(
      'placeholder',
      'Saved; leave empty to keep'
    )

    fireEvent.click(screen.getByText('Test saved connection'))
    expect(await screen.findByText('Connected in 5 ms')).toBeInTheDocument()
    expect(api.testSavedIntegration).toHaveBeenCalledWith('i1')

    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'TV' } })
    fireEvent.click(screen.getByText('Save Changes'))
    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    const [, request] = vi.mocked(api.updateIntegration).mock.calls[0]
    expect(request).toMatchObject({ name: 'TV' })
    expect(request.credentials).toBeUndefined()
    expect(request.kind).toBeUndefined()
  })

  it('changes basic auth credentials only as a pair when editing', async () => {
    const integration = makeIntegration({ kind: 'multi', auth_type: 'basic' })
    vi.mocked(api.updateIntegration).mockResolvedValue({ data: integration })
    render(
      <IntegrationForm
        kinds={kinds}
        integration={integration}
        onClose={() => {}}
        onSaved={() => {}}
      />
    )

    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'new-pass' } })
    fireEvent.click(screen.getByText('Save Changes'))
    expect(await screen.findByText('Enter the username too')).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Password'), { target: { value: '' } })
    fireEvent.change(screen.getByLabelText('Username'), { target: { value: 'admin' } })
    fireEvent.click(screen.getByText('Save Changes'))
    expect(await screen.findByText(/Enter the password too/)).toBeInTheDocument()
    expect(api.updateIntegration).not.toHaveBeenCalled()

    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'new-pass' } })
    fireEvent.click(screen.getByText('Save Changes'))
    await waitFor(() => expect(api.updateIntegration).toHaveBeenCalled())
    expect(vi.mocked(api.updateIntegration).mock.calls[0][1].credentials).toEqual({
      username: 'admin',
      password: 'new-pass',
    })
  })

  it('keeps saved logins out of the secret fields', () => {
    render(
      <IntegrationForm
        kinds={kinds}
        integration={makeIntegration()}
        onClose={() => {}}
        onSaved={() => {}}
      />
    )
    expect(screen.getByLabelText('API key')).toHaveAttribute('autocomplete', 'new-password')
  })
})
