'use client'

import { useTranslations } from 'next-intl'
import type { DockerContainersWidgetConfig } from '@/types'
import type { WidgetFormProps } from '@/components/widgets/registry'
import { Toggle } from '@/components/ui/Toggle'

export default function DockerContainersForm({
  config,
  onChange,
  disabled,
}: WidgetFormProps<DockerContainersWidgetConfig>) {
  const t = useTranslations('widgets.dockerContainers.form')
  return (
    <Toggle
      id="docker-hide-stopped"
      enabled={config.hide_stopped}
      onChange={(hide_stopped) => onChange({ ...config, hide_stopped })}
      label={t('hideStopped')}
      description={t('hideStoppedHint')}
      disabled={disabled}
    />
  )
}
