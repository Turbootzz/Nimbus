'use client'

import type { CardSize } from '@/types'

interface CardSizeSelectorProps {
  value: CardSize
  onChange: (size: CardSize) => void
}

const sizes: { value: CardSize; label: string; description: string }[] = [
  { value: '1x1', label: 'Compact', description: 'Half width - icon & name only' },
  { value: '2x1', label: 'Standard', description: 'Default - icon, status, description' },
  { value: '1x2', label: 'Tall', description: 'Half width, double height' },
  { value: '2x2', label: 'Large', description: 'Standard width, double height' },
]

export default function CardSizeSelector({ value, onChange }: CardSizeSelectorProps) {
  return (
    <div>
      <label className="text-text-secondary mb-2 block text-sm font-medium">Card Size</label>
      <div className="grid grid-cols-4 gap-2">
        {sizes.map((size) => (
          <button
            key={size.value}
            type="button"
            onClick={() => onChange(size.value)}
            className={`flex flex-col items-center rounded-md p-3 transition-all ${
              value === size.value
                ? 'bg-primary text-white shadow-md'
                : 'border-card-border text-text-primary hover:border-primary border-2'
            }`}
            style={
              value !== size.value ? { backgroundColor: 'var(--color-background)' } : undefined
            }
          >
            {/* Visual preview box */}
            <SizePreview size={size.value} selected={value === size.value} />
            <span className="mt-2 text-xs font-medium">{size.label}</span>
          </button>
        ))}
      </div>
      <p className="text-text-muted mt-2 text-xs">
        {sizes.find((s) => s.value === value)?.description}
      </p>
    </div>
  )
}

function SizePreview({ size, selected }: { size: CardSize; selected: boolean }) {
  const borderClass = selected ? 'border-white' : 'border-card-border'

  // Previews show relative proportions: width ratio 1:2:1:2, height ratio 1:1:2:2
  switch (size) {
    case '1x1':
      return <div className={`h-6 w-6 rounded border-2 ${borderClass}`} />
    case '2x1':
      return <div className={`h-6 w-12 rounded border-2 ${borderClass}`} />
    case '1x2':
      return <div className={`h-12 w-6 rounded border-2 ${borderClass}`} />
    case '2x2':
      return <div className={`h-12 w-12 rounded border-2 ${borderClass}`} />
  }
}
