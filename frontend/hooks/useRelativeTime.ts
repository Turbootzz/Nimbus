import { useTranslations } from 'next-intl'
import { timeAgo } from '@/lib/date-utils'

// formatRelativeTime in the user's language: "just now", "2m ago", "3h ago"
export function useRelativeTime() {
  const t = useTranslations('time')
  return (timestamp: string) => {
    const { unit, count } = timeAgo(timestamp)
    return unit === 'now' ? t('justNow') : t(`${unit}Ago`, { count })
  }
}
