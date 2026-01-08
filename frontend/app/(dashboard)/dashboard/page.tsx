'use client'

import { useState, useEffect } from 'react'
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
import type { Service, CardSize } from '@/types'
import { useTheme } from '@/contexts/ThemeContext'
import { sizeToGridSpan } from '@/lib/card-utils'
import ServiceCard from '@/components/ServiceCard'

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
  const { openInNewTab, enableCardResizing } = useTheme()
  const [services, setServices] = useState<Service[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isEditMode, setIsEditMode] = useState(false)
  const [activeId, setActiveId] = useState<string | null>(null)
  const [stats, setStats] = useState({
    total: 0,
    online: 0,
    offline: 0,
    avgResponseTime: 0,
  })

  // DnD sensors for mouse and touch
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 8 },
    }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: 150, tolerance: 5 },
    })
  )

  useEffect(() => {
    fetchServices()
  }, [])

  useEffect(() => {
    // Calculate stats
    const online = services.filter((s) => s.status === 'online').length
    const offline = services.filter((s) => s.status === 'offline').length
    const responseTimes = services
      .filter((s) => s.response_time !== undefined && s.response_time !== null)
      .map((s) => s.response_time as number)
    const avgResponseTime =
      responseTimes.length > 0
        ? Math.round(responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length)
        : 0

    setStats({
      total: services.length,
      online,
      offline,
      avgResponseTime,
    })
  }, [services])

  const fetchServices = async () => {
    setIsLoading(true)
    try {
      const response = await api.getServices()

      if (response.data) {
        setServices(response.data)
      }
    } catch (error) {
      console.error('Failed to fetch services:', error)
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
    if (!over || active.id === over.id) return

    const oldIndex = services.findIndex((s) => s.id === active.id)
    const newIndex = services.findIndex((s) => s.id === over.id)
    if (oldIndex === -1 || newIndex === -1) return

    // Capture current state for rollback
    const previousServices = services

    // Optimistic update
    const reordered = arrayMove(services, oldIndex, newIndex)
    setServices(reordered)

    // Persist to backend
    try {
      await api.reorderServices({
        services: reordered.map((s, i) => ({ id: s.id, position: i })),
      })
    } catch (error) {
      console.error('Failed to save order:', error)
      setServices(previousServices)
    }
  }

  const handleSizeChange = async (id: string, newSize: CardSize) => {
    // Capture current state for rollback
    const previousServices = services

    // Optimistic update
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
      {/* Page header */}
      <div className="mb-8">
        <h1 className="text-text-primary text-3xl font-bold">Welcome to Nimbus</h1>
        <p className="text-text-secondary mt-2">Monitor and manage your homelab services</p>
      </div>

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

      {/* Services grid */}
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-text-primary text-xl font-semibold">Services</h2>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setIsEditMode(!isEditMode)}
            className={`inline-flex items-center rounded-md px-4 py-2 text-sm font-medium transition-colors ${
              isEditMode
                ? 'bg-success hover:bg-success/80 text-white'
                : 'border-card-border text-text-primary hover:bg-card-hover border'
            }`}
          >
            {isEditMode ? (
              <>
                <CheckIcon className="mr-2 h-4 w-4" />
                Done
              </>
            ) : (
              <>
                <PencilIcon className="mr-2 h-4 w-4" />
                Edit
              </>
            )}
          </button>
          <Link
            href="/services/new"
            className="bg-primary hover:bg-primary-hover inline-flex items-center rounded-md px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            <PlusIcon className="mr-2 h-4 w-4" />
            Add Service
          </Link>
        </div>
      </div>

      {isEditMode ? (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          onDragCancel={handleDragCancel}
        >
          <SortableContext items={services.map((s) => s.id)} strategy={rectSortingStrategy}>
            <div
              className="grid grid-cols-2 gap-4 sm:grid-cols-4 xl:grid-cols-6 2xl:grid-cols-8"
              style={{ gridAutoFlow: 'dense' }}
            >
              {services.map((service) => (
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
            {(() => {
              const activeService = activeId ? services.find((s) => s.id === activeId) : null
              return activeService ? (
                <ServiceCard
                  service={activeService}
                  openInNewTab={openInNewTab}
                  isEditMode={true}
                  onSizeChange={() => {}}
                  isDragging={true}
                  enableCardResizing={enableCardResizing}
                />
              ) : null
            })()}
          </DragOverlay>
        </DndContext>
      ) : (
        <div
          className="grid grid-cols-2 gap-4 sm:grid-cols-4 xl:grid-cols-6 2xl:grid-cols-8"
          style={{ gridAutoFlow: 'dense' }}
        >
          {services.map((service) => (
            <ServiceCard
              key={service.id}
              service={service}
              openInNewTab={openInNewTab}
              enableCardResizing={enableCardResizing}
            />
          ))}

          {/* Add new service card - spans 2 cols like standard card */}
          <Link
            href="/services/new"
            className="bg-card border-card-border hover:border-primary hover:bg-primary-light col-span-2 flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed p-6 transition-all"
          >
            <PlusIcon className="text-text-muted mb-2 h-12 w-12" />
            <span className="text-text-secondary">Add Service</span>
          </Link>
        </div>
      )}
    </div>
  )
}
