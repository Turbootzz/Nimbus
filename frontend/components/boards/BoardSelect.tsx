'use client'

import { useBoards } from '@/hooks/useBoards'
import { defaultBoard, moveTargets } from '@/lib/boards'
import { inputClass, labelClass } from '@/components/widgets/forms/fieldStyles'

interface BoardSelectProps {
  id: string
  // The board the tile is on
  currentBoardId?: string
  // '' keeps the tile where it is, a board ID moves it there
  value: string
  onChange: (boardId: string) => void
  disabled?: boolean
}

// Moves a service or widget to another of the user's boards. Nothing shows
// for a user with one board of their own.
export default function BoardSelect({
  id,
  currentBoardId,
  value,
  onChange,
  disabled,
}: BoardSelectProps) {
  const { boards } = useBoards()
  const targets = moveTargets(boards, currentBoardId)
  if (targets.length === 0) return null

  const current = boards.find((b) => b.id === currentBoardId) ?? defaultBoard(boards)
  return (
    <div>
      <label htmlFor={id} className={labelClass}>
        Board
      </label>
      <select
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        className={inputClass}
        aria-describedby={`${id}-hint`}
      >
        <option value="">{current ? `${current.name} (this board)` : 'This board'}</option>
        {targets.map((board) => (
          <option key={board.id} value={board.id}>
            {board.name}
          </option>
        ))}
      </select>
      <p id={`${id}-hint`} className="text-text-muted mt-1 text-xs">
        On another board it is added at the end, outside any group.
      </p>
    </div>
  )
}
