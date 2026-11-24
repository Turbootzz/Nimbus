import { useTranslations } from 'next-intl'
import { weatherConditionId } from '@/lib/weather'

// describeWeather's label in the user's language
export function useWeatherLabel() {
  const t = useTranslations('weather')
  return (code: number) => t(weatherConditionId(code))
}
