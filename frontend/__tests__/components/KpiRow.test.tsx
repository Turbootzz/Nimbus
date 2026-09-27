import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import '@testing-library/jest-dom'
import KpiRow from '@/components/KpiRow'
import type { Snapshot } from '@/types'

const kpis = [
  { key: 'wanted', label: 'Wanted' },
  { key: 'queued', label: 'Queued' },
  { key: 'size', label: 'Size', unit: 'GB' },
]
const snapshot = (overrides: Partial<Snapshot> = {}): Snapshot => ({
  source_kind: 'integration',
  source_id: 'i1',
  payload: { kpis: { wanted: 1234, queued: 0, size: 12.5 } },
  fetched_at: '2026-09-27T12:00:00Z',
  stale: false,
  ...overrides,
})

describe('KpiRow', () => {
  it('shows each KPI with its label and unit', () => {
    render(<KpiRow kpis={kpis} snapshot={snapshot()} />)
    expect(screen.getByText('Wanted')).toBeInTheDocument()
    expect(screen.getByText((1234).toLocaleString())).toBeInTheDocument()
    expect(screen.getByText('0')).toBeInTheDocument()
    expect(screen.getByText(`${(12.5).toLocaleString()} GB`)).toBeInTheDocument()
  })

  it('shows loading blocks before the first snapshot', () => {
    render(<KpiRow kpis={kpis} />)
    expect(screen.getAllByText('Loading')).toHaveLength(3)
  })

  it('shows dashes and the error when there is no data', () => {
    const { container } = render(
      <KpiRow
        kpis={kpis}
        snapshot={snapshot({ payload: null, error: 'the API key was rejected' })}
      />
    )
    expect(screen.getAllByText('-')).toHaveLength(3)
    expect(container.firstChild).toHaveAttribute(
      'title',
      'Last update failed: the API key was rejected'
    )
  })

  it('marks old data and shows at most four values', () => {
    const many = [1, 2, 3, 4, 5].map((n) => ({ key: `k${n}`, label: `K${n}` }))
    const { container } = render(<KpiRow kpis={many} snapshot={snapshot({ stale: true })} />)
    expect(screen.queryByText('K5')).not.toBeInTheDocument()
    expect(container.firstChild).toHaveAttribute('title', 'This data may be out of date')
  })

  it('renders nothing without KPIs', () => {
    const { container } = render(<KpiRow kpis={[]} />)
    expect(container).toBeEmptyDOMElement()
  })
})
