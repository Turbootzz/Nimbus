import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import CustomApiWidget from '@/components/widgets/renderers/CustomApiWidget'
import CustomApiForm from '@/components/widgets/forms/CustomApiForm'
import RssWidget from '@/components/widgets/renderers/RssWidget'
import RssForm from '@/components/widgets/forms/RssForm'
import type { CustomApiWidgetConfig, RssPayload } from '@/types'
import { makeSnapshot, makeWidget } from './fixtures'

const apiConfig: CustomApiWidgetConfig = {
  url: 'http://10.0.0.2/stats',
  headers: [],
  fields: [
    { label: 'Users', path: 'users.length', unit: '' },
    { label: 'Disk', path: 'disk.used', unit: '%' },
    { label: 'Name', path: 'name', unit: '' },
  ],
  verify_tls: true,
}

describe('CustomApiWidget', () => {
  const base = { widget: makeWidget({ type: 'custom_api' }), openInNewTab: false }

  it('shows each value with its label and unit', () => {
    const { container } = render(
      <CustomApiWidget
        {...base}
        config={apiConfig}
        cardSize="2x1"
        snapshot={makeSnapshot({
          payload: { kpis: { Users: 1234, Disk: 42 }, missing: ['Name'] },
        })}
      />
    )
    expect(screen.getByText('Users')).toBeInTheDocument()
    expect(screen.getByText((1234).toLocaleString())).toBeInTheDocument()
    expect(screen.getByText('42 %')).toBeInTheDocument()
    expect(screen.getByText('-')).toHaveAttribute('title', 'Nothing found at name')
    expect(container.querySelector('dl')).toHaveClass('grid-cols-3')
  })

  it('uses two columns on a square tile', () => {
    const { container } = render(
      <CustomApiWidget
        {...base}
        config={apiConfig}
        cardSize="1x1"
        snapshot={makeSnapshot({ payload: { kpis: { Users: 1, Disk: 2, Name: 'x' } } })}
      />
    )
    expect(container.querySelector('dl')).toHaveClass('grid-cols-2')
  })

  it('renders nothing without data (the card shows loading)', () => {
    const { container } = render(<CustomApiWidget {...base} config={apiConfig} cardSize="2x1" />)
    expect(container).toBeEmptyDOMElement()
  })
})

describe('CustomApiForm', () => {
  it('edits, adds and removes values and headers', () => {
    const onChange = vi.fn()
    render(<CustomApiForm config={apiConfig} onChange={onChange} />)

    fireEvent.change(screen.getByLabelText('Value 2 path'), { target: { value: 'disk.free' } })
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        fields: [
          apiConfig.fields[0],
          { ...apiConfig.fields[1], path: 'disk.free' },
          apiConfig.fields[2],
        ],
      })
    )

    fireEvent.click(screen.getByText('Add value'))
    expect(onChange.mock.lastCall![0].fields).toHaveLength(4)

    fireEvent.click(screen.getByLabelText('Remove value 1'))
    expect(onChange.mock.lastCall![0].fields).toEqual(apiConfig.fields.slice(1))

    fireEvent.click(screen.getByText('Add header'))
    expect(onChange.mock.lastCall![0].headers).toEqual([{ name: '', value: '' }])

    fireEvent.click(screen.getByRole('switch', { name: 'Verify TLS certificate' }))
    expect(onChange.mock.lastCall![0].verify_tls).toBe(false)
  })

  it('stops at four values', () => {
    const full = { ...apiConfig, fields: [...apiConfig.fields, apiConfig.fields[0]] }
    render(<CustomApiForm config={full} onChange={() => {}} />)
    expect(screen.queryByText('Add value')).not.toBeInTheDocument()
  })
})

const feed: RssPayload = {
  items: [
    {
      title: 'Nimbus 2.0 released',
      link: 'https://news.test/2',
      date: '2026-10-05T10:00:00Z',
      source: 'Homelab News',
    },
    { title: 'No link here', source: 'Other' },
  ],
}

describe('RssWidget', () => {
  const base = {
    widget: makeWidget({ type: 'rss' }),
    config: { feeds: [], limit: 10, verify_tls: true },
    cardSize: '2x2' as const,
  }

  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-10-05T12:00:00Z'))
  })
  afterEach(() => vi.useRealTimers())

  it('lists items with their source and age', () => {
    render(<RssWidget {...base} openInNewTab snapshot={makeSnapshot({ payload: feed })} />)
    const link = screen.getByRole('link', { name: /Nimbus 2.0 released/ })
    expect(link).toHaveAttribute('href', 'https://news.test/2')
    expect(link).toHaveAttribute('target', '_blank')
    expect(screen.getByText(/· 2h ago/)).toBeInTheDocument()
    expect(screen.getByText('No link here')).toBeInTheDocument()
    expect(screen.getAllByRole('link')).toHaveLength(1)
  })

  it('warns about feeds that failed and shows an empty state', () => {
    render(
      <RssWidget
        {...base}
        openInNewTab={false}
        snapshot={makeSnapshot({
          payload: { items: [], failed: ['a.test: unexpected status 404'] },
        })}
      />
    )
    expect(screen.getByText('1 feed could not be loaded')).toHaveAttribute(
      'title',
      'a.test: unexpected status 404'
    )
    expect(screen.getByText('No items yet')).toBeInTheDocument()
  })
})

describe('RssForm', () => {
  it('adds feeds up to three and sets the item count', () => {
    const onChange = vi.fn()
    const { rerender } = render(
      <RssForm config={{ feeds: [''], limit: 10, verify_tls: true }} onChange={onChange} />
    )
    expect(screen.queryByLabelText('Remove feed 1')).not.toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Feed 1 URL'), {
      target: { value: 'https://a.test/feed' },
    })
    expect(onChange).toHaveBeenLastCalledWith({
      feeds: ['https://a.test/feed'],
      limit: 10,
      verify_tls: true,
    })

    fireEvent.change(screen.getByLabelText('Number of items'), { target: { value: '5' } })
    expect(onChange).toHaveBeenLastCalledWith({ feeds: [''], limit: 5, verify_tls: true })

    rerender(
      <RssForm
        config={{ feeds: ['a', 'b', 'c'], limit: 10, verify_tls: true }}
        onChange={onChange}
      />
    )
    expect(screen.queryByText('Add feed')).not.toBeInTheDocument()
    fireEvent.click(screen.getByLabelText('Remove feed 2'))
    expect(onChange).toHaveBeenLastCalledWith({ feeds: ['a', 'c'], limit: 10, verify_tls: true })
  })
})
