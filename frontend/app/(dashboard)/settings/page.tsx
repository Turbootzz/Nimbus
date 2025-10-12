import Link from 'next/link'

export default function SettingsPage() {
  const settingsSections = [
    {
      title: 'Theme',
      description: 'Customize colors, dark mode, and background',
      href: '/settings/theme',
      icon: (
        <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01"
          />
        </svg>
      ),
    },
    {
      title: 'Profile',
      description: 'Manage your account and personal information',
      href: '/settings/profile',
      icon: (
        <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
          />
        </svg>
      ),
    },
  ]

  return (
    <div className="max-w-4xl p-6">
      <h1 className="mb-2 text-3xl font-bold">Settings</h1>
      <p className="text-base-content/70 mb-8">Configure your dashboard preferences</p>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {settingsSections.map((section) => (
          <Link
            key={section.href}
            href={section.href}
            className="bg-card border-card-border hover:border-primary/50 block rounded-lg border p-6 transition-all hover:shadow-lg"
          >
            <div className="flex items-start gap-4">
              <div className="text-primary flex-shrink-0">{section.icon}</div>
              <div className="min-w-0 flex-1">
                <h2 className="text-text-primary mb-1 text-lg font-semibold">{section.title}</h2>
                <p className="text-text-secondary text-sm">{section.description}</p>
              </div>
              <svg
                className="text-text-muted h-5 w-5 flex-shrink-0"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 5l7 7-7 7"
                />
              </svg>
            </div>
          </Link>
        ))}
      </div>
    </div>
  )
}
