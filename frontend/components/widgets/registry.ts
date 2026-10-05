import type { ComponentType } from 'react'
import {
  BookmarkIcon,
  ClockIcon,
  CloudIcon,
  CodeBracketIcon,
  CpuChipIcon,
  CubeIcon,
  DocumentTextIcon,
  GlobeAltIcon,
  RssIcon,
} from '@heroicons/react/24/outline'
import type {
  BookmarksWidgetConfig,
  CardSize,
  ClockWidgetConfig,
  CustomApiWidgetConfig,
  DockerContainersWidgetConfig,
  EmbedWidgetConfig,
  NoteWidgetConfig,
  RssWidgetConfig,
  Snapshot,
  SystemStatsWidgetConfig,
  WeatherWidgetConfig,
  Widget,
} from '@/types'
import ClockWidget from '@/components/widgets/renderers/ClockWidget'
import NoteWidget from '@/components/widgets/renderers/NoteWidget'
import BookmarksWidget from '@/components/widgets/renderers/BookmarksWidget'
import EmbedWidget from '@/components/widgets/renderers/EmbedWidget'
import WeatherWidget from '@/components/widgets/renderers/WeatherWidget'
import CustomApiWidget from '@/components/widgets/renderers/CustomApiWidget'
import RssWidget from '@/components/widgets/renderers/RssWidget'
import SystemStatsWidget from '@/components/widgets/renderers/SystemStatsWidget'
import DockerContainersWidget from '@/components/widgets/renderers/DockerContainersWidget'
import ClockForm from '@/components/widgets/forms/ClockForm'
import NoteForm from '@/components/widgets/forms/NoteForm'
import BookmarksForm from '@/components/widgets/forms/BookmarksForm'
import EmbedForm from '@/components/widgets/forms/EmbedForm'
import WeatherForm from '@/components/widgets/forms/WeatherForm'
import CustomApiForm from '@/components/widgets/forms/CustomApiForm'
import RssForm from '@/components/widgets/forms/RssForm'
import SystemStatsForm from '@/components/widgets/forms/SystemStatsForm'
import DockerContainersForm from '@/components/widgets/forms/DockerContainersForm'

export interface WidgetRendererProps<C> {
  widget: Widget
  config: C
  cardSize: CardSize
  openInNewTab: boolean
  // Latest data of a polled widget; WidgetCard handles loading and errors
  // before there is a payload
  snapshot?: Snapshot
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
  // The backend polls it; the renderer needs a snapshot
  polled?: boolean
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
  defineWidget<WeatherWidgetConfig>({
    type: 'weather',
    label: 'Weather',
    description: 'Current weather and a short forecast',
    icon: CloudIcon,
    polled: true,
    defaultConfig: { location: '', units: 'metric' },
    Renderer: WeatherWidget,
    ConfigForm: WeatherForm,
  }),
  defineWidget<RssWidgetConfig>({
    type: 'rss',
    label: 'RSS',
    description: 'Latest items from up to three feeds',
    icon: RssIcon,
    polled: true,
    defaultConfig: { feeds: [''], limit: 10, verify_tls: true },
    Renderer: RssWidget,
    ConfigForm: RssForm,
  }),
  defineWidget<CustomApiWidgetConfig>({
    type: 'custom_api',
    label: 'Custom API',
    description: 'Up to four values from any JSON API',
    icon: CodeBracketIcon,
    polled: true,
    defaultConfig: {
      url: '',
      headers: [],
      fields: [{ label: '', path: '', unit: '' }],
      verify_tls: true,
    },
    Renderer: CustomApiWidget,
    ConfigForm: CustomApiForm,
  }),
  defineWidget<SystemStatsWidgetConfig>({
    type: 'system_stats',
    label: 'System stats',
    description: 'CPU, memory, disk and uptime of the Nimbus server',
    icon: CpuChipIcon,
    polled: true,
    defaultConfig: { disk_path: '/' },
    Renderer: SystemStatsWidget,
    ConfigForm: SystemStatsForm,
  }),
  defineWidget<DockerContainersWidgetConfig>({
    type: 'docker_containers',
    label: 'Docker containers',
    description: 'Containers of a Docker integration and their state',
    icon: CubeIcon,
    polled: true,
    defaultConfig: { hide_stopped: false },
    Renderer: DockerContainersWidget,
    ConfigForm: DockerContainersForm,
  }),
  defineWidget<EmbedWidgetConfig>({
    type: 'iframe',
    label: 'Embed',
    description: 'Show a web page inside a tile',
    icon: GlobeAltIcon,
    defaultConfig: { url: '' },
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
