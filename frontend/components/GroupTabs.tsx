'use client'

import { useRef, useEffect } from 'react'
import { useDroppable } from '@dnd-kit/core'
import { PlusIcon, PencilIcon, TrashIcon } from '@heroicons/react/24/outline'
import type { Group } from '@/types'

interface GroupTabsProps {
  groups: Group[]
  selectedGroupId: string | null
  onSelectGroup: (groupId: string | null) => void
  onCreateGroup: () => void
  onEditGroup: (group: Group) => void
  onDeleteGroup: (group: Group) => void
  isEditMode: boolean
  isDraggingService?: boolean
}

// Droppable tab component - accepts service drops to move services between groups
function DroppableTab({
  group,
  isSelected,
  isEditMode,
  isDraggingService,
  onSelect,
  onEdit,
  onDelete,
}: {
  group: Group
  isSelected: boolean
  isEditMode: boolean
  isDraggingService?: boolean
  onSelect: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  // Use droppable with group- prefix to distinguish from service IDs
  const { setNodeRef, isOver } = useDroppable({
    id: `group-${group.id}`,
    data: { type: 'group', groupId: group.id },
  })

  // Highlight when a service is being dragged over this tab
  const showDropIndicator = isDraggingService && isOver

  return (
    <div
      ref={setNodeRef}
      data-group-id={group.id}
      className={`group relative flex shrink-0 items-center gap-2 rounded-t-lg px-3 py-2 whitespace-nowrap transition-colors sm:px-4 ${
        isSelected
          ? 'bg-card border-card-border border-b-card border-t border-r border-l'
          : 'bg-card-hover/50 hover:bg-card-hover'
      } ${showDropIndicator ? 'ring-primary ring-2' : ''} cursor-pointer`}
      onClick={onSelect}
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

export default function GroupTabs({
  groups,
  selectedGroupId,
  onSelectGroup,
  onCreateGroup,
  onEditGroup,
  onDeleteGroup,
  isEditMode,
  isDraggingService,
}: GroupTabsProps) {
  const scrollRef = useRef<HTMLDivElement>(null)

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
    <div>
      <div
        ref={scrollRef}
        className="border-card-border -mx-4 flex items-end gap-1 overflow-x-auto border-b px-4 pb-0 sm:mx-0 sm:px-0"
      >
        {groups.map((group) => (
          <DroppableTab
            key={group.id}
            group={group}
            isSelected={selectedGroupId === group.id}
            isEditMode={isEditMode}
            isDraggingService={isDraggingService}
            onSelect={() => onSelectGroup(group.id)}
            onEdit={() => onEditGroup(group)}
            onDelete={() => onDeleteGroup(group)}
          />
        ))}

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
    </div>
  )
}
