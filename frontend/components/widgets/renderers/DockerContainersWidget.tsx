'use client'

import type { DockerContainersPayload, DockerContainersWidgetConfig } from '@/types'
import type { WidgetRendererProps } from '@/components/widgets/registry'

function stateColor(state: string): string {
  if (state === 'running') return 'bg-success'
  if (state === 'paused' || state === 'restarting') return 'bg-warning'
  return 'bg-text-muted'
}

export default function DockerContainersWidget({
  cardSize,
  snapshot,
}: WidgetRendererProps<DockerContainersWidgetConfig>) {
  const docker = snapshot?.payload as DockerContainersPayload | null | undefined
  if (!docker) return null // WidgetCard shows loading and errors

  // Narrow tiles show names only; the status is in the tooltip
  const wide = cardSize === '2x1' || cardSize === '2x2'
  return (
    <div>
      <p className="text-text-muted mb-1 text-xs">
        {docker.running} of {docker.total} running
      </p>
      {docker.containers.length === 0 ? (
        <p className="text-text-muted text-sm">No containers</p>
      ) : (
        <ul>
          {docker.containers.map((container) => (
            <li
              key={container.id}
              className="flex items-center gap-2 py-1 text-sm"
              title={`${container.image} · ${container.status}`}
            >
              <span
                className={`h-2 w-2 shrink-0 rounded-full ${stateColor(container.state)}`}
                aria-hidden="true"
              />
              <span className="text-text-primary min-w-0 flex-1 truncate">
                {container.name || container.image}
                <span className="sr-only">, {container.state}</span>
              </span>
              {wide && (
                <span className="text-text-muted max-w-[50%] shrink-0 truncate text-xs">
                  {container.status}
                </span>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
