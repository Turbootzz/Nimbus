'use client'

import { useState, type ReactNode } from 'react'
import { useDroppable } from '@dnd-kit/core'
import { useTranslations } from 'next-intl'
import {
  ArrowDownIcon,
  ArrowUpIcon,
  ChevronDownIcon,
  EyeSlashIcon,
  PencilIcon,
  PlusIcon,
  TrashIcon,
} from '@heroicons/react/24/outline'
import type { Group } from '@/types'
import { GROUP_DROP_PREFIX, type TileSection } from '@/lib/tiles'

const STORAGE_KEY = 'nimbus-collapsed-sections'

function readCollapsed(): string[] {
  try {
    const stored: unknown = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '[]')
    return Array.isArray(stored) ? stored.filter((id) => typeof id === 'string') : []
  } catch {
    return []
  }
}

const actionClass =
  'text-text-muted hover:text-primary wallpaper-surface rounded p-1 transition-colors disabled:opacity-40 disabled:hover:text-text-muted'

interface GroupSectionsProps {
  sections: TileSection[]
  isEditMode: boolean
  // A tile is being dragged, so sections show where it can be dropped
  isDraggingTile: boolean
  onCreateGroup: () => void
  onEditGroup: (group: Group) => void
  onDeleteGroup: (group: Group) => void
  // Swaps the group with its neighbour: -1 up, 1 down
  onMoveGroup: (group: Group, direction: -1 | 1) => void
  // Renders the tiles of a section
  children: (section: TileSection) => ReactNode
}

// All groups stacked as sections, each under its own header. The alternative
// to GroupTabs, which shows one group at a time.
export default function GroupSections({
  sections,
  isEditMode,
  isDraggingTile,
  onCreateGroup,
  onEditGroup,
  onDeleteGroup,
  onMoveGroup,
  children,
}: GroupSectionsProps) {
  const t = useTranslations('dashboard.groups')
  const [collapsed, setCollapsed] = useState(readCollapsed)

  const toggle = (groupId: string) => {
    const next = collapsed.includes(groupId)
      ? collapsed.filter((id) => id !== groupId)
      : [...collapsed, groupId]
    setCollapsed(next)
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
  }

  return (
    <div>
      {sections.map((section, index) => (
        <GroupSection
          key={section.group.id}
          section={section}
          collapsed={collapsed.includes(section.group.id)}
          onToggle={() => toggle(section.group.id)}
          isEditMode={isEditMode}
          isDraggingTile={isDraggingTile}
          isFirst={index === 0}
          isLast={index === sections.length - 1}
          onEditGroup={onEditGroup}
          onDeleteGroup={onDeleteGroup}
          onMoveGroup={onMoveGroup}
        >
          {children(section)}
        </GroupSection>
      ))}

      {isEditMode && (
        <button
          type="button"
          onClick={onCreateGroup}
          className="text-text-muted hover:text-primary border-card-border wallpaper-surface mb-6 flex items-center gap-1 rounded-md border border-dashed px-3 py-2 text-sm transition-colors"
        >
          <PlusIcon className="h-4 w-4" />
          {t('sections.addGroup')}
        </button>
      )}
    </div>
  )
}

interface GroupSectionProps extends Omit<
  GroupSectionsProps,
  'sections' | 'onCreateGroup' | 'children'
> {
  section: TileSection
  collapsed: boolean
  onToggle: () => void
  isFirst: boolean
  isLast: boolean
  children: ReactNode
}

function GroupSection({
  section: { group, tiles },
  collapsed,
  onToggle,
  isEditMode,
  isDraggingTile,
  isFirst,
  isLast,
  onEditGroup,
  onDeleteGroup,
  onMoveGroup,
  children,
}: GroupSectionProps) {
  const t = useTranslations('dashboard.groups')
  // The whole section takes drops, so an empty or collapsed one can be filled
  const { setNodeRef, isOver } = useDroppable({
    id: GROUP_DROP_PREFIX + group.id,
    data: { type: 'group', groupId: group.id },
    disabled: !isEditMode,
  })
  const bodyId = `section-${group.id}`

  return (
    <section
      ref={setNodeRef}
      aria-label={group.name}
      className={`mb-6 rounded-lg ${isDraggingTile && isOver ? 'ring-primary ring-2' : ''}`}
    >
      <div className="mb-2 flex items-center gap-1">
        <h2 className="min-w-0">
          <button
            type="button"
            onClick={onToggle}
            aria-expanded={!collapsed}
            aria-controls={bodyId}
            className="wallpaper-surface hover:bg-card-hover wallpaper:hover:bg-card-hover flex max-w-full items-center gap-2 rounded-md px-2 py-1 transition-colors"
          >
            <ChevronDownIcon
              className={`text-text-muted h-4 w-4 shrink-0 transition-transform ${collapsed ? '-rotate-90' : ''}`}
            />
            <span
              className="h-2.5 w-2.5 shrink-0 rounded-full"
              style={{ backgroundColor: group.color }}
            />
            <span className="text-text-secondary truncate text-xs font-semibold tracking-wider uppercase">
              {group.name}
            </span>
            <span className="text-text-muted text-xs tabular-nums">{tiles.length}</span>
          </button>
        </h2>

        {!group.monitoring_enabled && (
          <EyeSlashIcon
            className="text-text-muted h-3.5 w-3.5 shrink-0"
            title={t('monitoringDisabled')}
          />
        )}

        {isEditMode && (
          <div className="ml-1 flex items-center gap-1">
            <button
              type="button"
              onClick={() => onMoveGroup(group, -1)}
              disabled={isFirst}
              className={actionClass}
              aria-label={t('sections.moveUp', { name: group.name })}
            >
              <ArrowUpIcon className="h-3.5 w-3.5" />
            </button>
            <button
              type="button"
              onClick={() => onMoveGroup(group, 1)}
              disabled={isLast}
              className={actionClass}
              aria-label={t('sections.moveDown', { name: group.name })}
            >
              <ArrowDownIcon className="h-3.5 w-3.5" />
            </button>
            <button
              type="button"
              onClick={() => onEditGroup(group)}
              className={actionClass}
              aria-label={t('sections.edit', { name: group.name })}
            >
              <PencilIcon className="h-3.5 w-3.5" />
            </button>
            {!group.is_default && (
              <button
                type="button"
                onClick={() => onDeleteGroup(group)}
                className={`${actionClass} hover:text-error`}
                aria-label={t('sections.delete', { name: group.name })}
              >
                <TrashIcon className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
        )}
      </div>

      {!collapsed && (
        <div id={bodyId}>
          {tiles.length > 0 ? (
            children
          ) : (
            <p className="text-text-muted border-card-border wallpaper-surface rounded-lg border border-dashed px-4 py-3 text-sm">
              {isEditMode ? t('sections.dropHere') : t('sections.empty')}
            </p>
          )}
        </div>
      )}
    </section>
  )
}
