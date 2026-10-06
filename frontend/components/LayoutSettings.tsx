'use client'

import { useTheme } from '@/contexts/ThemeContext'
import { useLayoutMode } from '@/lib/layout-store'
import { defaultGlass } from '@/lib/wallpaper'
import type { LayoutMode } from '@/types'

// Glass the canvas starts with when the cards are still plain
const canvasGlass = { cardOpacity: 75, cardBlur: 12 }

// Small sketches of both layouts
function Preview({ mode }: { mode: LayoutMode }) {
  const tile = 'bg-text-muted/30 h-4 rounded-sm'
  return (
    <div
      aria-hidden="true"
      className="bg-background border-card-border flex h-20 overflow-hidden rounded border"
    >
      {mode === 'classic' && <div className="bg-text-muted/20 w-5 shrink-0" />}
      <div className="flex flex-1 flex-col gap-1 p-1.5">
        <div className={`bg-text-muted/20 h-2.5 rounded-sm ${mode === 'canvas' ? '' : 'w-1/2'}`} />
        {mode === 'classic' && (
          <div className="grid grid-cols-4 gap-1">
            {[0, 1, 2, 3].map((i) => (
              <div key={i} className="bg-text-muted/20 h-2 rounded-sm" />
            ))}
          </div>
        )}
        <div className="grid grid-cols-3 gap-1">
          {[0, 1, 2, 3, 4, 5].map((i) => (
            <div key={i} className={tile} />
          ))}
        </div>
      </div>
    </div>
  )
}

const options: { mode: LayoutMode; label: string; description: string }[] = [
  { mode: 'classic', label: 'Classic', description: 'Sidebar, stats and your tiles' },
  { mode: 'canvas', label: 'Canvas', description: 'The dashboard fills the page, with a top bar' },
]

export default function LayoutSettings() {
  const { effectiveTheme, setTheme, glass, setGlass, setLayoutMode } = useTheme()
  const layoutMode = useLayoutMode()

  const choose = (mode: LayoutMode) => {
    if (mode === layoutMode) return
    setLayoutMode(mode)
    const plainCards =
      glass.cardOpacity === defaultGlass.cardOpacity && glass.cardBlur === defaultGlass.cardBlur
    if (mode === 'canvas' && plainCards) setGlass(canvasGlass)
  }

  return (
    <div className="glass-card border-card-border rounded-lg border p-6">
      <h2 className="text-text-primary mb-2 text-xl font-semibold">Dashboard Layout</h2>
      <p className="text-text-secondary mb-4 text-sm">
        Canvas only changes the dashboard; other pages keep the sidebar.
      </p>
      <div className="grid grid-cols-2 gap-3">
        {options.map((option) => (
          <button
            key={option.mode}
            type="button"
            aria-pressed={layoutMode === option.mode}
            onClick={() => choose(option.mode)}
            className={`rounded-lg border-2 p-2 text-left transition-colors ${
              layoutMode === option.mode
                ? 'border-primary'
                : 'border-card-border hover:border-text-muted'
            }`}
          >
            <Preview mode={option.mode} />
            <span className="text-text-primary mt-2 block text-sm font-medium">{option.label}</span>
            <span className="text-text-muted block text-xs">{option.description}</span>
          </button>
        ))}
      </div>
      {layoutMode === 'canvas' && effectiveTheme === 'light' && (
        <p className="text-text-secondary mt-3 text-sm">
          Canvas looks best in dark mode, with a wallpaper.{' '}
          <button
            type="button"
            onClick={() => setTheme('dark')}
            className="text-primary font-medium underline"
          >
            Use dark mode
          </button>
        </p>
      )}
    </div>
  )
}
