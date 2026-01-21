'use client'

import { forwardRef, InputHTMLAttributes } from 'react'

type ThemedInputProps = InputHTMLAttributes<HTMLInputElement>

export const ThemedInput = forwardRef<HTMLInputElement, ThemedInputProps>(
  ({ className = '', onFocus, onBlur, ...rest }, ref) => {
    const handleFocus = (e: React.FocusEvent<HTMLInputElement>) => {
      e.currentTarget.style.borderColor = 'var(--color-primary)'
      onFocus?.(e)
    }

    const handleBlur = (e: React.FocusEvent<HTMLInputElement>) => {
      e.currentTarget.style.borderColor = 'var(--color-card-border)'
      onBlur?.(e)
    }

    return (
      <input
        ref={ref}
        className={`w-full rounded-lg border px-4 py-2 transition focus:ring-2 focus:outline-none ${className}`}
        style={{
          backgroundColor: 'var(--color-background)',
          borderColor: 'var(--color-card-border)',
          color: 'var(--color-text-primary)',
        }}
        onFocus={handleFocus}
        onBlur={handleBlur}
        {...rest}
      />
    )
  }
)

ThemedInput.displayName = 'ThemedInput'
