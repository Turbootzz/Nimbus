'use client'

import { Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeftIcon } from '@heroicons/react/24/outline'
import ServiceForm from '@/components/ServiceForm'
import { useBoards } from '@/hooks/useBoards'
import { boardHref } from '@/lib/boards'

// Whitelist of valid `from` values to prevent open-redirect via crafted URLs.
// href and label live together so adding a target can't leave the back-link
// pointing one place while the label says another.
const RETURN_TARGETS: Record<string, { href: string; label: string }> = {
  dashboard: { href: '/dashboard', label: 'Dashboard' },
}
const DEFAULT_RETURN = { href: '/services', label: 'Services' }

function NewServiceContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { boards } = useBoards()
  // A board ID only counts when it is one of the user's own boards; the link
  // back is made from that board, never taken from the URL
  const board = boards.find((b) => b.id === searchParams.get('board') && b.can_edit)
  const fromTarget = RETURN_TARGETS[searchParams.get('from') ?? '']
  const returnTarget =
    fromTarget && board
      ? { ...fromTarget, href: boardHref(board, boards) }
      : (fromTarget ?? DEFAULT_RETURN)

  return (
    <div className="mx-auto max-w-2xl">
      {/* Back button */}
      <Link
        href={returnTarget.href}
        className="text-text-secondary hover:text-text-primary wallpaper-panel mb-6 inline-flex items-center text-sm transition-colors"
      >
        <ArrowLeftIcon className="mr-2 h-4 w-4" />
        Back to {returnTarget.label}
      </Link>

      {/* Page header */}
      <div className="wallpaper-panel mb-6">
        <h1 className="text-text-primary text-3xl font-bold">Add New Service</h1>
        <p className="text-text-secondary mt-1">
          Add a service to monitor in your homelab dashboard
        </p>
      </div>

      <ServiceForm
        initialGroupId={searchParams.get('group')}
        boardId={board?.id}
        cancelHref={returnTarget.href}
        onSaved={() => router.push(returnTarget.href)}
      />
    </div>
  )
}

export default function NewServicePage() {
  return (
    <Suspense fallback={<div className="text-text-secondary wallpaper-panel">Loading...</div>}>
      <NewServiceContent />
    </Suspense>
  )
}
