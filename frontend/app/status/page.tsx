import type { Metadata } from 'next'
import { cache } from 'react'
import { headers } from 'next/headers'
import { notFound } from 'next/navigation'
import type { PublicStatusPage, PublicStatusService, StatusPageOverall } from '@/types'
import { fetchPublicStatus, overallLabels, statusLabels } from '@/lib/status-page'
import { getStatusBgColor } from '@/lib/status-utils'
import UptimeBar from '@/components/status/UptimeBar'
import StatusPageIcon, { StatusPageImage } from '@/components/status/StatusPageIcon'
import StatusRefresh from '@/components/status/StatusRefresh'

// One fetch per request, shared by the metadata and the page. Reading the
// headers also keeps this page out of the build-time prerender.
const loadStatus = cache(async () => fetchPublicStatus(await headers()))

export async function generateMetadata(): Promise<Metadata> {
  const result = await loadStatus()
  if (result.state === 'disabled') notFound()
  return {
    title: result.state === 'ok' ? result.page.title : 'Status',
    robots: { index: false, follow: false },
  }
}

const overallTone: Record<StatusPageOverall, string> = {
  operational: 'bg-success',
  degraded: 'bg-warning',
  outage: 'bg-error',
  unknown: 'bg-text-muted',
}

function OverallBanner({ page }: { page: PublicStatusPage }) {
  const online = page.services.filter((service) => service.status === 'online').length
  return (
    <section
      aria-label="Overall status"
      className="glass-card border-card-border flex items-center gap-4 rounded-xl border p-5"
    >
      <span aria-hidden="true" className="relative flex h-4 w-4 shrink-0">
        {page.overall === 'operational' && (
          <span className="bg-success absolute inline-flex h-full w-full animate-ping rounded-full opacity-40 motion-reduce:hidden" />
        )}
        <span
          className={`relative inline-flex h-4 w-4 rounded-full ${overallTone[page.overall]}`}
        />
      </span>
      <div className="min-w-0">
        <p className="text-text-primary text-lg font-semibold sm:text-xl">
          {overallLabels[page.overall]}
        </p>
        {page.services.length > 0 && (
          <p className="text-text-secondary text-sm">
            {online} of {page.services.length} online
          </p>
        )}
      </div>
    </section>
  )
}

function ServiceRow({ service }: { service: PublicStatusService }) {
  return (
    <li className="p-4 sm:p-5">
      <div className="flex items-center gap-3">
        <StatusPageIcon service={service} />
        <h2 className="text-text-primary min-w-0 flex-1 truncate font-medium">
          {service.url ? (
            <a
              href={service.url}
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-primary focus-visible:ring-primary rounded-sm hover:underline focus-visible:ring-2 focus-visible:outline-none"
            >
              {service.name}
            </a>
          ) : (
            service.name
          )}
        </h2>
        <p className="text-text-secondary flex shrink-0 items-center gap-1.5 text-sm font-medium">
          <span
            aria-hidden="true"
            className={`h-2.5 w-2.5 rounded-full ${getStatusBgColor(service.status)}`}
          />
          {statusLabels[service.status] ?? statusLabels.unknown}
        </p>
      </div>
      <div className="mt-3 grid gap-3 sm:grid-cols-2 sm:gap-6">
        <UptimeBar
          label="24 hours"
          bars={service.bars_24h}
          hoursPerBar={1}
          uptime={service.uptime_24h}
        />
        <UptimeBar
          label="7 days"
          bars={service.bars_7d}
          hoursPerBar={6}
          uptime={service.uptime_7d}
        />
      </div>
    </li>
  )
}

export default async function StatusPage() {
  const result = await loadStatus()
  if (result.state === 'disabled') notFound()
  const page = result.state === 'ok' ? result.page : null

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-6 px-4 py-8 sm:py-12">
      <header className="flex items-center gap-3">
        {page?.logo && (
          <StatusPageImage file={page.logo} alt="" className="h-10 w-10 shrink-0 object-contain" />
        )}
        <h1 className="text-text-primary min-w-0 text-2xl font-bold break-words sm:text-3xl">
          {page?.title ?? 'Status'}
        </h1>
      </header>

      {page ? (
        <>
          <OverallBanner page={page} />
          {page.services.length > 0 ? (
            <ul className="glass-card border-card-border divide-card-border divide-y rounded-xl border">
              {page.services.map((service, index) => (
                <ServiceRow key={index} service={service} />
              ))}
            </ul>
          ) : (
            <p className="text-text-secondary text-sm">No services are being monitored yet.</p>
          )}
        </>
      ) : (
        <p role="status" className="glass-card border-card-border rounded-xl border p-5">
          <span className="text-text-primary font-semibold">
            Status is temporarily unavailable.
          </span>{' '}
          <span className="text-text-secondary">This page retries by itself.</span>
        </p>
      )}

      <footer className="text-text-muted mt-auto pt-4 text-xs">
        <StatusRefresh updatedAt={page?.updated_at} />
      </footer>
    </main>
  )
}
