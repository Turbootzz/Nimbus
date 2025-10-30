'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { PlusIcon } from '@heroicons/react/24/outline'
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
import { arrayMove, SortableContext, rectSortingStrategy } from '@dnd-kit/sortable'
import { api } from '@/lib/api'
import type { Service } from '@/types'
import { DraggableServiceManagementCard } from '@/components/DraggableServiceManagementCard'
import { ServiceManagementCardPresentation } from '@/components/ServiceManagementCardPresentation'

export default function ServicesPage() {
  const [services, setServices] = useState<Service[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [activeId, setActiveId] = useState<string | null>(null)

  // Configure sensors for both mouse/touch input
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8, // 8px movement required before drag starts
      },
    }),
    useSensor(TouchSensor, {
      activationConstraint: {
        delay: 150, // 150ms press before drag starts on touch (shorter delay)
        tolerance: 5, // 5px movement tolerance
      },
    })
  )

  useEffect(() => {
    fetchServices()
  }, [])

  const fetchServices = async () => {
    setIsLoading(true)
    setError('')

    try {
      const response = await api.getServices()

      if (response.error) {
        setError(response.error.message)
      } else if (response.data) {
        setServices(response.data)
      }
    } catch (error) {
      console.error('Failed to fetch services:', error)
      const message =
        error instanceof Error ? error.message : 'Unable to load services. Please try again.'
      setError(message)
    } finally {
      setIsLoading(false)
    }
  }

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Are you sure you want to delete "${name}"?`)) {
      return
    }

    try {
      const response = await api.deleteService(id)

      if (response.error) {
        alert(`Failed to delete service: ${response.error.message}`)
      } else {
        // Remove from list
        setServices(services.filter((s) => s.id !== id))
      }
    } catch (error) {
      console.error('Failed to delete service:', error)
      const message =
        error instanceof Error ? error.message : 'Unable to delete service. Please try again.'
      alert(`Failed to delete service: ${message}`)
    }
  }

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(event.active.id as string)
  }

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event

    setActiveId(null)

    if (!over || active.id === over.id) {
      return
    }

    const oldIndex = services.findIndex((s) => s.id === active.id)
    const newIndex = services.findIndex((s) => s.id === over.id)

    if (oldIndex === -1 || newIndex === -1) {
      return
    }

    // Optimistically update UI
    const reorderedServices = arrayMove(services, oldIndex, newIndex)
    setServices(reorderedServices)

    // Update positions on backend
    const updatedPositions = reorderedServices.map((service, index) => ({
      id: service.id,
      position: index,
    }))

    try {
      const response = await api.reorderServices({ services: updatedPositions })

      if (response.error) {
        console.error('Failed to save order:', response.error.message || response.error)
        // Revert on error
        setServices(services)
      }
    } catch (error) {
      console.error('Failed to save order:', error)
      // Revert on error
      setServices(services)
    }
  }

  const handleDragCancel = () => {
    setActiveId(null)
  }

  if (isLoading) {
    return (
      <div className="flex min-h-96 items-center justify-center">
        <div className="text-center">
          <div className="border-primary mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-t-2 border-b-2"></div>
          <p className="text-text-secondary">Loading services...</p>
        </div>
      </div>
    )
  }

  const activeService = services.find((s) => s.id === activeId)

  return (
    <div>
      {/* Page header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-text-primary text-3xl font-bold">Services</h1>
          <p className="text-text-secondary mt-1">Manage your homelab services</p>
        </div>
        <Link
          href="/services/new"
          className="bg-primary hover:bg-primary-hover inline-flex items-center rounded-md px-4 py-2 text-sm font-medium text-white transition-colors"
        >
          <PlusIcon className="mr-2 h-4 w-4" />
          Add Service
        </Link>
      </div>

      {/* Error message */}
      {error && (
        <div
          className="mb-4 rounded-lg border p-3 text-sm"
          style={{
            backgroundColor: 'var(--color-error)',
            borderColor: 'var(--color-error)',
            color: 'white',
            opacity: 0.9,
          }}
        >
          {error}
        </div>
      )}

      {/* Services grid */}
      {services.length === 0 ? (
        <div className="bg-card border-card-border flex flex-col items-center justify-center rounded-lg border p-12 text-center">
          <div className="text-text-muted mb-4 text-6xl">🔗</div>
          <h3 className="text-text-primary mb-2 text-xl font-semibold">No services yet</h3>
          <p className="text-text-secondary mb-6 max-w-md">
            Get started by adding your first service. Services can be websites, apps, or any URL you
            want to monitor.
          </p>
          <Link
            href="/services/new"
            className="bg-primary hover:bg-primary-hover inline-flex items-center rounded-md px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            <PlusIcon className="mr-2 h-4 w-4" />
            Add Your First Service
          </Link>
        </div>
      ) : (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          onDragCancel={handleDragCancel}
        >
          <SortableContext items={services.map((s) => s.id)} strategy={rectSortingStrategy}>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {services.map((service) => (
                <DraggableServiceManagementCard
                  key={service.id}
                  service={service}
                  onDelete={handleDelete}
                />
              ))}
            </div>
          </SortableContext>

          <DragOverlay>
            {activeService ? (
              <ServiceManagementCardPresentation
                service={activeService}
                onDelete={handleDelete}
                style={{ opacity: 0.5 }}
              />
            ) : null}
          </DragOverlay>
        </DndContext>
      )}
    </div>
  )
}
