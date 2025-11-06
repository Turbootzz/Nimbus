interface MetricsCardProps {
  label: string
  value: string | number
  subtext?: string
  trend?: 'up' | 'down' | 'neutral'
  className?: string
}

export default function MetricsCard({
  label,
  value,
  subtext,
  trend,
  className = '',
}: MetricsCardProps) {
  const getTrendColor = () => {
    if (!trend) return ''
    switch (trend) {
      case 'up':
        return 'text-success'
      case 'down':
        return 'text-error'
      case 'neutral':
        return 'text-text-secondary'
    }
  }

  return (
    <div className={`border-card-border bg-card rounded-lg border p-4 ${className}`}>
      <p className="text-text-secondary mb-1 text-sm">{label}</p>
      <p
        className={`text-xl font-bold break-words sm:text-2xl ${getTrendColor() || 'text-text-primary'}`}
      >
        {value}
      </p>
      {subtext && <p className="text-text-muted mt-1 text-xs break-words">{subtext}</p>}
    </div>
  )
}
