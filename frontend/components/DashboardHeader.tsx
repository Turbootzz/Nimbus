'use client'

import Link from 'next/link'
import { PlusIcon, PencilIcon, CheckIcon } from '@heroicons/react/24/outline'
import type { Group } from '@/types'
import GroupTabs from '@/components/GroupTabs'

interface DashboardHeaderProps {
  groups: Group[]
  selectedGroupId: string | null
  onSelectGroup: (groupId: string | null) => void
  onCreateGroup: () => void
  onEditGroup: (group: Group) => void
  onDeleteGroup: (group: Group) => void
  isEditMode: boolean
  onToggleEditMode: () => void
  enableServiceGrouping: boolean
  activeId: string | null
  isDraggingTab: boolean
  addServiceHref: string
}

export default function DashboardHeader({
  groups,
  selectedGroupId,
  onSelectGroup,
  onCreateGroup,
  onEditGroup,
  onDeleteGroup,
  isEditMode,
  onToggleEditMode,
  enableServiceGrouping,
  activeId,
  isDraggingTab,
  addServiceHref,
}: DashboardHeaderProps) {
  return (
    <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0 flex-1">
        {enableServiceGrouping && groups.length > 0 && (
          <GroupTabs
            groups={groups}
            selectedGroupId={selectedGroupId}
            onSelectGroup={onSelectGroup}
            onCreateGroup={onCreateGroup}
            onEditGroup={onEditGroup}
            onDeleteGroup={onDeleteGroup}
            isEditMode={isEditMode}
            isDraggingService={!!activeId && !isDraggingTab}
            isDraggingTab={isDraggingTab}
          />
        )}
      </div>

      <div className="flex shrink-0 items-center gap-2">
        {isEditMode ? (
          <button
            onClick={onToggleEditMode}
            className="bg-success hover:bg-success/80 inline-flex items-center rounded-md px-3 py-2 text-sm font-medium text-white transition-colors sm:px-4"
          >
            <CheckIcon className="h-4 w-4 sm:mr-2" />
            <span className="hidden sm:inline">Done</span>
          </button>
        ) : (
          <button
            onClick={onToggleEditMode}
            className="border-card-border text-text-primary hover:bg-card-hover inline-flex items-center rounded-md border px-3 py-2 text-sm font-medium transition-colors sm:px-4"
          >
            <PencilIcon className="h-4 w-4 sm:mr-2" />
            <span className="hidden sm:inline">Edit</span>
          </button>
        )}
        <Link
          href={addServiceHref}
          className="bg-primary hover:bg-primary-hover inline-flex items-center rounded-md px-3 py-2 text-sm font-medium text-white transition-colors sm:px-4"
        >
          <PlusIcon className="h-4 w-4 sm:mr-2" />
          <span className="hidden sm:inline">Add Service</span>
        </Link>
      </div>
    </div>
  )
}
