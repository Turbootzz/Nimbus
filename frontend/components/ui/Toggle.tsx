interface ToggleProps {
  enabled: boolean
  onChange: (enabled: boolean) => void
  label: string
  description?: string
}

export function Toggle({ enabled, onChange, label, description }: ToggleProps) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex-1">
        <p className="text-text-primary text-sm font-medium">{label}</p>
        {description && <p className="text-text-muted mt-1 text-xs">{description}</p>}
      </div>
      <button
        onClick={() => onChange(!enabled)}
        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
          enabled ? 'bg-primary' : 'bg-card-border'
        }`}
        aria-label={label}
      >
        <span
          className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
            enabled ? 'translate-x-6' : 'translate-x-1'
          }`}
        />
      </button>
    </div>
  )
}
