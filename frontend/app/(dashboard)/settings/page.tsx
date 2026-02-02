'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { api } from '@/lib/api'
import type { User } from '@/types'

export default function SettingsPage() {
  const [currentUser, setCurrentUser] = useState<User | null>(null)

  useEffect(() => {
    const loadUser = async () => {
      const response = await api.getCurrentUser()
      if (response.data) {
        setCurrentUser(response.data)
      }
    }
    loadUser()
  }, [])

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
    {
      title: 'Notifications',
      description: 'Configure webhooks for service status alerts',
      href: '/settings/notifications',
      icon: (
        <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
          />
        </svg>
      ),
    },
  ]

  const adminSection = {
    title: 'Admin Settings',
    description: 'Configure system-wide settings for Nimbus',
    href: '/settings/admin',
    icon: (
      <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
        />
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
        />
      </svg>
    ),
  }

  return (
    <div className="max-w-4xl p-4 sm:p-6">
      <h1 className="mb-2 text-2xl font-bold sm:text-3xl">Settings</h1>
      <p className="text-base-content/70 mb-8 text-sm sm:text-base">
        Configure your dashboard preferences
      </p>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {settingsSections.map((section) => (
          <Link
            key={section.href}
            href={section.href}
            className="bg-card border-card-border hover:border-primary/50 block h-full rounded-lg border p-6 transition-all hover:shadow-lg"
          >
            <div className="flex items-start gap-4">
              <div className="text-primary shrink-0">{section.icon}</div>
              <div className="min-w-0 flex-1">
                <h2 className="text-text-primary mb-1 text-lg font-semibold">{section.title}</h2>
                <p className="text-text-secondary min-h-10 text-sm">{section.description}</p>
              </div>
              <svg
                className="text-text-muted h-5 w-5 shrink-0"
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

        {currentUser?.role === 'admin' && (
          <Link
            href={adminSection.href}
            className="bg-card border-card-border hover:border-primary/50 block h-full rounded-lg border p-6 transition-all hover:shadow-lg"
          >
            <div className="flex items-start gap-4">
              <div className="text-primary shrink-0">{adminSection.icon}</div>
              <div className="min-w-0 flex-1">
                <h2 className="text-text-primary mb-1 text-lg font-semibold">
                  {adminSection.title}
                </h2>
                <p className="text-text-secondary min-h-10 text-sm">{adminSection.description}</p>
              </div>
              <svg
                className="text-text-muted h-5 w-5 shrink-0"
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
        )}
      </div>
    </div>
  )
}
