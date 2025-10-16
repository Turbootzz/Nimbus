import ThemeToggle from '@/components/ThemeToggle'
import Image from 'next/image'

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center">
      {/* Theme toggle button - fixed top right */}
      <div className="fixed top-4 right-4">
        <ThemeToggle />
      </div>

      <div className="w-full max-w-md px-6">
        {/* Logo/Branding */}
        <div className="mb-8 flex flex-col items-center text-center">
          <div className="mb-2 flex items-center gap-3">
            <Image src="/images/logo.png" alt="Nimbus Logo" width={48} height={48} priority />
            <h1 className="text-4xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
              Nimbus
            </h1>
          </div>
          <p style={{ color: 'var(--color-text-secondary)' }}>Your Personal Homelab Dashboard</p>
        </div>

        {/* Auth Form Container */}
        {children}
      </div>
    </div>
  )
}
