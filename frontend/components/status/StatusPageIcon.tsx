'use client'

import { useState, useSyncExternalStore } from 'react'
import type { PublicStatusService } from '@/types'
import { getApiUrl } from '@/lib/utils/api-url'

const noSubscription = () => () => {}

/**
 * The API URL as the browser sees it. It can differ from the server's, so it
 * is only known after hydration; until then this is null.
 */
function useBrowserApiUrl(): string | null {
  return useSyncExternalStore(noSubscription, getApiUrl, () => null)
}

/** Uploaded image by file name; alt is empty next to a visible name */
export function StatusPageImage({
  file,
  alt,
  className,
  fallback = null,
}: {
  file: string
  alt: string
  className: string
  fallback?: React.ReactNode
}) {
  const apiUrl = useBrowserApiUrl()
  // The file that failed to load, so a new file gets its own try
  const [failedFile, setFailedFile] = useState<string | null>(null)

  if (failedFile === file) return fallback
  if (!apiUrl) return <span className={className} />
  return (
    // eslint-disable-next-line @next/next/no-img-element -- served by the API, not optimisable
    <img
      src={`${apiUrl}/uploads/service-icons/${encodeURIComponent(file)}`}
      alt={alt}
      className={className}
      onError={() => setFailedFile(file)}
    />
  )
}

const iconBox = 'flex h-8 w-8 shrink-0 items-center justify-center'

/**
 * Service icon of a status row: emoji, uploaded image or (with links on) a
 * remote image. Takes the icon fields only: props of a client component end
 * up in the page source.
 */
export default function StatusPageIcon({
  service,
}: {
  service: Pick<PublicStatusService, 'icon' | 'icon_type' | 'icon_image_path'>
}) {
  const [failedUrl, setFailedUrl] = useState<string | null>(null)
  const emoji = (
    <span aria-hidden="true" className={`${iconBox} text-2xl leading-none`}>
      {service.icon}
    </span>
  )

  if (service.icon_type === 'image_upload' && service.icon_image_path) {
    return (
      <StatusPageImage
        file={service.icon_image_path}
        alt=""
        className="h-8 w-8 shrink-0 object-contain"
        fallback={emoji}
      />
    )
  }
  const url = service.icon_image_path
  if (service.icon_type === 'image_url' && url && url !== failedUrl) {
    return (
      // eslint-disable-next-line @next/next/no-img-element -- any remote host
      <img
        src={url}
        alt=""
        referrerPolicy="no-referrer"
        className="h-8 w-8 shrink-0 object-contain"
        onError={() => setFailedUrl(url)}
      />
    )
  }
  return emoji
}
