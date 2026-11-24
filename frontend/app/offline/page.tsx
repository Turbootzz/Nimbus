import type { Metadata } from 'next'
import OfflineServices from '@/components/OfflineServices'

export const metadata: Metadata = {
  title: 'Offline',
  robots: { index: false },
}

// Shown by the service worker when a page cannot be loaded. Public and static:
// it is cached at install and must render without the network.
export default function OfflinePage() {
  return (
    <main className="mx-auto flex min-h-screen w-full max-w-xl flex-col justify-center px-4 py-10">
      <OfflineServices />
    </main>
  )
}
