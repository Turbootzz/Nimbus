'use client'

import type { DockerContainersWidgetConfig } from '@/types'
import type { WidgetFormProps } from '@/components/widgets/registry'
import { Toggle } from '@/components/ui/Toggle'

export default function DockerContainersForm({
  config,
  onChange,
  disabled,
}: WidgetFormProps<DockerContainersWidgetConfig>) {
  return (
    <Toggle
      id="docker-hide-stopped"
      enabled={config.hide_stopped}
      onChange={(hide_stopped) => onChange({ ...config, hide_stopped })}
      label="Hide stopped containers"
      description="The count still includes them"
      disabled={disabled}
    />
  )
}
