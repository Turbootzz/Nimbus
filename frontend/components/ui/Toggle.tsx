interface ToggleProps {
  enabled: boolean
  onChange: (enabled: boolean) => void
  label: string
  description?: string
  disabled?: boolean
  id?: string
}

export function Toggle({
  enabled,
  onChange,
  label,
  description,
  disabled = false,
  id,
}: ToggleProps) {
  const labelId = id ? `${id}-label` : undefined
  const descriptionId = id && description ? `${id}-description` : undefined

  return (
    <div className="flex items-center justify-between gap-4">
      <div className="flex-1">
        <label id={labelId} htmlFor={id} className="text-text-primary text-sm font-medium">
          {label}
        </label>
        {description && (
          <p id={descriptionId} className="text-text-muted mt-0.5 text-xs">
            {description}
          </p>
        )}
      </div>
      <button
        id={id}
        type="button"
        role="switch"
        aria-checked={enabled}
        aria-labelledby={labelId}
        aria-describedby={descriptionId}
        onClick={() => onChange(!enabled)}
        disabled={disabled}
        className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:ring-2 focus:ring-offset-2 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50 ${
          enabled ? 'bg-primary' : 'bg-gray-400'
        }`}
      >
        <span
          className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
            enabled ? 'translate-x-5' : 'translate-x-0'
          }`}
        />
      </button>
    </div>
  )
}
