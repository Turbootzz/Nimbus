'use client'

import { useRef, useEffect } from 'react'
import { useDroppable } from '@dnd-kit/core'
import { useSortable, SortableContext, horizontalListSortingStrategy } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { PlusIcon, PencilIcon, TrashIcon, EyeSlashIcon } from '@heroicons/react/24/outline'
import type { Group } from '@/types'
import ScrollArea from '@/components/ui/ScrollArea'

interface GroupTabsProps {
  groups: Group[]
  selectedGroupId: string | null
  onSelectGroup: (groupId: string | null) => void
  onCreateGroup: () => void
  onEditGroup: (group: Group) => void
  onDeleteGroup: (group: Group) => void
  isEditMode: boolean
  isDraggingService?: boolean
  isDraggingTab?: boolean
}

// Sortable + Droppable tab component
// - Sortable: allows reordering tabs in edit mode
// - Droppable: accepts service drops to move services between groups
function SortableDroppableTab({
  group,
  isSelected,
  isEditMode,
  isDraggingService,
  isDraggingTab,
  onSelect,
  onEdit,
  onDelete,
}: {
  group: Group
  isSelected: boolean
  isEditMode: boolean
  isDraggingService?: boolean
  isDraggingTab?: boolean
  onSelect: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  // Sortable hook for tab reordering (only in edit mode)
  const {
    attributes,
    listeners,
    setNodeRef: setSortableRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: group.id,
    disabled: !isEditMode,
  })

  // Droppable hook for receiving service drops (group- prefix to distinguish)
  const { setNodeRef: setDroppableRef, isOver } = useDroppable({
    id: `group-${group.id}`,
    data: { type: 'group', groupId: group.id },
  })

  // Combine refs for both sortable and droppable
  const setNodeRef = (node: HTMLElement | null) => {
    setSortableRef(node)
    setDroppableRef(node)
  }

  // Apply transform for tab dragging
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  // Show drop indicator when a service (not a tab) is being dragged over this tab
  const showDropIndicator = isDraggingService && !isDraggingTab && isOver

  // Handle keyboard navigation for accessibility
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onSelect()
    }
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...(isEditMode ? { ...attributes, ...listeners } : {})}
      role="tab"
      tabIndex={0}
      aria-selected={isSelected}
      data-group-id={group.id}
      className={`group relative flex shrink-0 items-center gap-2 rounded-t-lg px-3 py-2 whitespace-nowrap transition-colors sm:px-4 ${
        isSelected
          ? 'bg-card border-card-border border-b-card border-t border-r border-l'
          : 'bg-card-hover/50 hover:bg-card-hover'
      } ${showDropIndicator ? 'ring-primary ring-2' : ''} ${isDragging ? 'z-50' : ''} cursor-pointer`}
      onClick={onSelect}
      onKeyDown={handleKeyDown}
    >
      {/* Color indicator */}
      <div className="h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: group.color }} />

      {/* Group name */}
      <span
        className={`text-sm font-medium ${isSelected ? 'text-text-primary' : 'text-text-secondary'}`}
      >
        {group.name}
      </span>

      {/* Monitoring disabled indicator */}
      {!group.monitoring_enabled && (
        <EyeSlashIcon
          className="text-text-muted h-3.5 w-3.5 shrink-0"
          title="Monitoring disabled for this group"
        />
      )}

      {/* Edit/delete buttons in edit mode */}
      {isEditMode && (
        <div className="ml-2 flex items-center gap-1">
          <button
            type="button"
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
              type="button"
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
  isDraggingTab,
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
      <ScrollArea
        ref={scrollRef}
        orientation="horizontal"
        className="border-card-border -mx-4 flex items-end gap-1 border-b px-4 pb-0 sm:mx-0 sm:px-0"
      >
        <SortableContext items={groups.map((g) => g.id)} strategy={horizontalListSortingStrategy}>
          {groups.map((group) => (
            <SortableDroppableTab
              key={group.id}
              group={group}
              isSelected={selectedGroupId === group.id}
              isEditMode={isEditMode}
              isDraggingService={isDraggingService}
              isDraggingTab={isDraggingTab}
              onSelect={() => onSelectGroup(group.id)}
              onEdit={() => onEditGroup(group)}
              onDelete={() => onDeleteGroup(group)}
            />
          ))}
        </SortableContext>

        {/* Add group button - outside SortableContext */}
        {isEditMode && (
          <button
            type="button"
            onClick={onCreateGroup}
            className="text-text-muted hover:text-primary hover:bg-card-hover mb-0 flex items-center gap-1 rounded-t-lg px-3 py-2 transition-colors"
            title="Add new group"
          >
            <PlusIcon className="h-4 w-4" />
            <span className="text-sm">Add</span>
          </button>
        )}
      </ScrollArea>
    </div>
  )
}
