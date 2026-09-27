import type { ComponentType } from 'react'
import {
  BookmarkIcon,
  ClockIcon,
  DocumentTextIcon,
  GlobeAltIcon,
} from '@heroicons/react/24/outline'
import type {
  BookmarksWidgetConfig,
  CardSize,
  ClockWidgetConfig,
  EmbedWidgetConfig,
  NoteWidgetConfig,
  Widget,
} from '@/types'
import ClockWidget from '@/components/widgets/renderers/ClockWidget'
import NoteWidget from '@/components/widgets/renderers/NoteWidget'
import BookmarksWidget from '@/components/widgets/renderers/BookmarksWidget'
import EmbedWidget from '@/components/widgets/renderers/EmbedWidget'
import ClockForm from '@/components/widgets/forms/ClockForm'
import NoteForm from '@/components/widgets/forms/NoteForm'
import BookmarksForm from '@/components/widgets/forms/BookmarksForm'
import EmbedForm from '@/components/widgets/forms/EmbedForm'

export interface WidgetRendererProps<C> {
  widget: Widget
  config: C
  cardSize: CardSize
  openInNewTab: boolean
}

export interface WidgetFormProps<C> {
  config: C
  onChange: (config: C) => void
  disabled?: boolean
}

// Frontend half of a widget type. Sizes, category and integration needs
// come from the backend (GET /widgets/types).
export interface WidgetDefinition<C = Record<string, unknown>> {
  type: string
  label: string
  description: string
  icon: ComponentType<{ className?: string }>
  defaultConfig: C
  Renderer: ComponentType<WidgetRendererProps<C>>
  ConfigForm: ComponentType<WidgetFormProps<C>>
}

// Erases the config type so every definition fits in one map
function defineWidget<C>(definition: WidgetDefinition<C>): WidgetDefinition {
  return definition as unknown as WidgetDefinition
}

const definitions: WidgetDefinition[] = [
  defineWidget<ClockWidgetConfig>({
    type: 'clock',
    label: 'Clock',
    description: 'Time and date, in any timezone',
    icon: ClockIcon,
    defaultConfig: { timezone: '', hour12: false, show_seconds: false, date_format: 'short' },
    Renderer: ClockWidget,
    ConfigForm: ClockForm,
  }),
  defineWidget<NoteWidgetConfig>({
    type: 'markdown',
    label: 'Note',
    description: 'Text with Markdown formatting',
    icon: DocumentTextIcon,
    defaultConfig: { content: '' },
    Renderer: NoteWidget,
    ConfigForm: NoteForm,
  }),
  defineWidget<BookmarksWidgetConfig>({
    type: 'bookmarks',
    label: 'Bookmarks',
    description: 'A compact list of links',
    icon: BookmarkIcon,
    defaultConfig: { items: [] },
    Renderer: BookmarksWidget,
    ConfigForm: BookmarksForm,
  }),
  defineWidget<EmbedWidgetConfig>({
    type: 'iframe',
    label: 'Embed',
    description: 'Show a web page inside a tile',
    icon: GlobeAltIcon,
    defaultConfig: { url: '', height: 300 },
    Renderer: EmbedWidget,
    ConfigForm: EmbedForm,
  }),
]

export const widgetRegistry: Record<string, WidgetDefinition> = Object.fromEntries(
  definitions.map((definition) => [definition.type, definition])
)

export function getWidgetDefinition(type: string): WidgetDefinition | undefined {
  return widgetRegistry[type]
}

// Stored config with defaults for any field it lacks
export function widgetConfig(definition: WidgetDefinition, widget: Widget) {
  return { ...definition.defaultConfig, ...widget.config }
}
