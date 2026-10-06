import { render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import '@testing-library/jest-dom'
import SystemStatsWidget, { formatUptime } from '@/components/widgets/renderers/SystemStatsWidget'
import DockerContainersWidget from '@/components/widgets/renderers/DockerContainersWidget'
import type { DockerContainersPayload, SystemStatsPayload } from '@/types'
import { makeSnapshot, makeWidget } from './fixtures'

const GB = 1024 ** 3
const stats: SystemStatsPayload = {
  cpu_percent: 12.4,
  memory_percent: 78,
  memory_used: 12.6 * GB,
  memory_total: 16 * GB,
  disk_percent: 93.2,
  disk_used: 466 * GB,
  disk_total: 500 * GB,
  uptime_seconds: 3 * 86400 + 5 * 3600,
}

describe('SystemStatsWidget', () => {
  const base = {
    widget: makeWidget({ type: 'system_stats' }),
    config: { disk_path: '/' },
    openInNewTab: false,
    snapshot: makeSnapshot({ payload: stats }),
  }

  it('shows a bar per resource, colored by how full it is', () => {
    render(<SystemStatsWidget {...base} cardSize="2x1" />)
    expect(screen.getByRole('meter', { name: 'CPU' })).toHaveAttribute('aria-valuenow', '12')
    expect(screen.getByText('13 GB / 16 GB')).toBeInTheDocument()
    expect(screen.getByText('466 GB / 500 GB')).toBeInTheDocument()
    expect(screen.getByRole('meter', { name: 'RAM' }).firstChild).toHaveClass('bg-warning')
    expect(screen.getByRole('meter', { name: 'Disk' }).firstChild).toHaveClass('bg-error')
    expect(screen.getByText('Up 3d 5h')).toBeInTheDocument()
  })

  it('only shows percentages on a small tile', () => {
    render(<SystemStatsWidget {...base} cardSize="1x1" />)
    expect(screen.getByText('78%')).toBeInTheDocument()
    expect(screen.queryByText(/GB/)).not.toBeInTheDocument()
    expect(screen.queryByText(/^Up/)).not.toBeInTheDocument()
  })

  it('formats uptime', () => {
    expect(formatUptime(59)).toBe('0m')
    expect(formatUptime(3 * 3600 + 120)).toBe('3h 2m')
    expect(formatUptime(86400)).toBe('1d 0h')
  })
})

describe('DockerContainersWidget', () => {
  const payload: DockerContainersPayload = {
    containers: [
      { id: 'a1', name: 'web', image: 'nginx', state: 'running', status: 'Up 2 days' },
      { id: 'b2', name: '', image: 'busybox', state: 'exited', status: 'Exited (0) 1 hour ago' },
    ],
    running: 1,
    total: 3,
  }
  const base = {
    widget: makeWidget({ type: 'docker_containers' }),
    config: { hide_stopped: false },
    openInNewTab: false,
    snapshot: makeSnapshot({ payload }),
  }

  it('lists containers with their state and status', () => {
    render(<DockerContainersWidget {...base} cardSize="2x2" />)
    expect(screen.getByText('1 of 3 running')).toBeInTheDocument()
    const items = screen.getAllByRole('listitem')
    expect(within(items[0]).getByText('web')).toBeInTheDocument()
    expect(within(items[0]).getByText(', running')).toHaveClass('sr-only')
    expect(within(items[0]).getByText('Up 2 days')).toBeInTheDocument()
    expect(within(items[1]).getByText('busybox')).toBeInTheDocument() // no name
    expect(items[1]).toHaveAttribute('title', 'busybox · Exited (0) 1 hour ago')
  })

  it('leaves the status out on a narrow tile', () => {
    render(<DockerContainersWidget {...base} cardSize="1x2" />)
    expect(screen.queryByText('Up 2 days')).not.toBeInTheDocument()
  })
})
