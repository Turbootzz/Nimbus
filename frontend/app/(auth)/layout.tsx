import ThemeToggle from '@/components/ThemeToggle'
import NimbusLogo from '@/components/NimbusLogo'
import { getTranslations } from 'next-intl/server'

export default async function AuthLayout({ children }: { children: React.ReactNode }) {
  const t = await getTranslations('auth.layout')
  return (
    <div className="flex min-h-screen items-center justify-center">
      {/* Theme toggle button - fixed top right */}
      <div className="fixed top-4 right-4">
        <ThemeToggle />
      </div>

      <div className="w-full max-w-md px-6">
        {/* Logo/Branding */}
        <div className="wallpaper-panel mx-auto mb-8 flex flex-col items-center text-center">
          <div className="mb-2 flex items-center gap-3">
            <NimbusLogo size={48} priority />
            <h1 className="text-4xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
              Nimbus
            </h1>
          </div>
          <p style={{ color: 'var(--color-text-secondary)' }}>{t('tagline')}</p>
        </div>

        {/* Auth Form Container */}
        {children}
      </div>
    </div>
  )
}
