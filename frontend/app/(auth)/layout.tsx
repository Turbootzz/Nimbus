import ThemeToggle from '@/components/ThemeToggle'

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center">
      {/* Theme toggle button - fixed top right */}
      <div className="fixed top-4 right-4">
        <ThemeToggle />
      </div>

      <div className="w-full max-w-md px-6">
        {/* Logo/Branding */}
        <div className="mb-8 text-center">
          <h1 className="mb-2 text-4xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
            ☁️ Nimbus
          </h1>
          <p style={{ color: 'var(--color-text-secondary)' }}>Your Personal Homelab Dashboard</p>
        </div>

        {/* Auth Form Container */}
        {children}
      </div>
    </div>
  )
}
