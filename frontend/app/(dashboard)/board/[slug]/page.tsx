'use client'

import { useEffect } from 'react'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { Squares2X2Icon } from '@heroicons/react/24/outline'
import DashboardView, { DashboardLoading } from '@/components/DashboardView'
import { useBoards } from '@/hooks/useBoards'
import { useKiosk } from '@/hooks/useKiosk'
import { findBoard, rememberBoard } from '@/lib/boards'

// /board/<slug>: one of the user's boards, or one shared with them. The
// slug can also be a board ID (a shared board whose slug the user has too).
export default function BoardPage() {
  const t = useTranslations('dashboard.board')
  const { slug } = useParams<{ slug: string }>()
  const { boards, loaded, failed } = useBoards()
  const kiosk = useKiosk()
  const board = findBoard(boards, decodeURIComponent(slug))

  // /dashboard comes back here; a kiosk's URL already says where it is
  useEffect(() => {
    if (board && !kiosk) rememberBoard(board)
  }, [board, kiosk])

  // Not loaded is not "not found": the page tries again when it is opened again
  if (!loaded && failed) {
    return (
      <div className="glass-card border-card-border mx-auto max-w-md rounded-lg border p-12 text-center">
        <p className="text-text-secondary">{t('loadFailed')}</p>
      </div>
    )
  }
  if (!loaded) return <DashboardLoading />
  if (!board) {
    return (
      <div className="glass-card border-card-border mx-auto max-w-md rounded-lg border p-12 text-center">
        <Squares2X2Icon className="text-text-muted mx-auto mb-4 h-12 w-12" aria-hidden="true" />
        <h1 className="text-text-primary mb-2 text-lg font-medium">{t('notFound')}</h1>
        <p className="text-text-secondary mb-4">{t('notFoundDescription')}</p>
        <Link
          href="/dashboard"
          className="text-primary hover:text-primary-hover text-sm font-medium"
        >
          {t('goToDashboard')}
        </Link>
      </div>
    )
  }
  return <DashboardView key={board.id} board={board} />
}
