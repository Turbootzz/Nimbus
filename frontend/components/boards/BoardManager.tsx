'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { ArrowDownIcon, ArrowUpIcon, TrashIcon, XMarkIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import { refreshBoards, useBoards } from '@/hooks/useBoards'
import { useModalLayer } from '@/hooks/useModalLayer'
import { useBoardOnScreen } from '@/lib/dashboard-store'
import { defaultBoard, setLastBoard } from '@/lib/boards'
import { inputClass, labelClass } from '@/components/widgets/forms/fieldStyles'
import type { ApiResponse, Board } from '@/types'

const OPEN_EVENT = 'nimbus:open-boards'

// Opens the boards dialog from a menu, like openCommandPalette
export function openBoardManager() {
  window.dispatchEvent(new Event(OPEN_EVENT))
}

const iconButton =
  'text-text-muted hover:text-text-primary hover:bg-card-hover focus-visible:ring-primary rounded p-1.5 transition-colors focus-visible:ring-2 focus-visible:outline-none disabled:opacity-40'
const primaryButton =
  'bg-primary hover:bg-primary-hover rounded-md px-4 py-2 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed disabled:opacity-50'
const cancelButton =
  'text-text-secondary hover:text-text-primary hover:bg-card-hover rounded-md px-4 py-2 text-sm font-medium transition-colors'

// The user's boards: create, rename, reorder, share (admins) and delete.
// Mounted once in the layout; openBoardManager shows it.
export default function BoardManager() {
  const [open, setOpen] = useState(false)
  useEffect(() => {
    const show = () => setOpen(true)
    window.addEventListener(OPEN_EVENT, show)
    return () => window.removeEventListener(OPEN_EVENT, show)
  }, [])
  return open ? <BoardManagerDialog onClose={() => setOpen(false)} /> : null
}

function BoardManagerDialog({ onClose }: { onClose: () => void }) {
  const t = useTranslations('boards.manager')
  const router = useRouter()
  const { boards } = useBoards()
  const own = boards.filter((b) => b.owned)
  const onScreen = useBoardOnScreen()
  const [isAdmin, setIsAdmin] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [share, setShare] = useState(false)
  const [deleting, setDeleting] = useState<Board | null>(null)
  const dialogRef = useModalLayer(onClose, !busy && !deleting)

  useEffect(() => {
    api
      .getCurrentUser()
      .then((response) => setIsAdmin(response.data?.role === 'admin'))
      .catch(() => {})
  }, [])

  // Runs one change and reloads the boards; the API says what went wrong
  const run = async (change: () => Promise<ApiResponse<unknown>>): Promise<boolean> => {
    setBusy(true)
    setError(null)
    try {
      const response = await change()
      if (response.error) {
        setError(response.error.message)
        return false
      }
      await refreshBoards()
      return true
    } finally {
      setBusy(false)
    }
  }

  const create = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    const created = await run(() =>
      api.createBoard({ name: name.trim(), ...(share && { is_shared: true }) })
    )
    if (created) {
      setName('')
      setShare(false)
    }
  }

  const rename = (board: Board, next: string) => {
    if (next.trim() && next.trim() !== board.name) {
      run(() => api.updateBoard(board.id, { name: next.trim() }))
    }
  }

  const move = (index: number, direction: -1 | 1) => {
    const ids = own.map((b) => b.id)
    ;[ids[index], ids[index + direction]] = [ids[index + direction], ids[index]]
    run(() => api.reorderBoards(ids))
  }

  const remove = async (board: Board, moveTo: string) => {
    setDeleting(null)
    const deleted = await run(() => api.deleteBoard(board.id, moveTo || undefined))
    if (deleted && onScreen === board.id) {
      setLastBoard('')
      router.push('/dashboard')
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div
        className="absolute inset-0 bg-black/50"
        onClick={busy ? undefined : onClose}
        aria-hidden="true"
      />
      <div
        ref={dialogRef}
        tabIndex={-1}
        role="dialog"
        aria-modal="true"
        aria-labelledby="board-manager-title"
        className="bg-card border-card-border relative z-10 flex max-h-[90vh] w-full max-w-lg flex-col rounded-lg border shadow-lg focus:outline-none"
      >
        <div className="border-card-border flex items-center justify-between border-b p-4">
          <h3 id="board-manager-title" className="text-text-primary text-lg font-semibold">
            {t('title')}
          </h3>
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            className="text-text-secondary hover:text-text-primary transition-colors"
            aria-label={t('close')}
          >
            <XMarkIcon className="h-5 w-5" />
          </button>
        </div>

        <div className="space-y-4 overflow-y-auto p-4">
          {error && (
            <p role="alert" className="text-error text-sm">
              {error}
            </p>
          )}

          <ul className="space-y-2" aria-label={t('yourBoards')}>
            {own.map((board, index) => (
              <li key={board.id} className="border-card-border rounded-md border p-2">
                <div className="flex items-center gap-1">
                  <input
                    // A fresh field when the stored name changes
                    key={board.name}
                    defaultValue={board.name}
                    aria-label={t('nameOf', { name: board.name })}
                    maxLength={50}
                    disabled={busy}
                    onBlur={(e) => rename(board, e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') e.currentTarget.blur()
                    }}
                    className={`${inputClass} min-w-0 flex-1`}
                  />
                  <button
                    type="button"
                    className={iconButton}
                    onClick={() => move(index, -1)}
                    disabled={busy || index === 0}
                    aria-label={t('moveUp', { name: board.name })}
                  >
                    <ArrowUpIcon className="h-4 w-4" />
                  </button>
                  <button
                    type="button"
                    className={iconButton}
                    onClick={() => move(index, 1)}
                    disabled={busy || index === own.length - 1}
                    aria-label={t('moveDown', { name: board.name })}
                  >
                    <ArrowDownIcon className="h-4 w-4" />
                  </button>
                  {!board.is_default && (
                    <button
                      type="button"
                      className={`${iconButton} hover:text-error`}
                      onClick={() => setDeleting(board)}
                      disabled={busy}
                      aria-label={t('delete', { name: board.name })}
                    >
                      <TrashIcon className="h-4 w-4" />
                    </button>
                  )}
                </div>
                <p className="text-text-muted mt-1 px-1 text-xs">
                  {board.is_default ? t('defaultBoard') : `/board/${board.slug}`}
                </p>
                {/* Only an admin shares; anyone can stop sharing their own */}
                {!board.is_default && (isAdmin || board.is_shared) && (
                  <label className="text-text-secondary mt-1 flex items-center gap-2 px-1 text-sm">
                    <input
                      type="checkbox"
                      checked={board.is_shared}
                      disabled={busy}
                      onChange={() =>
                        run(() => api.updateBoard(board.id, { is_shared: !board.is_shared }))
                      }
                    />
                    {t('shared')}
                  </label>
                )}
              </li>
            ))}
          </ul>

          <form onSubmit={create} className="border-card-border space-y-2 border-t pt-4">
            <label htmlFor="new-board-name" className={labelClass}>
              {t('newBoard')}
            </label>
            <div className="flex gap-2">
              <input
                id="new-board-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                maxLength={50}
                placeholder={t('namePlaceholder')}
                disabled={busy}
                autoFocus
                className={`${inputClass} min-w-0 flex-1`}
              />
              <button type="submit" disabled={busy || !name.trim()} className={primaryButton}>
                {t('create')}
              </button>
            </div>
            {isAdmin && (
              <label className="text-text-secondary flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={share}
                  onChange={(e) => setShare(e.target.checked)}
                  disabled={busy}
                />
                {t('share')}
              </label>
            )}
            {isAdmin && <p className="text-text-muted text-xs">{t('shareHint')}</p>}
          </form>
        </div>
      </div>

      {deleting && (
        <DeleteBoardDialog
          board={deleting}
          targets={own.filter((b) => b.id !== deleting.id)}
          fallback={defaultBoard(boards)?.id ?? ''}
          onConfirm={(moveTo) => remove(deleting, moveTo)}
          onCancel={() => setDeleting(null)}
        />
      )}
    </div>
  )
}

interface DeleteBoardDialogProps {
  board: Board
  // The boards its tiles can move to, and the one picked first
  targets: Board[]
  fallback: string
  onConfirm: (moveTo: string) => void
  onCancel: () => void
}

// Asks where the board's tiles and groups go: another board, or away
function DeleteBoardDialog({
  board,
  targets,
  fallback,
  onConfirm,
  onCancel,
}: DeleteBoardDialogProps) {
  const t = useTranslations('boards.deleteDialog')
  const dialogRef = useModalLayer(onCancel)
  const [keep, setKeep] = useState(true)
  const [moveTo, setMoveTo] = useState(fallback)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/50" onClick={onCancel} aria-hidden="true" />
      <div
        ref={dialogRef}
        tabIndex={-1}
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="delete-board-title"
        className="bg-card border-card-border relative z-10 w-full max-w-md space-y-4 rounded-lg border p-6 shadow-lg focus:outline-none"
      >
        <h3 id="delete-board-title" className="text-text-primary text-lg font-semibold">
          {t('title', { name: board.name })}
        </h3>
        <fieldset className="space-y-2">
          <legend className="text-text-secondary mb-2 text-sm">{t('tilesLegend')}</legend>
          <label className="text-text-primary flex items-center gap-2 text-sm">
            <input type="radio" name="board-tiles" checked={keep} onChange={() => setKeep(true)} />
            {t('moveTo')}
            <select
              aria-label={t('moveToLabel')}
              value={moveTo}
              onChange={(e) => setMoveTo(e.target.value)}
              disabled={!keep}
              className={`${inputClass} w-auto`}
            >
              {targets.map((target) => (
                <option key={target.id} value={target.id}>
                  {target.name}
                </option>
              ))}
            </select>
          </label>
          <label className="text-text-primary flex items-center gap-2 text-sm">
            <input
              type="radio"
              name="board-tiles"
              checked={!keep}
              onChange={() => setKeep(false)}
            />
            {t('deleteTiles')}
          </label>
        </fieldset>
        {!keep && <p className="text-error text-sm">{t('warning')}</p>}
        <div className="flex justify-end gap-2">
          <button type="button" onClick={onCancel} className={cancelButton}>
            {t('cancel')}
          </button>
          <button
            type="button"
            onClick={() => onConfirm(keep ? moveTo : '')}
            className="bg-error hover:bg-error/80 rounded-md px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            {t('confirm')}
          </button>
        </div>
      </div>
    </div>
  )
}
