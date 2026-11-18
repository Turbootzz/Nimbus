'use client'

import type {
  CardSize,
  CardScale,
  Kpi,
  Service,
  Tile,
  TileStyle,
  ViewMode,
  Widget,
  WidgetTypeMeta,
} from '@/types'
import ServiceCard from '@/components/ServiceCard'
import ServiceChip from '@/components/ServiceChip'
import SortableTile from '@/components/SortableTile'
import ServicesList from '@/components/ServicesList'
import WidgetCard from '@/components/widgets/WidgetCard'
import { isServiceEffectivelyMonitored, type GroupMonitoringMap } from '@/lib/monitoring'
import { tileKey } from '@/lib/tiles'
import { snapshotKey, type SnapshotMap } from '@/hooks/useDashboardStream'

// Grid classes for different card scales
// Mobile always uses large layout (grid-cols-2) to avoid clutter; the canvas
// layout makes that one column (globals.css, data-tile-grid)
// Scale-specific columns apply from sm breakpoint and up
// The minimum row height is about a 2x1 service card's own height, so a row of
// only widgets matches a row of services. Widget content never grows a row.
const gridClasses: Record<CardScale, string> = {
  small:
    'grid-cols-2 gap-4 auto-rows-[minmax(9rem,auto)] sm:grid-cols-6 sm:gap-2 xl:grid-cols-8 2xl:grid-cols-10',
  medium:
    'grid-cols-2 gap-4 auto-rows-[minmax(10.5rem,auto)] sm:grid-cols-4 sm:gap-3 xl:grid-cols-6 2xl:grid-cols-8',
  large:
    'grid-cols-2 gap-4 auto-rows-[minmax(13rem,auto)] sm:grid-cols-4 xl:grid-cols-6 2xl:grid-cols-8',
}

// Chips sit closer together on denser scales
const chipGaps: Record<CardScale, string> = { small: 'gap-1.5', medium: 'gap-2', large: 'gap-3' }

interface ServicesGridProps {
  tiles: Tile[]
  // Chips show services as one-line links; widgets stay cards below them
  tileStyle?: TileStyle
  openInNewTab: boolean
  enableCardResizing: boolean
  cardScale: CardScale
  viewMode: ViewMode
  isEditMode?: boolean
  onSizeChange?: (id: string, newSize: CardSize) => void
  onEditService?: (service: Service) => void
  groupMonitoringMap?: GroupMonitoringMap
  widgetTypes?: Record<string, WidgetTypeMeta>
  onWidgetSizeChange?: (widget: Widget, size: CardSize) => void
  onEditWidget?: (widget: Widget) => void
  onDeleteWidget?: (widget: Widget) => void
  onRefreshWidget?: (widget: Widget) => void
  snapshots?: SnapshotMap
  // KPI definitions per integration ID, for services linked to one
  integrationKpis?: Record<string, Kpi[]>
}

export default function ServicesGrid({
  tiles,
  tileStyle = 'cards',
  openInNewTab,
  enableCardResizing,
  cardScale,
  viewMode,
  isEditMode = false,
  onSizeChange,
  onEditService,
  groupMonitoringMap,
  widgetTypes,
  onWidgetSizeChange,
  onEditWidget,
  onDeleteWidget,
  onRefreshWidget,
  snapshots,
  integrationKpis,
}: ServicesGridProps) {
  const services = tiles.flatMap((tile) => (tile.kind === 'service' ? [tile.service] : []))
  const widgetTiles = tiles.filter((tile) => tile.kind === 'widget')

  // List view shows services only; widgets need the grid
  const widgetNote = viewMode === 'list' && widgetTiles.length > 0 && (
    <p className="text-text-muted wallpaper-panel mt-3 text-xs">
      {widgetTiles.length} {widgetTiles.length === 1 ? 'widget is' : 'widgets are'} only shown in
      grid view.
    </p>
  )

  if (viewMode === 'list' && tileStyle !== 'chips') {
    return (
      <>
        <ServicesList
          services={services}
          openInNewTab={openInNewTab}
          isEditMode={isEditMode}
          groupMonitoringMap={groupMonitoringMap}
          onEditService={onEditService}
        />
        {widgetNote}
      </>
    )
  }

  const renderTile = (tile: Tile) => {
    const key = tileKey(tile)

    if (tile.kind === 'widget') {
      const widgetProps = {
        widget: tile.widget,
        meta: widgetTypes?.[tile.widget.type],
        snapshot: snapshots?.[snapshotKey('widget', tile.widget.id)],
        openInNewTab,
        cardScale,
        enableCardResizing,
      }
      return isEditMode ? (
        <SortableTile key={key} id={key} cardSize={tile.widget.card_size}>
          {(dragHandleProps) => (
            <WidgetCard
              {...widgetProps}
              isEditMode
              dragHandleProps={dragHandleProps}
              onSizeChange={onWidgetSizeChange}
              onEdit={onEditWidget}
              onDelete={onDeleteWidget}
              onRefresh={onRefreshWidget}
            />
          )}
        </SortableTile>
      ) : (
        <WidgetCard key={key} {...widgetProps} />
      )
    }

    const service = tile.service
    const monitored = isServiceEffectivelyMonitored(service, groupMonitoringMap)
    const integrationId = service.integration_id
    const kpiDefs = integrationId ? integrationKpis?.[integrationId] : undefined
    const kpis =
      integrationId && kpiDefs
        ? { kpis: kpiDefs, snapshot: snapshots?.[snapshotKey('integration', integrationId)] }
        : undefined
    // When card resizing is disabled, services always use 2x1
    const cardSize = enableCardResizing ? service.card_size || '2x1' : '2x1'
    return isEditMode && onSizeChange ? (
      <SortableTile key={key} id={key} cardSize={cardSize}>
        {(dragHandleProps) => (
          <ServiceCard
            service={service}
            openInNewTab={openInNewTab}
            isEditMode
            onSizeChange={onSizeChange}
            onEdit={onEditService}
            dragHandleProps={dragHandleProps}
            enableCardResizing={enableCardResizing}
            cardScale={cardScale}
            isMonitored={monitored}
            kpis={kpis}
          />
        )}
      </SortableTile>
    ) : (
      <ServiceCard
        key={key}
        service={service}
        openInNewTab={openInNewTab}
        enableCardResizing={enableCardResizing}
        cardScale={cardScale}
        isMonitored={monitored}
        kpis={kpis}
      />
    )
  }

  const renderGrid = (gridTiles: Tile[]) => (
    <div
      data-tile-grid
      className={`grid ${gridClasses[cardScale]}`}
      style={{ gridAutoFlow: 'dense' }}
    >
      {gridTiles.map(renderTile)}
    </div>
  )

  if (tileStyle === 'chips') {
    return (
      <>
        <div className={`flex flex-wrap ${chipGaps[cardScale]}`}>
          {services.map((service) => {
            const key = tileKey({ kind: 'service', id: service.id })
            const chipProps = {
              service,
              openInNewTab,
              cardScale,
              isMonitored: isServiceEffectivelyMonitored(service, groupMonitoringMap),
            }
            return isEditMode ? (
              <SortableTile key={key} id={key}>
                {(dragHandleProps) => (
                  <ServiceChip
                    {...chipProps}
                    isEditMode
                    onEdit={onEditService}
                    dragHandleProps={dragHandleProps}
                  />
                )}
              </SortableTile>
            ) : (
              <ServiceChip key={key} {...chipProps} />
            )
          })}
        </div>
        {viewMode === 'grid' && widgetTiles.length > 0 && (
          <div className={services.length > 0 ? 'mt-3' : ''}>{renderGrid(widgetTiles)}</div>
        )}
        {widgetNote}
      </>
    )
  }

  return renderGrid(tiles)
}
