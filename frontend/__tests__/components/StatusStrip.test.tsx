import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import StatusStrip from '@/components/StatusStrip'
import StatusStripSettings from '@/components/StatusStripSettings'
import type { Integration, IntegrationKindMeta, StatusStrip as Strip, Widget } from '@/types'
import { makeWidget } from './widgets/fixtures'

const piholeId = '11111111-1111-1111-1111-111111111111'
const apiWidgetId = '22222222-2222-2222-2222-222222222222'
const integrations = [{ id: piholeId, kind: 'pihole', name: 'Pi-hole' }] as Integration[]
const kinds: IntegrationKindMeta[] = [
  {
    kind: 'pihole',
    name: 'Pi-hole',
    auth_types: ['none'],
    kpis: [
      { key: 'blocked_percent', label: 'Blocked', unit: '%' },
      { key: 'queries', label: 'Queries' },
    ],
  },
]
const widgets: Widget[] = [
  makeWidget({
    id: apiWidgetId,
    type: 'custom_api',
    title: 'NAS',
    config: { fields: [{ label: 'CPU', path: 'cpu', unit: '%' }] },
  }),
]
const snapshot = (payload: unknown) => ({
  source_kind: 'integration' as const,
  source_id: '',
  payload,
  fetched_at: '',
  stale: false,
})

describe('StatusStrip', () => {
  const strip: Strip = {
    enabled: true,
    chips: [
      { source: 'integration', id: piholeId, kpi: 'blocked_percent' },
      { source: 'widget', id: apiWidgetId, kpi: 'CPU' },
      { source: 'integration', id: '33333333-3333-3333-3333-333333333333', kpi: 'queries' },
    ],
  }
  const snapshots = {
    [`integration:${piholeId}`]: snapshot({ kpis: { blocked_percent: 26 } }),
    [`widget:${apiWidgetId}`]: snapshot({ kpis: { CPU: 12.5 } }),
  }

  it('shows the pill and the chips, leaving out removed sources', () => {
    render(
      <StatusStrip
        strip={strip}
        snapshots={snapshots}
        integrations={integrations}
        kinds={kinds}
        widgets={widgets}
        up={3}
        down={0}
      />
    )
    expect(screen.getByText('Operational')).toBeInTheDocument()
    expect(screen.getByText('Pi-hole · Blocked')).toBeInTheDocument()
    expect(screen.getByText('26 %')).toBeInTheDocument()
    expect(screen.getByText('NAS · CPU')).toBeInTheDocument()
    expect(screen.getByText((12.5).toLocaleString() + ' %')).toBeInTheDocument()
    expect(screen.getAllByText(/·/)).toHaveLength(2)
  })

  it('mutes old data and hides the pill while no service status is known', () => {
    render(
      <StatusStrip
        strip={strip}
        snapshots={{
          [`integration:${piholeId}`]: {
            ...snapshot({ kpis: { blocked_percent: 26 } }),
            stale: true,
            error: 'timeout',
          },
        }}
        integrations={integrations}
        kinds={kinds}
        widgets={widgets}
        up={0}
        down={0}
      />
    )
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    expect(screen.getByText('26 %')).toHaveClass('text-text-muted')
    expect(screen.getByText('26 %').parentElement).toHaveAttribute(
      'title',
      'Last update failed: timeout'
    )
  })

  it('counts services that are down and shows - before data arrives', () => {
    render(
      <StatusStrip
        strip={strip}
        snapshots={{}}
        integrations={integrations}
        kinds={kinds}
        widgets={widgets}
        up={1}
        down={2}
      />
    )
    expect(screen.getByText('2 down')).toBeInTheDocument()
    expect(screen.getAllByText('-')).toHaveLength(2)
  })
})

const theme = {
  statusStrip: { enabled: true, chips: [] } as Strip,
  setStatusStrip: vi.fn(),
}
vi.mock('@/contexts/ThemeContext', () => ({ useTheme: () => theme }))
vi.mock('@/hooks/useIntegrations', () => ({
  useIntegrations: () => ({ integrations, kinds, isLoading: false, error: null }),
}))
vi.mock('@/lib/api', () => ({
  api: { getWidgets: vi.fn(() => Promise.resolve({ data: widgets })) },
}))

describe('StatusStripSettings', () => {
  beforeEach(() => {
    theme.statusStrip = { enabled: true, chips: [] }
    theme.setStatusStrip.mockClear()
  })

  it('adds, changes and removes numbers', async () => {
    const { rerender } = render(<StatusStripSettings />)
    await waitFor(() => expect(screen.getByText('Add number')).toBeInTheDocument())

    fireEvent.click(screen.getByText('Add number'))
    expect(theme.setStatusStrip).toHaveBeenLastCalledWith({
      enabled: true,
      chips: [{ source: 'integration', id: piholeId, kpi: 'blocked_percent' }],
    })

    theme.statusStrip = {
      enabled: true,
      chips: [{ source: 'integration', id: piholeId, kpi: 'blocked_percent' }],
    }
    rerender(<StatusStripSettings />)
    await screen.findByRole('option', { name: 'NAS · CPU' })
    fireEvent.change(screen.getByLabelText('Number 1'), {
      target: { value: `widget:${apiWidgetId}:CPU` },
    })
    expect(theme.setStatusStrip).toHaveBeenLastCalledWith({
      enabled: true,
      chips: [{ source: 'widget', id: apiWidgetId, kpi: 'CPU' }],
    })

    fireEvent.click(screen.getByLabelText('Remove number 1'))
    expect(theme.setStatusStrip).toHaveBeenLastCalledWith({ enabled: true, chips: [] })
  })

  it('keeps a chip whose source is gone visible so it can be removed', async () => {
    theme.statusStrip = {
      enabled: true,
      chips: [{ source: 'integration', id: '33333333-3333-3333-3333-333333333333', kpi: 'x' }],
    }
    render(<StatusStripSettings />)
    expect(await screen.findByLabelText('Number 1')).toHaveDisplayValue(
      'Removed integration or widget'
    )
  })
})
