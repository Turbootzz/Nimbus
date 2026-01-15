'use client'

import { useState, useRef, useEffect } from 'react'
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
import {
  SortableContext,
  horizontalListSortingStrategy,
  useSortable,
  arrayMove,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { PlusIcon, PencilIcon, TrashIcon } from '@heroicons/react/24/outline'
import type { Group } from '@/types'

interface GroupTabsProps {
  groups: Group[]
  selectedGroupId: string | null
  onSelectGroup: (groupId: string | null) => void
  onCreateGroup: () => void
  onEditGroup: (group: Group) => void
  onDeleteGroup: (group: Group) => void
  onReorderGroups: (groups: Group[]) => void
  isEditMode: boolean
}

// Individual sortable tab component
function SortableTab({
  group,
  isSelected,
  isEditMode,
  onSelect,
  onEdit,
  onDelete,
}: {
  group: Group
  isSelected: boolean
  isEditMode: boolean
  onSelect: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: group.id,
    disabled: !isEditMode,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`group relative flex items-center gap-2 rounded-t-lg px-4 py-2 transition-colors ${
        isSelected
          ? 'bg-card border-card-border border-b-card border-t border-r border-l'
          : 'bg-card-hover/50 hover:bg-card-hover'
      } ${isEditMode ? 'cursor-grab' : 'cursor-pointer'}`}
      onClick={isEditMode ? undefined : onSelect}
      {...(isEditMode ? { ...attributes, ...listeners } : {})}
    >
      {/* Color indicator */}
      <div className="h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: group.color }} />

      {/* Group name */}
      <span
        className={`text-sm font-medium ${isSelected ? 'text-text-primary' : 'text-text-secondary'}`}
      >
        {group.name}
      </span>

      {/* Edit/delete buttons in edit mode */}
      {isEditMode && (
        <div className="ml-2 flex items-center gap-1">
          <button
            onClick={(e) => {
              e.stopPropagation()
              onEdit()
            }}
            className="text-text-muted hover:text-primary p-1 transition-colors"
            title="Edit group"
          >
            <PencilIcon className="h-3.5 w-3.5" />
          </button>
          {!group.is_default && (
            <button
              onClick={(e) => {
                e.stopPropagation()
                onDelete()
              }}
              className="text-text-muted hover:text-error p-1 transition-colors"
              title="Delete group"
            >
              <TrashIcon className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      )}
    </div>
  )
}

// Static tab (used for drag overlay)
function TabOverlay({ group }: { group: Group }) {
  return (
    <div className="bg-card border-card-border flex items-center gap-2 rounded-t-lg border px-4 py-2 shadow-lg">
      <div className="h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: group.color }} />
      <span className="text-text-primary text-sm font-medium">{group.name}</span>
    </div>
  )
}

export default function GroupTabs({
  groups,
  selectedGroupId,
  onSelectGroup,
  onCreateGroup,
  onEditGroup,
  onDeleteGroup,
  onReorderGroups,
  isEditMode,
}: GroupTabsProps) {
  const [activeId, setActiveId] = useState<string | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  // DnD sensors
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 8 },
    }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: 150, tolerance: 5 },
    })
  )

  // Find active group for overlay
  const activeGroup = activeId ? groups.find((g) => g.id === activeId) : null

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(event.active.id as string)
  }

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event
    setActiveId(null)

    if (!over || active.id === over.id) return

    const oldIndex = groups.findIndex((g) => g.id === active.id)
    const newIndex = groups.findIndex((g) => g.id === over.id)

    if (oldIndex === -1 || newIndex === -1) return

    const reordered = arrayMove(groups, oldIndex, newIndex)
    onReorderGroups(reordered)
  }

  const handleDragCancel = () => {
    setActiveId(null)
  }

  // Scroll selected tab into view
  useEffect(() => {
    if (selectedGroupId && scrollRef.current) {
      const selectedTab = scrollRef.current.querySelector(`[data-group-id="${selectedGroupId}"]`)
      if (selectedTab) {
        selectedTab.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'center' })
      }
    }
  }, [selectedGroupId])

  if (groups.length === 0) {
    return null
  }

  return (
    <div className="mb-4">
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
        onDragCancel={handleDragCancel}
      >
        <div
          ref={scrollRef}
          className="border-card-border flex items-end gap-1 overflow-x-auto border-b pb-0"
        >
          <SortableContext items={groups.map((g) => g.id)} strategy={horizontalListSortingStrategy}>
            {groups.map((group) => (
              <div key={group.id} data-group-id={group.id}>
                <SortableTab
                  group={group}
                  isSelected={selectedGroupId === group.id}
                  isEditMode={isEditMode}
                  onSelect={() => onSelectGroup(group.id)}
                  onEdit={() => onEditGroup(group)}
                  onDelete={() => onDeleteGroup(group)}
                />
              </div>
            ))}
          </SortableContext>

          {/* Add group button */}
          {isEditMode && (
            <button
              onClick={onCreateGroup}
              className="text-text-muted hover:text-primary hover:bg-card-hover mb-0 flex items-center gap-1 rounded-t-lg px-3 py-2 transition-colors"
              title="Add new group"
            >
              <PlusIcon className="h-4 w-4" />
              <span className="text-sm">Add</span>
            </button>
          )}
        </div>

        <DragOverlay dropAnimation={null}>
          {activeGroup && <TabOverlay group={activeGroup} />}
        </DragOverlay>
      </DndContext>
    </div>
  )
}
