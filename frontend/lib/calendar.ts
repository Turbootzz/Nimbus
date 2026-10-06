import type { CalendarEvent } from '@/types'

// Tailwind needs whole class names; one color per source, in order
export const sourceColors = [
  'bg-sky-500',
  'bg-violet-500',
  'bg-amber-500',
  'bg-emerald-500',
  'bg-rose-500',
  'bg-teal-500',
  'bg-orange-500',
  'bg-indigo-500',
  'bg-lime-500',
]

// dayKey is YYYY-MM-DD of a date in the browser's zone
export function dayKey(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

// eventDay is the local day an event is on; all-day events keep their date
export function eventDay(event: CalendarEvent): string {
  return event.all_day ? event.start : dayKey(new Date(event.start))
}

// monthWeeks lists the weeks (Monday first) of the month that starts on
// from (YYYY-MM-DD), with null for the days before the 1st and after the end
export function monthWeeks(from: string): (string | null)[][] {
  const [year, month] = from.split('-').map(Number)
  const first = new Date(year, month - 1, 1)
  const daysInMonth = new Date(year, month, 0).getDate()
  const lead = (first.getDay() + 6) % 7 // Monday = 0
  const cells: (string | null)[] = Array(lead).fill(null)
  for (let day = 1; day <= daysInMonth; day++) {
    cells.push(dayKey(new Date(year, month - 1, day)))
  }
  while (cells.length % 7 !== 0) cells.push(null)
  const weeks = []
  for (let i = 0; i < cells.length; i += 7) weeks.push(cells.slice(i, i + 7))
  return weeks
}
