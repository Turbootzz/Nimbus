'use client'

import { forwardRef, InputHTMLAttributes } from 'react'

interface ThemedInputProps extends InputHTMLAttributes<HTMLInputElement> {
  // All standard input props are inherited
}

export const ThemedInput = forwardRef<HTMLInputElement, ThemedInputProps>(
  ({ className = '', ...props }, ref) => {
    return (
      <input
        ref={ref}
        className={`w-full rounded-lg border px-4 py-2 transition focus:ring-2 focus:outline-none ${className}`}
        style={{
          backgroundColor: 'var(--color-background)',
          borderColor: 'var(--color-card-border)',
          color: 'var(--color-text-primary)',
        }}
        onFocus={(e) => {
          e.currentTarget.style.borderColor = 'var(--color-primary)'
        }}
        onBlur={(e) => {
          e.currentTarget.style.borderColor = 'var(--color-card-border)'
        }}
        {...props}
      />
    )
  }
)

ThemedInput.displayName = 'ThemedInput'
