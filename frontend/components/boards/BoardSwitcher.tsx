'use client'

import { useEffect, useId, useRef, useState } from 'react'
import Link from 'next/link'
import { ChevronDownIcon, Cog6ToothIcon } from '@heroicons/react/24/outline'
import { useBoards } from '@/hooks/useBoards'
import { useConfigFile } from '@/hooks/useConfigFile'
import { useBoardOnScreen } from '@/lib/dashboard-store'
import { boardHref, defaultBoard, rememberBoard, showBoardSwitcher } from '@/lib/boards'
import { openBoardManager } from '@/components/boards/BoardManager'
import type { Board } from '@/types'

interface BoardSwitcherProps {
  // 'menu': a button with a dropdown (canvas top bar); 'list': links in the
  // classic sidebar, initials only when it is collapsed
  variant: 'menu' | 'list'
  collapsed?: boolean
  onNavigate?: () => void
}

// Marks a shared board, for its owner and its viewers alike: a viewer may
// have a board of their own with the same name
function SharedMark({ board }: { board: Board }) {
  return board.is_shared ? (
    <span className="text-text-muted shrink-0 text-xs"> · shared</span>
  ) : null
}

// Switches between the boards the user sees. Nothing shows with one board:
// the dashboard then looks as it did before boards existed.
export default function BoardSwitcher({ variant, collapsed, onNavigate }: BoardSwitcherProps) {
  const { boards } = useBoards()
  const onScreen = useBoardOnScreen()
  // A dashboard locked by a config file has no board to create or change
  const locked = useConfigFile()?.read_only ?? false
  const [open, setOpen] = useState(false)
  const menuId = useId()
  const rootRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!open) return
    const onClick = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [open])

  if (!showBoardSwitcher(boards)) return null

  const activeId = onScreen === '' ? defaultBoard(boards)?.id : onScreen
  const active = boards.find((b) => b.id === activeId)
  const choose = (board: Board) => {
    rememberBoard(board)
    setOpen(false)
    onNavigate?.()
  }
  const manage = () => {
    setOpen(false)
    onNavigate?.()
    openBoardManager()
  }

  if (variant === 'list') {
    return (
      <div role="group" aria-label="Boards" className="space-y-1 pt-3">
        {!collapsed && (
          <p className="text-text-muted px-3 text-xs font-medium tracking-wide uppercase">Boards</p>
        )}
        {boards.map((board) => {
          const current = board.id === activeId
          return (
            <Link
              key={board.id}
              href={boardHref(board, boards)}
              onClick={() => choose(board)}
              aria-current={current ? 'page' : undefined}
              title={collapsed ? board.name : undefined}
              className={`flex items-center gap-2 rounded-md px-3 py-1.5 text-sm transition-colors ${
                collapsed ? 'justify-center' : ''
              } ${
                current
                  ? 'bg-card-border text-text-primary font-medium'
                  : 'text-text-secondary hover:bg-card-border hover:text-text-primary'
              }`}
            >
              {collapsed ? (
                <span aria-hidden="true" className="w-5 text-center font-semibold">
                  {board.name.charAt(0).toUpperCase()}
                </span>
              ) : (
                <>
                  <span className="truncate">{board.name}</span>
                  <SharedMark board={board} />
                </>
              )}
              {collapsed && (
                <span className="sr-only">
                  {board.name}
                  <SharedMark board={board} />
                </span>
              )}
            </Link>
          )
        })}
        {!locked && !collapsed && (
          <button
            type="button"
            onClick={manage}
            className="text-text-muted hover:text-text-primary hover:bg-card-border flex w-full items-center rounded-md px-3 py-1.5 text-sm transition-colors"
          >
            Manage boards
          </button>
        )}
      </div>
    )
  }

  return (
    <div
      ref={rootRef}
      className="relative min-w-0"
      onKeyDown={(e) => {
        if (e.key === 'Escape' && open) {
          e.stopPropagation()
          setOpen(false)
          triggerRef.current?.focus()
        }
      }}
    >
      <button
        ref={triggerRef}
        type="button"
        aria-haspopup="true"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => setOpen(!open)}
        className="text-text-primary hover:bg-card-border focus-visible:ring-primary flex w-full max-w-40 items-center gap-1 rounded-md px-2 py-1.5 text-sm font-medium transition-colors focus-visible:ring-2 focus-visible:outline-none sm:max-w-56"
      >
        <span className="sr-only">Board: </span>
        <span className="truncate">{active?.name ?? 'Boards'}</span>
        <ChevronDownIcon className="text-text-muted h-4 w-4 shrink-0" aria-hidden="true" />
      </button>
      {open && (
        <div
          id={menuId}
          className="bg-card border-card-border absolute left-0 z-40 mt-2 w-56 rounded-md border py-1 shadow-lg"
        >
          {boards.map((board) => (
            <Link
              key={board.id}
              href={boardHref(board, boards)}
              onClick={() => choose(board)}
              aria-current={board.id === activeId ? 'page' : undefined}
              className={`hover:bg-card-hover block truncate px-4 py-2 text-sm ${
                board.id === activeId ? 'text-primary font-medium' : 'text-text-primary'
              }`}
            >
              {board.name}
              <SharedMark board={board} />
            </Link>
          ))}
          {!locked && (
            <>
              <div className="bg-card-border my-1 h-px" />
              <button
                type="button"
                onClick={manage}
                className="text-text-secondary hover:bg-card-hover hover:text-text-primary flex w-full items-center px-4 py-2 text-sm"
              >
                <Cog6ToothIcon className="text-text-muted mr-3 h-5 w-5" aria-hidden="true" />
                Manage boards
              </button>
            </>
          )}
        </div>
      )}
    </div>
  )
}
