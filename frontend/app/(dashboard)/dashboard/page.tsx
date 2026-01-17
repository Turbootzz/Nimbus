'use client'

import { useState, useEffect, useMemo, useCallback } from 'react'
import {
  ServerIcon,
  ClockIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
  PlusIcon,
  PencilIcon,
  CheckIcon,
} from '@heroicons/react/24/outline'
import Link from 'next/link'
import {
  DndContext,
  closestCenter,
  pointerWithin,
  CollisionDetection,
  DragEndEvent,
  DragStartEvent,
  DragOverlay,
  PointerSensor,
  TouchSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import { arrayMove, SortableContext, rectSortingStrategy, useSortable } from '@dnd-kit/sortable'
import { api } from '@/lib/api'
import type { Service, CardSize, Group } from '@/types'
import { useTheme } from '@/contexts/ThemeContext'
import { sizeToGridSpan } from '@/lib/card-utils'
import ServiceCard from '@/components/ServiceCard'
import GroupTabs from '@/components/GroupTabs'
import GroupForm from '@/components/GroupForm'

// Custom collision detection: prioritize group tabs (pointer-based), then service grid (center-based)
const customCollisionDetection: CollisionDetection = (args) => {
  // First check if pointer is within any group tabs
  const pointerCollisions = pointerWithin(args)
  const groupTabCollision = pointerCollisions.find((c) => String(c.id).startsWith('group-'))
  if (groupTabCollision) {
    return [groupTabCollision]
  }

  // Fall back to closestCenter for service card reordering
  return closestCenter(args)
}

// Sortable wrapper for ServiceCard in edit mode
function SortableServiceCard({
  service,
  openInNewTab,
  isEditMode,
  onSizeChange,
  enableCardResizing,
}: {
  service: Service
  openInNewTab: boolean
  isEditMode: boolean
  onSizeChange: (id: string, newSize: CardSize) => void
  enableCardResizing: boolean
}) {
  const { attributes, listeners, setNodeRef, isDragging } = useSortable({
    id: service.id,
  })

  // When card resizing is disabled, always use 2x1
  const cardSize = enableCardResizing ? service.card_size || '2x1' : '2x1'
  const gridSpan = sizeToGridSpan[cardSize]

  // Don't apply transforms - with dense grid and variable sizes, transforms cause
  // weird stretching/compression. Cards stay in place, only reorder after drop.
  const style = {
    opacity: isDragging ? 0.4 : 1,
    transition: 'opacity 150ms ease',
  }

  return (
    <div ref={setNodeRef} style={style} className={`${gridSpan} h-full`}>
      <ServiceCard
        service={service}
        openInNewTab={openInNewTab}
        isEditMode={isEditMode}
        onSizeChange={onSizeChange}
        dragHandleProps={{ ...attributes, ...listeners }}
        isDragging={false}
        enableCardResizing={enableCardResizing}
      />
    </div>
  )
}

export default function DashboardPage() {
  const { openInNewTab, enableCardResizing, enableServiceGrouping } = useTheme()
  const [services, setServices] = useState<Service[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [selectedGroupId, setSelectedGroupId] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isEditMode, setIsEditMode] = useState(false)
  const [activeId, setActiveId] = useState<string | null>(null)
  const [resizingId, setResizingId] = useState<string | null>(null)

  // Group form modal state
  const [showGroupForm, setShowGroupForm] = useState(false)
  const [editingGroup, setEditingGroup] = useState<Group | null>(null)
  const [groupFormLoading, setGroupFormLoading] = useState(false)

  // Delete confirmation state
  const [deletingGroup, setDeletingGroup] = useState<Group | null>(null)
  const [deleteGroupServices, setDeleteGroupServices] = useState(false)

  const [stats, setStats] = useState({
    total: 0,
    online: 0,
    offline: 0,
    avgResponseTime: 0,
  })

  // Filter services by selected group
  // When grouping is disabled: show all services
  // When grouping is enabled:
  //   - Default group: show services with this group_id OR no group_id (backwards compat)
  //   - Other groups: show only services with this specific group_id
  const filteredServices = useMemo(() => {
    if (!enableServiceGrouping) {
      return services
    }
    if (!selectedGroupId) {
      return services
    }
    const selectedGroup = groups.find((g) => g.id === selectedGroupId)
    if (selectedGroup?.is_default) {
      // Default group includes ungrouped services for backwards compatibility
      return services.filter((s) => s.group_id === selectedGroupId || !s.group_id)
    }
    return services.filter((s) => s.group_id === selectedGroupId)
  }, [services, selectedGroupId, enableServiceGrouping, groups])

  // Memoize active service for drag overlay
  const activeService = useMemo(
    () => (activeId ? services.find((s) => s.id === activeId) : null),
    [activeId, services]
  )

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

  useEffect(() => {
    // Calculate stats from filtered services
    const servicesToCount = enableServiceGrouping ? filteredServices : services
    const online = servicesToCount.filter((s) => s.status === 'online').length
    const offline = servicesToCount.filter((s) => s.status === 'offline').length
    const responseTimes = servicesToCount
      .filter((s) => s.response_time !== undefined && s.response_time !== null)
      .map((s) => s.response_time as number)
    const avgResponseTime =
      responseTimes.length > 0
        ? Math.round(responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length)
        : 0

    setStats({
      total: servicesToCount.length,
      online,
      offline,
      avgResponseTime,
    })
  }, [services, filteredServices, enableServiceGrouping])

  const fetchData = async () => {
    setIsLoading(true)
    try {
      const [servicesResponse, groupsResponse] = await Promise.all([
        api.getServices(),
        api.getGroups(),
      ])

      if (servicesResponse.data) {
        setServices(servicesResponse.data)
      }
      if (groupsResponse.data) {
        setGroups(groupsResponse.data)
      }
    } catch (error) {
      console.error('Failed to fetch data:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(event.active.id as string)
  }

  const handleDragCancel = () => {
    setActiveId(null)
  }

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event
    setActiveId(null)
    if (!over) return

    const activeServiceId = active.id as string

    // Check if dropped on a group tab (IDs prefixed with "group-")
    const overId = over.id as string
    if (overId.startsWith('group-')) {
      const targetGroupId = overId.replace('group-', '')
      const draggedService = services.find((s) => s.id === activeServiceId)
      if (!draggedService) return

      // Find target group to check if it's the default
      const targetGroup = groups.find((g) => g.id === targetGroupId)
      const isTargetDefault = targetGroup?.is_default

      // For default group: set group_id to null (ungrouped services appear there)
      // For other groups: set group_id to the target group's ID
      const newGroupId = isTargetDefault ? null : targetGroupId

      // Check if already in the target group
      // Service is in default if: group_id is null/undefined OR matches default group ID
      const isCurrentlyInDefault =
        !draggedService.group_id || draggedService.group_id === targetGroupId
      if (isTargetDefault && isCurrentlyInDefault && !draggedService.group_id) return
      if (!isTargetDefault && draggedService.group_id === targetGroupId) return

      // Capture current state for rollback
      const previousServices = services

      // Optimistic update - move service to new group
      setServices(
        services.map((s) =>
          s.id === activeServiceId ? { ...s, group_id: newGroupId ?? undefined } : s
        )
      )

      // Persist to backend
      try {
        await api.updateService(activeServiceId, {
          name: draggedService.name,
          url: draggedService.url,
          description: draggedService.description || '',
          icon: draggedService.icon || '',
          icon_type: draggedService.icon_type || 'emoji',
          icon_image_path: draggedService.icon_image_path || '',
          group_id: newGroupId,
        })
      } catch (error) {
        console.error('Failed to move service to group:', error)
        setServices(previousServices)
      }
      return
    }

    // Normal reorder within the same group
    if (active.id === over.id) return

    // Use filtered services for the reorder operation
    const servicesToReorder = enableServiceGrouping ? filteredServices : services
    const oldIndex = servicesToReorder.findIndex((s) => s.id === active.id)
    const newIndex = servicesToReorder.findIndex((s) => s.id === over.id)
    if (oldIndex === -1 || newIndex === -1) return

    // Capture current state for rollback
    const previousServices = services

    // Reorder within filtered list
    const reorderedFiltered = arrayMove(servicesToReorder, oldIndex, newIndex)

    // Merge back into full services list
    let reorderedAll: Service[]
    if (enableServiceGrouping && selectedGroupId) {
      // Replace filtered services in their positions
      const otherServices = services.filter((s) => !servicesToReorder.some((fs) => fs.id === s.id))
      reorderedAll = [...otherServices, ...reorderedFiltered]
    } else {
      reorderedAll = reorderedFiltered
    }

    // Optimistic update
    setServices(reorderedAll)

    // Persist to backend (send all services with new positions)
    try {
      await api.reorderServices({
        services: reorderedFiltered.map((s, i) => ({ id: s.id, position: i })),
      })
    } catch (error) {
      console.error('Failed to save order:', error)
      setServices(previousServices)
    }
  }

  const handleSizeChange = async (id: string, newSize: CardSize) => {
    // Prevent rapid clicks while already resizing
    if (resizingId) return

    // Capture current state for rollback
    const previousServices = services

    // Optimistic update with loading state
    setResizingId(id)
    setServices(services.map((s) => (s.id === id ? { ...s, card_size: newSize } : s)))

    // Fetch fresh service data to avoid overwriting concurrent updates
    try {
      const response = await api.getService(id)
      if (response.error || !response.data) {
        console.error('Failed to fetch service:', response.error?.message)
        setServices(previousServices)
        return
      }

      const freshService = response.data
      await api.updateService(id, {
        name: freshService.name,
        url: freshService.url,
        description: freshService.description || '',
        icon: freshService.icon || '',
        icon_type: freshService.icon_type || 'emoji',
        icon_image_path: freshService.icon_image_path || '',
        card_size: newSize,
      })
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

  const handleGroupFormSubmit = async (data: { name?: string; color?: string }) => {
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
        const response = await api.createGroup({ name: data.name!, color: data.color })
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
        // Remove services that were in the deleted group
        setServices(services.filter((s) => s.group_id !== deletingGroup.id))
      } else {
        // Move services from deleted group to null (ungrouped) locally
        setServices(
          services.map((s) => (s.group_id === deletingGroup.id ? { ...s, group_id: undefined } : s))
        )
      }
    } finally {
      setDeletingGroup(null)
      setDeleteGroupServices(false)
    }
  }

  if (isLoading) {
    return (
      <div className="flex min-h-96 items-center justify-center">
        <div className="text-center">
          <div className="border-primary mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-t-2 border-b-2"></div>
          <p className="text-text-secondary">Loading dashboard...</p>
        </div>
      </div>
    )
  }

  return (
    <div>
      {/* Stats cards */}
      <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="bg-card border-card-border rounded-lg border p-6">
          <div className="flex items-center">
            <ServerIcon className="text-primary h-8 w-8" />
            <div className="ml-4">
              <p className="text-text-muted text-sm">Total Services</p>
              <p className="text-text-primary text-2xl font-semibold">{stats.total}</p>
            </div>
          </div>
        </div>

        <div className="bg-card border-card-border rounded-lg border p-6">
          <div className="flex items-center">
            <CheckCircleIcon className="text-success h-8 w-8" />
            <div className="ml-4">
              <p className="text-text-muted text-sm">Online</p>
              <p className="text-text-primary text-2xl font-semibold">{stats.online}</p>
            </div>
          </div>
        </div>

        <div className="bg-card border-card-border rounded-lg border p-6">
          <div className="flex items-center">
            <ExclamationCircleIcon className="text-error h-8 w-8" />
            <div className="ml-4">
              <p className="text-text-muted text-sm">Offline</p>
              <p className="text-text-primary text-2xl font-semibold">{stats.offline}</p>
            </div>
          </div>
        </div>

        <div className="bg-card border-card-border rounded-lg border p-6">
          <div className="flex items-center">
            <ClockIcon className="text-info h-8 w-8" />
            <div className="ml-4">
              <p className="text-text-muted text-sm">Avg Response</p>
              <p className="text-text-primary text-2xl font-semibold">{stats.avgResponseTime}ms</p>
            </div>
          </div>
        </div>
      </div>

      {/* Edit mode: wrap tabs and grid in single DndContext for drag-to-tab functionality */}
      {isEditMode ? (
        <DndContext
          sensors={sensors}
          collisionDetection={customCollisionDetection}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          onDragCancel={handleDragCancel}
        >
          {/* Tabs and action buttons - stacked on mobile, row on larger screens */}
          <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            {/* Group tabs (only show when grouping is enabled and groups exist) */}
            <div className="min-w-0 flex-1">
              {enableServiceGrouping && groups.length > 0 && (
                <GroupTabs
                  groups={groups}
                  selectedGroupId={selectedGroupId}
                  onSelectGroup={handleSelectGroup}
                  onCreateGroup={handleCreateGroup}
                  onEditGroup={handleEditGroup}
                  onDeleteGroup={handleDeleteGroup}
                  isEditMode={isEditMode}
                  isDraggingService={!!activeId}
                />
              )}
            </div>

            {/* Action buttons */}
            <div className="flex shrink-0 items-center gap-2">
              <button
                onClick={() => setIsEditMode(!isEditMode)}
                className="bg-success hover:bg-success/80 inline-flex items-center rounded-md px-3 py-2 text-sm font-medium text-white transition-colors sm:px-4"
              >
                <CheckIcon className="h-4 w-4 sm:mr-2" />
                <span className="hidden sm:inline">Done</span>
              </button>
              <Link
                href={
                  enableServiceGrouping && selectedGroupId
                    ? `/services/new?group=${selectedGroupId}`
                    : '/services/new'
                }
                className="bg-primary hover:bg-primary-hover inline-flex items-center rounded-md px-3 py-2 text-sm font-medium text-white transition-colors sm:px-4"
              >
                <PlusIcon className="h-4 w-4 sm:mr-2" />
                <span className="hidden sm:inline">Add Service</span>
              </Link>
            </div>
          </div>

          {/* Services grid */}
          <SortableContext items={filteredServices.map((s) => s.id)} strategy={rectSortingStrategy}>
            <div
              className="grid grid-cols-2 gap-4 sm:grid-cols-4 xl:grid-cols-6 2xl:grid-cols-8"
              style={{ gridAutoFlow: 'dense' }}
            >
              {filteredServices.map((service) => (
                <SortableServiceCard
                  key={service.id}
                  service={service}
                  openInNewTab={openInNewTab}
                  isEditMode={isEditMode}
                  onSizeChange={handleSizeChange}
                  enableCardResizing={enableCardResizing}
                />
              ))}
            </div>
          </SortableContext>

          <DragOverlay dropAnimation={null}>
            {activeService && (
              <ServiceCard
                service={activeService}
                openInNewTab={openInNewTab}
                isEditMode={true}
                onSizeChange={() => {}}
                isDragging={true}
                enableCardResizing={enableCardResizing}
              />
            )}
          </DragOverlay>
        </DndContext>
      ) : (
        <>
          {/* Tabs and action buttons - stacked on mobile, row on larger screens */}
          <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            {/* Group tabs (only show when grouping is enabled and groups exist) */}
            <div className="min-w-0 flex-1">
              {enableServiceGrouping && groups.length > 0 && (
                <GroupTabs
                  groups={groups}
                  selectedGroupId={selectedGroupId}
                  onSelectGroup={handleSelectGroup}
                  onCreateGroup={handleCreateGroup}
                  onEditGroup={handleEditGroup}
                  onDeleteGroup={handleDeleteGroup}
                  isEditMode={isEditMode}
                  isDraggingService={false}
                />
              )}
            </div>

            {/* Action buttons */}
            <div className="flex shrink-0 items-center gap-2">
              <button
                onClick={() => setIsEditMode(!isEditMode)}
                className="border-card-border text-text-primary hover:bg-card-hover inline-flex items-center rounded-md border px-3 py-2 text-sm font-medium transition-colors sm:px-4"
              >
                <PencilIcon className="h-4 w-4 sm:mr-2" />
                <span className="hidden sm:inline">Edit</span>
              </button>
              <Link
                href={
                  enableServiceGrouping && selectedGroupId
                    ? `/services/new?group=${selectedGroupId}`
                    : '/services/new'
                }
                className="bg-primary hover:bg-primary-hover inline-flex items-center rounded-md px-3 py-2 text-sm font-medium text-white transition-colors sm:px-4"
              >
                <PlusIcon className="h-4 w-4 sm:mr-2" />
                <span className="hidden sm:inline">Add Service</span>
              </Link>
            </div>
          </div>

          {/* Services grid (non-edit mode) */}
          <div
            className="grid grid-cols-2 gap-4 sm:grid-cols-4 xl:grid-cols-6 2xl:grid-cols-8"
            style={{ gridAutoFlow: 'dense' }}
          >
            {filteredServices.map((service) => (
              <ServiceCard
                key={service.id}
                service={service}
                openInNewTab={openInNewTab}
                enableCardResizing={enableCardResizing}
              />
            ))}
          </div>
        </>
      )}

      {/* Empty state for groups with no services */}
      {enableServiceGrouping && filteredServices.length === 0 && !isLoading && (
        <div className="bg-card border-card-border rounded-lg border p-12 text-center">
          <ServerIcon className="text-text-muted mx-auto mb-4 h-12 w-12" />
          <h3 className="text-text-primary mb-2 text-lg font-medium">No services in this group</h3>
          <p className="text-text-secondary mb-4">
            Add a service and assign it to this group to see it here.
          </p>
          <Link
            href={
              enableServiceGrouping && selectedGroupId
                ? `/services/new?group=${selectedGroupId}`
                : '/services/new'
            }
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
                ' Services in this group will be moved to the default group.'}
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
                Permanently delete all services in this group
              </span>
            </label>

            {deleteGroupServices && (
              <p className="text-error mb-4 text-sm">
                Warning: This will permanently delete all services in this group. This action cannot
                be undone.
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
