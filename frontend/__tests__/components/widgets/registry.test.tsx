import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import '@testing-library/jest-dom'
import { getWidgetDefinition, widgetConfig, widgetRegistry } from '@/components/widgets/registry'
import { addableTypes } from '@/components/widgets/WidgetModal'
import { backendStaticTypes, makeWidget } from './fixtures'

describe('widget registry', () => {
  it('has a definition for every static backend type', () => {
    for (const meta of backendStaticTypes) {
      expect(getWidgetDefinition(meta.type), meta.type).toBeDefined()
    }
  })

  it('renders every type with its default config', () => {
    for (const definition of Object.values(widgetRegistry)) {
      const widget = makeWidget({ type: definition.type })
      const { unmount } = render(
        <definition.Renderer
          widget={widget}
          config={widgetConfig(definition, widget)}
          cardSize="2x1"
          openInNewTab={false}
        />
      )
      unmount()
    }
  })

  it('renders every config form with its default config', () => {
    for (const definition of Object.values(widgetRegistry)) {
      const { unmount } = render(
        <definition.ConfigForm config={definition.defaultConfig} onChange={() => {}} />
      )
      unmount()
    }
  })

  it('fills in defaults for fields a stored config lacks', () => {
    const definition = getWidgetDefinition('clock')!
    const config = widgetConfig(definition, makeWidget({ type: 'clock', config: { hour12: true } }))
    expect(config).toMatchObject({ hour12: true, date_format: 'short', timezone: '' })
  })

  it('only offers types this version can add', () => {
    const types = [
      ...backendStaticTypes,
      { ...backendStaticTypes[0], type: 'future', name: 'Future' },
      { ...backendStaticTypes[0], type: 'clock', integration_kinds: ['sonarr'] },
    ]
    const addable = addableTypes(types).map((t) => t.type)
    expect(addable).toEqual(['bookmarks', 'clock', 'iframe', 'markdown'])
  })

  it('renders an unknown type message instead of crashing', async () => {
    const { default: WidgetCard } = await import('@/components/widgets/WidgetCard')
    render(<WidgetCard widget={makeWidget({ type: 'future' })} openInNewTab={false} />)
    expect(screen.getByText(/not supported by this version/)).toBeInTheDocument()
  })
})
