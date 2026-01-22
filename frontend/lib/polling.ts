import type { Service } from '@/types'

/**
 * Health data from polled services
 */
export interface HealthData {
  status: Service['status']
  response_time?: number
}

/**
 * Builds a health map from polled services for efficient lookup
 */
export function buildHealthMap(services: Service[]): Map<string, HealthData> {
  return new Map(services.map((s) => [s.id, { status: s.status, response_time: s.response_time }]))
}

/**
 * Checks if health data has changed between existing and polled service
 */
export function hasHealthChanged(existing: Service, health: HealthData): boolean {
  return existing.status !== health.status || existing.response_time !== health.response_time
}

/**
 * Merges health data into an existing service, preserving all other fields
 */
export function mergeHealthData(existing: Service, health: HealthData): Service {
  return {
    ...existing,
    status: health.status,
    response_time: health.response_time,
  }
}

/**
 * Merges polled health data into existing services.
 * Only updates status and response_time to avoid disrupting UI state.
 * Services not found in the health map are preserved unchanged.
 */
export function mergeServicesHealth(
  existingServices: Service[],
  polledServices: Service[]
): Service[] {
  const healthMap = buildHealthMap(polledServices)

  return existingServices.map((service) => {
    const health = healthMap.get(service.id)
    if (health && hasHealthChanged(service, health)) {
      return mergeHealthData(service, health)
    }
    return service
  })
}

/**
 * Determines if a poll should be skipped based on recent fetch activity
 */
export function shouldSkipPoll(lastPollTime: number, overlapWindow: number = 5000): boolean {
  return Date.now() - lastPollTime < overlapWindow
}

/**
 * Checks if response time should be displayed for a service
 */
export function shouldShowResponseTime(service: Service): boolean {
  return (
    service.status === 'online' &&
    service.response_time !== undefined &&
    service.response_time !== null
  )
}

/**
 * Calculates average response time from services (only online services with response times)
 */
export function calculateAverageResponseTime(services: Service[]): number {
  const responseTimes = services
    .filter(shouldShowResponseTime)
    .map((s) => s.response_time as number)

  if (responseTimes.length === 0) return 0

  return Math.round(responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length)
}
