'use client'

import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { ServerIcon, PlusIcon } from '@heroicons/react/24/outline'
import Link from 'next/link'
import {
  DndContext,
  DragEndEvent,
  DragStartEvent,
  DragOverlay,
  PointerSensor,
  TouchSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import { arrayMove, SortableContext, rectSortingStrategy } from '@dnd-kit/sortable'
import { api } from '@/lib/api'
import { mergeHealthData, mergeServicesHealth, shouldSkipPoll } from '@/lib/polling'
import { snapshotKey, useDashboardStream } from '@/hooks/useDashboardStream'
import { kpisForIntegration } from '@/lib/kpis'
import { useIntegrations } from '@/hooks/useIntegrations'
import type {
  Service,
  CardSize,
  Group,
  GroupUpdateRequest,
  ServiceStatusEvent,
  Tile,
  TileStyle,
  WeatherPayload,
  Widget,
  WidgetTypeMeta,
} from '@/types'
import { useTheme } from '@/contexts/ThemeContext'
import { useLayoutMode } from '@/lib/layout-store'
import { setEditMode, setTopBarWeather, toggleEditMode, useEditMode } from '@/lib/dashboard-store'
import StatusStrip from '@/components/StatusStrip'
import DiscoveredTray from '@/components/DiscoveredTray'
import ServiceCard from '@/components/ServiceCard'
import ServiceChip from '@/components/ServiceChip'
import ServiceListItem from '@/components/ServiceListItem'
import GroupForm from '@/components/GroupForm'
import DashboardHeader from '@/components/DashboardHeader'
import DashboardStats from '@/components/DashboardStats'
import GroupSections from '@/components/GroupSections'
import ServicesGrid from '@/components/ServicesGrid'
import WidgetCard from '@/components/widgets/WidgetCard'
import WidgetModal from '@/components/widgets/WidgetModal'
import ServiceModal from '@/components/ServiceModal'
import ConfirmDialog from '@/components/ui/ConfirmDialog'
import { isServiceEffectivelyMonitored } from '@/lib/monitoring'
import { createCollisionDetection } from '@/lib/tile-collision'
import {
  GROUP_DROP_PREFIX,
  filterTilesByGroup,
  groupTilesBySection,
  mergeTiles,
  moveTile,
  reorderTiles,
  resolveGroupDisplay,
  sortableOrder,
  splitTiles,
  tileGroupId,
  tileKey,
  tileStyleOf,
  toTilePositions,
  undoMove,
  withGroup,
} from '@/lib/tiles'

function toStringId(id: string | number): string {
  return typeof id === 'string' ? id : String(id)
}

export default function DashboardPage() {
  const {
    openInNewTab,
    enableCardResizing,
    enableServiceGrouping,
    cardScale,
    viewMode,
    statusStrip,
    groupDisplay,
    loading: preferencesLoading,
  } = useTheme()
  const layoutMode = useLayoutMode()
  const [services, setServices] = useState<Service[]>([])
  const [widgets, setWidgets] = useState<Widget[]>([])
  // Reordering saves every tile, so it waits until widgets are loaded
  const [widgetsLoaded, setWidgetsLoaded] = useState(false)
  const [widgetTypes, setWidgetTypes] = useState<WidgetTypeMeta[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [selectedGroupId, setSelectedGroupId] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  // Shared with the canvas top bar, whose user menu also toggles it
  const isEditMode = useEditMode()
  const [activeId, setActiveId] = useState<string | null>(null)
  const [resizingId, setResizingId] = useState<string | null>(null)
  const [isDraggingTab, setIsDraggingTab] = useState(false)

  // Track last fetch time for polling coordination
  const lastPollTime = useRef<number>(Date.now())

  // Group form modal state
  const [showGroupForm, setShowGroupForm] = useState(false)
  const [editingGroup, setEditingGroup] = useState<Group | null>(null)
  const [groupFormLoading, setGroupFormLoading] = useState(false)

  // Delete confirmation state
  const [deletingGroup, setDeletingGroup] = useState<Group | null>(null)
  const [deleteGroupServices, setDeleteGroupServices] = useState(false)

  // Widget modal: open for a new widget, or for editing an existing one
  const [showWidgetModal, setShowWidgetModal] = useState(false)
  const [editingWidget, setEditingWidget] = useState<Widget | null>(null)
  const [deletingWidget, setDeletingWidget] = useState<Widget | null>(null)

  // Service being edited in the modal
  const [editingService, setEditingService] = useState<Service | null>(null)

  // Services and widgets in one grid, ordered by their shared positions
  const tiles = useMemo(() => mergeTiles(services, widgets), [services, widgets])

  // Sections stack every group on the page; tabs show one group at a time
  const sectioned =
    enableServiceGrouping &&
    groups.length > 0 &&
    resolveGroupDisplay(groupDisplay, layoutMode) === 'sections'
  const tabbed = enableServiceGrouping && !sectioned

  // Tiles of the selected group; the default group also shows ungrouped
  // tiles (backwards compat). Sections, and no grouping, show everything.
  const filteredTiles = useMemo(
    () => (tabbed ? filterTilesByGroup(tiles, selectedGroupId, groups) : tiles),
    [tiles, selectedGroupId, tabbed, groups]
  )

  const filteredServices = useMemo(
    () => filteredTiles.flatMap((t) => (t.kind === 'service' ? [t.service] : [])),
    [filteredTiles]
  )

  // Tiles that can be dragged: list view only shows services
  const sortableTiles = useMemo(
    () => (viewMode === 'list' ? filteredTiles.filter((t) => t.kind === 'service') : filteredTiles),
    [filteredTiles, viewMode]
  )

  const sections = useMemo(
    () => (sectioned ? groupTilesBySection(filteredTiles, groups) : []),
    [sectioned, filteredTiles, groups]
  )
  // Sections with only the tiles that can be dragged, for drops
  const sortableSections = useMemo(
    () => (sectioned ? groupTilesBySection(sortableTiles, groups) : []),
    [sectioned, sortableTiles, groups]
  )

  const widgetTypeMap = useMemo(
    () => Object.fromEntries(widgetTypes.map((t) => [t.type, t])),
    [widgetTypes]
  )

  // Create a map of group IDs to their monitoring status
  const groupMonitoringMap = useMemo(() => {
    const map = new Map<string, boolean>()
    groups.forEach((g) => {
      map.set(g.id, g.monitoring_enabled)
    })
    return map
  }, [groups])

  // Only effectively monitored services count (both service and group flag on)
  const monitoredOnly = useCallback(
    (list: Service[]) => list.filter((s) => isServiceEffectivelyMonitored(s, groupMonitoringMap)),
    [groupMonitoringMap]
  )

  // Up and down across all groups, for the status strip pill
  const allStatuses = useMemo(() => {
    const monitored = monitoredOnly(services)
    return {
      up: monitored.filter((s) => s.status === 'online').length,
      down: monitored.filter((s) => s.status === 'offline').length,
    }
  }, [services, monitoredOnly])

  // Calculate stats from filtered services using useMemo for efficiency
  // Only count monitored services for online/offline/response stats
  const stats = useMemo(() => {
    const servicesToCount = enableServiceGrouping ? filteredServices : services
    const monitoredServices = monitoredOnly(servicesToCount)

    const online = monitoredServices.filter((s) => s.status === 'online').length
    const offline = monitoredServices.filter((s) => s.status === 'offline').length
    const responseTimes = monitoredServices
      .filter(
        (s) => s.status === 'online' && s.response_time !== undefined && s.response_time !== null
      )
      .map((s) => s.response_time as number)
    const avgResponseTime =
      responseTimes.length > 0
        ? Math.round(responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length)
        : 0

    return {
      total: servicesToCount.length,
      online,
      offline,
      avgResponseTime,
    }
  }, [services, filteredServices, enableServiceGrouping, monitoredOnly])

  // Memoize active tile for drag overlay
  const activeTile = useMemo(
    () => (activeId ? tiles.find((t) => tileKey(t) === activeId) : null),
    [activeId, tiles]
  )

  // Create collision detection that handles both tab and service drags
  const groupIds = useMemo(() => groups.map((g) => g.id), [groups])
  const collisionDetection = useMemo(() => {
    const sectionOfTile = sectioned
      ? new Map(sortableSections.flatMap((s) => s.tiles.map((t) => [tileKey(t), s.group.id])))
      : undefined
    return createCollisionDetection(isDraggingTab, groupIds, sectionOfTile)
  }, [isDraggingTab, groupIds, sectioned, sortableSections])

  // DnD sensors for mouse and touch
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 8 },
    }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: 150, tolerance: 5 },
    })
  )

  // Load selected group from localStorage
  useEffect(() => {
    const saved = localStorage.getItem('selectedGroupId')
    if (saved) {
      setSelectedGroupId(saved)
    }
  }, [])

  // Save selected group to localStorage
  useEffect(() => {
    if (selectedGroupId) {
      localStorage.setItem('selectedGroupId', selectedGroupId)
    } else {
      localStorage.removeItem('selectedGroupId')
    }
  }, [selectedGroupId])

  // Fetch services and groups
  useEffect(() => {
    fetchData()
  }, [])

  // Live updates: health checks and widget data arrive over SSE; while the
  // stream is down the hook falls back to polling every 30 seconds
  const resyncServices = useCallback(async () => {
    // Skip if we just did a full fetch (within last 5 seconds)
    if (shouldSkipPoll(lastPollTime.current)) return
    try {
      const response = await api.getServices()
      if (response.data) {
        // Only update status and response_time to avoid disrupting UI state
        setServices((prev) => mergeServicesHealth(prev, response.data!))
      }
    } catch (error) {
      console.error('Failed to poll health:', error)
    }
  }, [])

  const applyServiceStatus = useCallback((event: ServiceStatusEvent) => {
    setServices((prev) =>
      prev.map((s) =>
        s.id === event.id
          ? mergeHealthData(s, { status: event.status, response_time: event.response_time })
          : s
      )
    )
  }, [])

  // KPI definitions per integration, for service tiles linked to one
  const { integrations, kinds } = useIntegrations()
  const integrationKpis = useMemo(
    () => Object.fromEntries(integrations.map((i) => [i.id, kpisForIntegration(i, kinds)])),
    [integrations, kinds]
  )

  const { snapshots } = useDashboardStream({
    onServiceStatus: applyServiceStatus,
    onResync: resyncServices,
  })

  // The canvas top bar shows the weather of the first weather widget
  const weatherWidget = useMemo(
    () => widgets.find((w) => w.type === 'weather' && w.enabled),
    [widgets]
  )
  const weatherSnapshot = weatherWidget && snapshots[snapshotKey('widget', weatherWidget.id)]
  useEffect(() => {
    const weather = weatherSnapshot?.payload as WeatherPayload | null | undefined
    setTopBarWeather(
      weatherWidget && weather
        ? { location: String(weatherWidget.config.location ?? ''), weather }
        : null
    )
  }, [weatherWidget, weatherSnapshot])

  // Leaving the dashboard ends what it shared with the top bar
  useEffect(
    () => () => {
      setEditMode(false)
      setTopBarWeather(null)
    },
    []
  )

  // Auto-select default group when groups load
  useEffect(() => {
    if (enableServiceGrouping && groups.length > 0 && !selectedGroupId) {
      const defaultGroup = groups.find((g) => g.is_default)
      if (defaultGroup) {
        setSelectedGroupId(defaultGroup.id)
      } else {
        setSelectedGroupId(groups[0].id)
      }
    }
  }, [groups, enableServiceGrouping, selectedGroupId])

  // Validate selected group still exists
  useEffect(() => {
    if (selectedGroupId && groups.length > 0) {
      const exists = groups.some((g) => g.id === selectedGroupId)
      if (!exists) {
        const defaultGroup = groups.find((g) => g.is_default)
        setSelectedGroupId(defaultGroup?.id || groups[0]?.id || null)
      }
    }
  }, [groups, selectedGroupId])

  const fetchData = async () => {
    setIsLoading(true)
    try {
      const [servicesResponse, groupsResponse, widgetsResponse, widgetTypesResponse] =
        await Promise.all([
          api.getServices(),
          api.getGroups(),
          api.getWidgets(),
          api.getWidgetTypes(),
        ])

      if (servicesResponse.data) {
        setServices(servicesResponse.data)
      }
      if (groupsResponse.data) {
        setGroups(groupsResponse.data)
      }
      if (widgetsResponse.data) {
        setWidgets(widgetsResponse.data)
        setWidgetsLoaded(true)
      } else {
        console.error('Failed to fetch widgets:', widgetsResponse.error?.message)
      }
      if (widgetTypesResponse.data) {
        setWidgetTypes(widgetTypesResponse.data)
      }
      // Track when we did a full fetch to avoid immediate poll overlap
      lastPollTime.current = Date.now()
    } catch (error) {
      console.error('Failed to fetch data:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const handleDragStart = (event: DragStartEvent) => {
    const id = toStringId(event.active.id)

    // Check if dragging a tab (ID matches a group ID)
    const isTabDrag = groupIds.includes(id)
    setIsDraggingTab(isTabDrag)

    if (!isTabDrag) {
      // Tile drag - set active ID for drag overlay
      setActiveId(id)
    }
  }

  const handleDragCancel = () => {
    setActiveId(null)
    setIsDraggingTab(false)
  }

  // Handle tab reorder
  const handleTabReorder = async (activeId: string, overId: string) => {
    const oldIndex = groups.findIndex((g) => g.id === activeId)
    const newIndex = groups.findIndex((g) => g.id === overId)

    if (oldIndex === -1 || newIndex === -1 || oldIndex === newIndex) return

    const reorderedGroups = arrayMove(groups, oldIndex, newIndex)

    // Optimistic update
    const previousGroups = groups
    setGroups(reorderedGroups)

    try {
      await api.reorderGroups({
        groups: reorderedGroups.map((g, index) => ({
          id: g.id,
          position: index,
        })),
      })
    } catch (error) {
      console.error('Failed to reorder groups:', error)
      setGroups(previousGroups)
    }
  }

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event
    setActiveId(null)
    setIsDraggingTab(false)
    if (!over) return

    const activeId = toStringId(active.id)
    const overId = toStringId(over.id)

    // Check if this is a tab reorder (both active and over are group IDs)
    const isTabDrag = groupIds.includes(activeId)
    const isTabTarget = groupIds.includes(overId)

    if (isTabDrag && isTabTarget && activeId !== overId) {
      await handleTabReorder(activeId, overId)
      return
    }

    // Skip if it was a tab drag that didn't land on another tab
    if (isTabDrag) return

    if (sectioned) {
      if (activeId !== overId && widgetsLoaded) dropInSections(activeId, overId)
      return
    }

    // Check if dropped on a group tab (IDs prefixed with "group-")
    if (overId.startsWith(GROUP_DROP_PREFIX)) {
      const draggedTile = tiles.find((t) => tileKey(t) === activeId)
      if (draggedTile) {
        moveTileToGroup(draggedTile, overId.replace(GROUP_DROP_PREFIX, ''))
      }
      return
    }

    // Normal reorder within the visible tiles; hidden tiles keep their slots
    if (activeId === overId || !widgetsLoaded) return
    const reordered = reorderTiles(tiles, sortableTiles, activeId, overId)
    if (reordered) saveTileMove(activeId, reordered)
  }

  // Tile moves save one at a time, so they reach the server in the order
  // they were made. A save that fails undoes its move on the latest tiles.
  const tileSaves = useRef(Promise.resolve())
  const tilesRef = useRef(tiles)
  useEffect(() => {
    tilesRef.current = tiles
  }, [tiles])

  const showTiles = (order: Tile[]) => {
    tilesRef.current = order
    const next = splitTiles(order)
    setServices(next.services)
    setWidgets(next.widgets)
  }

  // Shows the move of tile `key` at once, then saves the group it joined, if
  // any, and every position if the order changed, so services and widgets
  // never share one
  const saveTileMove = (key: string, order: Tile[], group?: Group) => {
    let before = tiles
    const moved = order.find((tile) => tileKey(tile) === key)
    const reordered = order.some((tile, i) => tileKey(tile) !== tileKey(tiles[i]))
    showTiles(order)
    tileSaves.current = tileSaves.current.then(async () => {
      try {
        if (group && moved) {
          await saveTileGroup(moved, group.is_default ? '' : group.id)
          // If the order can't be saved, the tile is still in its new group
          before = before.map((tile) => (tileKey(tile) === key ? moved : tile))
        }
        if (reordered) {
          const response = await api.reorderTiles({ tiles: toTilePositions(tilesRef.current) })
          if (response.error) throw new Error(response.error.message)
        }
      } catch (error) {
        console.error('Failed to move tile:', error)
        showTiles(undoMove(tilesRef.current, order, before, key))
      }
    })
  }

  // A drop in sections mode: a reorder, or a move to another section
  const dropInSections = (activeKey: string, overId: string) => {
    const move = moveTile(tiles, sortableSections, activeKey, overId)
    if (move) saveTileMove(activeKey, move.tiles, move.regrouped?.group)
  }

  // Saves the group of a service or widget; '' clears it (the default group).
  // The API client returns errors instead of throwing, so this throws them.
  const saveTileGroup = async (tile: Tile, groupId: string) => {
    const response =
      tile.kind === 'widget'
        ? await api.updateWidget(tile.id, { group_id: groupId })
        : // Only the group: other fields keep what is stored
          await api.updateService(tile.id, { group_id: groupId })
    if (response.error) throw new Error(response.error.message)
  }

  // Moves a service or widget to another group (dropped on a group tab)
  const moveTileToGroup = (tile: Tile, targetGroupId: string) => {
    const group = groups.find((g) => g.id === targetGroupId)
    // The default group holds the ungrouped tiles
    const currentGroupId = tileGroupId(tile)
    if (!group || (group.is_default ? !currentGroupId : currentGroupId === group.id)) return
    const key = tileKey(tile)
    saveTileMove(
      key,
      tiles.map((t) => (tileKey(t) === key ? withGroup(t, group) : t)),
      group
    )
  }

  const handleSizeChange = async (id: string, newSize: CardSize) => {
    // Prevent rapid clicks while already resizing
    if (resizingId) return

    // Capture current state for rollback
    const previousServices = services

    // Optimistic update with loading state
    setResizingId(id)
    setServices(services.map((s) => (s.id === id ? { ...s, card_size: newSize } : s)))

    // Only the size: an edit saved meanwhile keeps its other fields
    try {
      const updated = await api.updateService(id, { card_size: newSize })
      if (updated.error) throw new Error(updated.error.message)
    } catch (error) {
      console.error('Failed to update card size:', error)
      setServices(previousServices)
    } finally {
      setResizingId(null)
    }
  }

  // Group handlers
  const handleSelectGroup = useCallback((groupId: string | null) => {
    setSelectedGroupId(groupId)
  }, [])

  const handleCreateGroup = useCallback(() => {
    setEditingGroup(null)
    setShowGroupForm(true)
  }, [])

  const handleEditGroup = useCallback((group: Group) => {
    setEditingGroup(group)
    setShowGroupForm(true)
  }, [])

  const handleDeleteGroup = useCallback((group: Group) => {
    setDeletingGroup(group)
  }, [])

  // Widget handlers
  const handleAddWidget = useCallback(() => {
    setEditingWidget(null)
    setShowWidgetModal(true)
  }, [])

  const handleEditWidget = useCallback((widget: Widget) => {
    setEditingWidget(widget)
    setShowWidgetModal(true)
  }, [])

  const handleWidgetSaved = (saved: Widget) => {
    setWidgets((prev) =>
      prev.some((w) => w.id === saved.id)
        ? prev.map((w) => (w.id === saved.id ? saved : w))
        : [...prev, saved]
    )
    setShowWidgetModal(false)
    setEditingWidget(null)
  }

  const closeServiceModal = useCallback(() => setEditingService(null), [])

  // The response is the whole service, health included
  const handleServiceSaved = useCallback((saved: Service) => {
    setServices((prev) => prev.map((s) => (s.id === saved.id ? saved : s)))
    setEditingService(null)
  }, [])

  const handleWidgetSizeChange = async (widget: Widget, size: CardSize) => {
    if (resizingId) return
    const previousWidgets = widgets
    setResizingId(widget.id)
    setWidgets(widgets.map((w) => (w.id === widget.id ? { ...w, card_size: size } : w)))
    try {
      const response = await api.updateWidget(widget.id, { card_size: size })
      if (response.error) throw new Error(response.error.message)
    } catch (error) {
      console.error('Failed to update widget size:', error)
      setWidgets(previousWidgets)
    } finally {
      setResizingId(null)
    }
  }

  const handleRefreshWidget = useCallback(async (widget: Widget) => {
    const response = await api.refreshWidget(widget.id)
    if (response.error) console.error('Failed to refresh widget:', response.error.message)
  }, [])

  const handleConfirmDeleteWidget = async () => {
    if (!deletingWidget) return
    try {
      const response = await api.deleteWidget(deletingWidget.id)
      if (response.error) {
        console.error('Failed to delete widget:', response.error.message)
        return
      }
      setWidgets((prev) => prev.filter((w) => w.id !== deletingWidget.id))
    } finally {
      setDeletingWidget(null)
    }
  }

  // Memoize add service href to avoid recalculating on every render.
  // `from=dashboard` tells /services/new where to redirect after create.
  const addServiceHref = useMemo(() => {
    const params = new URLSearchParams()
    if (tabbed && selectedGroupId) params.set('group', selectedGroupId)
    params.set('from', 'dashboard')
    return `/services/new?${params.toString()}`
  }, [tabbed, selectedGroupId])

  // Sections have no tabs to drag, so a group swaps places with its neighbour
  const handleMoveGroup = (group: Group, direction: -1 | 1) => {
    const neighbour = groups[groups.findIndex((g) => g.id === group.id) + direction]
    if (neighbour) handleTabReorder(group.id, neighbour.id)
  }

  const handleGroupFormSubmit = async (data: GroupUpdateRequest) => {
    setGroupFormLoading(true)
    try {
      if (editingGroup) {
        const response = await api.updateGroup(editingGroup.id, data)
        if (response.error) {
          throw new Error(response.error.message)
        }
        if (response.data) {
          setGroups(groups.map((g) => (g.id === editingGroup.id ? response.data! : g)))
        }
      } else {
        const response = await api.createGroup({
          name: data.name!,
          color: data.color,
          monitoring_enabled: data.monitoring_enabled,
          tile_style: data.tile_style,
        })
        if (response.error) {
          throw new Error(response.error.message)
        }
        if (response.data) {
          setGroups([...groups, response.data])
          // Select the newly created group
          setSelectedGroupId(response.data.id)
        }
      }
      setShowGroupForm(false)
      setEditingGroup(null)
    } finally {
      setGroupFormLoading(false)
    }
  }

  const handleConfirmDeleteGroup = async () => {
    if (!deletingGroup) return

    try {
      const response = await api.deleteGroup(deletingGroup.id, deleteGroupServices)
      if (response.error) {
        console.error('Failed to delete group:', response.error.message)
        return
      }

      // Remove from state
      const newGroups = groups.filter((g) => g.id !== deletingGroup.id)
      setGroups(newGroups)

      // If deleted group was selected, select default
      if (selectedGroupId === deletingGroup.id) {
        const defaultGroup = newGroups.find((g) => g.is_default)
        setSelectedGroupId(defaultGroup?.id || newGroups[0]?.id || null)
      }

      // Update services state based on whether they were deleted or moved
      if (deleteGroupServices) {
        // Remove services and widgets that were in the deleted group
        setServices(services.filter((s) => s.group_id !== deletingGroup.id))
        setWidgets(widgets.filter((w) => w.group_id !== deletingGroup.id))
      } else {
        // Move services and widgets from deleted group to ungrouped locally
        setServices(
          services.map((s) => (s.group_id === deletingGroup.id ? { ...s, group_id: undefined } : s))
        )
        setWidgets(
          widgets.map((w) => (w.group_id === deletingGroup.id ? { ...w, group_id: null } : w))
        )
      }
    } finally {
      setDeletingGroup(null)
      setDeleteGroupServices(false)
    }
  }

  // Without a cached choice, tabs or sections is only known once the
  // preferences are in; waiting avoids showing one and then the other
  if (isLoading || (preferencesLoading && enableServiceGrouping && groupDisplay === null)) {
    return (
      <div className="flex min-h-96 items-center justify-center">
        <div className="wallpaper-panel text-center">
          <div className="border-primary mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-t-2 border-b-2"></div>
          <p className="text-text-secondary">Loading dashboard...</p>
        </div>
      </div>
    )
  }

  const header = (
    <DashboardHeader
      groups={groups}
      selectedGroupId={selectedGroupId}
      onSelectGroup={handleSelectGroup}
      onCreateGroup={handleCreateGroup}
      onEditGroup={handleEditGroup}
      onDeleteGroup={handleDeleteGroup}
      isEditMode={isEditMode}
      onToggleEditMode={toggleEditMode}
      enableServiceGrouping={tabbed}
      activeId={activeId}
      isDraggingTab={isDraggingTab}
      addServiceHref={addServiceHref}
      onAddWidget={handleAddWidget}
    />
  )

  // A grid (or list) of tiles; sortable in edit mode
  const renderTiles = (gridTiles: Tile[], tileStyle?: TileStyle) => {
    const grid = (
      <ServicesGrid
        tiles={gridTiles}
        tileStyle={tileStyle}
        openInNewTab={openInNewTab}
        enableCardResizing={enableCardResizing}
        cardScale={cardScale}
        viewMode={viewMode}
        groupMonitoringMap={groupMonitoringMap}
        widgetTypes={widgetTypeMap}
        snapshots={snapshots}
        integrationKpis={integrationKpis}
        {...(isEditMode && {
          isEditMode,
          onSizeChange: handleSizeChange,
          onEditService: setEditingService,
          onWidgetSizeChange: handleWidgetSizeChange,
          onEditWidget: handleEditWidget,
          onDeleteWidget: setDeletingWidget,
          onRefreshWidget: handleRefreshWidget,
        })}
      />
    )
    if (!isEditMode) return grid
    const sortable = sortableOrder(gridTiles, viewMode, tileStyle)
    return (
      <SortableContext items={sortable.map(tileKey)} strategy={rectSortingStrategy}>
        {grid}
      </SortableContext>
    )
  }

  const tileArea = sectioned ? (
    <GroupSections
      sections={sections}
      isEditMode={isEditMode}
      isDraggingTile={!!activeId}
      onCreateGroup={handleCreateGroup}
      onEditGroup={handleEditGroup}
      onDeleteGroup={handleDeleteGroup}
      onMoveGroup={handleMoveGroup}
    >
      {(section) => renderTiles(section.tiles, section.group.tile_style)}
    </GroupSections>
  ) : (
    renderTiles(
      filteredTiles,
      tabbed ? groups.find((g) => g.id === selectedGroupId)?.tile_style : undefined
    )
  )

  return (
    <div>
      <DiscoveredTray variant="link" />
      {statusStrip.enabled && (
        <StatusStrip
          strip={statusStrip}
          snapshots={snapshots}
          integrations={integrations}
          kinds={kinds}
          widgets={widgets}
          up={allStatuses.up}
          down={allStatuses.down}
        />
      )}

      {/* Stats cards; the canvas layout leaves them out */}
      {layoutMode === 'classic' && <DashboardStats stats={stats} />}

      {!widgetsLoaded && (
        <p className="text-error wallpaper-panel mb-4 text-sm">
          Widgets could not be loaded. Reload the page to see them; until then the order cannot be
          changed.
        </p>
      )}

      {/* Header with group tabs and action buttons, then the tiles */}
      {isEditMode ? (
        <DndContext
          sensors={sensors}
          collisionDetection={collisionDetection}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          onDragCancel={handleDragCancel}
        >
          {header}
          {tileArea}

          <DragOverlay dropAnimation={null}>
            {activeTile?.kind === 'widget' && (
              <WidgetCard
                widget={activeTile.widget}
                snapshot={snapshots[snapshotKey('widget', activeTile.widget.id)]}
                openInNewTab={openInNewTab}
                cardScale={cardScale}
                isEditMode
                isDragging
              />
            )}
            {activeTile?.kind === 'service' &&
              (enableServiceGrouping && tileStyleOf(activeTile, groups) === 'chips' ? (
                <ServiceChip
                  service={activeTile.service}
                  openInNewTab={openInNewTab}
                  cardScale={cardScale}
                  isEditMode
                  isDragging
                  isMonitored={isServiceEffectivelyMonitored(
                    activeTile.service,
                    groupMonitoringMap
                  )}
                />
              ) : viewMode === 'list' ? (
                <ServiceListItem
                  service={activeTile.service}
                  openInNewTab={openInNewTab}
                  isEditMode={true}
                  isDragging={true}
                  isMonitored={isServiceEffectivelyMonitored(
                    activeTile.service,
                    groupMonitoringMap
                  )}
                />
              ) : (
                <ServiceCard
                  service={activeTile.service}
                  openInNewTab={openInNewTab}
                  isEditMode={true}
                  onSizeChange={() => {}}
                  isDragging={true}
                  enableCardResizing={enableCardResizing}
                  cardScale={cardScale}
                  isMonitored={isServiceEffectivelyMonitored(
                    activeTile.service,
                    groupMonitoringMap
                  )}
                />
              ))}
          </DragOverlay>
        </DndContext>
      ) : (
        <>
          {header}
          {tileArea}
        </>
      )}

      {/* Empty state for groups with no tiles; sections have their own */}
      {tabbed && filteredTiles.length === 0 && !isLoading && (
        <div className="glass-card border-card-border rounded-lg border p-12 text-center">
          <ServerIcon className="text-text-muted mx-auto mb-4 h-12 w-12" />
          <h3 className="text-text-primary mb-2 text-lg font-medium">Nothing in this group yet</h3>
          <p className="text-text-secondary mb-4">
            Add a service or a widget to this group to see it here.
          </p>
          <Link
            href={addServiceHref}
            className="bg-primary hover:bg-primary-hover inline-flex items-center rounded-md px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            <PlusIcon className="mr-2 h-4 w-4" />
            Add Service
          </Link>
        </div>
      )}

      {/* Group form modal */}
      {showGroupForm && (
        <GroupForm
          group={editingGroup}
          onSubmit={handleGroupFormSubmit}
          onClose={() => {
            setShowGroupForm(false)
            setEditingGroup(null)
          }}
          isLoading={groupFormLoading}
        />
      )}

      {/* Widget add/edit modal */}
      {showWidgetModal && (
        <WidgetModal
          types={widgetTypes}
          widget={editingWidget}
          groupId={tabbed ? selectedGroupId : null}
          onClose={() => {
            setShowWidgetModal(false)
            setEditingWidget(null)
          }}
          onSaved={handleWidgetSaved}
        />
      )}

      {/* Service edit modal */}
      {editingService && (
        <ServiceModal
          service={editingService}
          onClose={closeServiceModal}
          onSaved={handleServiceSaved}
        />
      )}

      {/* Widget delete confirmation */}
      {deletingWidget && (
        <ConfirmDialog
          title="Delete Widget"
          message="Are you sure you want to delete this widget? This cannot be undone."
          confirmLabel="Delete"
          onConfirm={handleConfirmDeleteWidget}
          onCancel={() => setDeletingWidget(null)}
        />
      )}

      {/* Delete confirmation modal */}
      {deletingGroup && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div
            className="absolute inset-0 bg-black/50"
            onClick={() => {
              setDeletingGroup(null)
              setDeleteGroupServices(false)
            }}
            aria-hidden="true"
          />
          <div className="bg-card border-card-border relative z-10 w-full max-w-md rounded-lg border p-6 shadow-lg">
            <h3 className="text-text-primary mb-2 text-lg font-semibold">Delete Group</h3>
            <p className="text-text-secondary mb-4">
              Are you sure you want to delete &ldquo;{deletingGroup.name}&rdquo;?
              {!deleteGroupServices &&
                ' Services and widgets in this group will be moved to the default group.'}
            </p>

            {/* Checkbox to delete services */}
            <label className="mb-4 flex cursor-pointer items-start gap-3">
              <input
                type="checkbox"
                checked={deleteGroupServices}
                onChange={(e) => setDeleteGroupServices(e.target.checked)}
                className="text-error focus:ring-error mt-0.5 h-4 w-4 rounded border-gray-300"
              />
              <span className="text-text-secondary text-sm">
                Permanently delete all services and widgets in this group
              </span>
            </label>

            {deleteGroupServices && (
              <p className="text-error mb-4 text-sm">
                Warning: This will permanently delete all services and widgets in this group. This
                action cannot be undone.
              </p>
            )}

            <div className="flex justify-end gap-2">
              <button
                onClick={() => {
                  setDeletingGroup(null)
                  setDeleteGroupServices(false)
                }}
                className="text-text-secondary hover:text-text-primary hover:bg-card-hover rounded-md px-4 py-2 text-sm font-medium transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmDeleteGroup}
                className="bg-error hover:bg-error/80 rounded-md px-4 py-2 text-sm font-medium text-white transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
