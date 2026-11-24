import type { Messages } from 'next-intl'
import {
  ChartBarIcon,
  CogIcon,
  HomeIcon,
  ServerIcon,
  UserGroupIcon,
} from '@heroicons/react/24/outline'

interface NavItem {
  id: keyof Messages['layout']['nav'] // its name: layout.nav.<id>
  href: string
  icon: typeof HomeIcon
}

// The app's main pages, shared by the sidebar and the canvas user menu
export function mainNavigation(isAdmin: boolean): NavItem[] {
  return [
    { id: 'dashboard', href: '/dashboard', icon: HomeIcon },
    { id: 'services', href: '/services', icon: ServerIcon },
    { id: 'metrics', href: '/metrics', icon: ChartBarIcon },
    ...(isAdmin ? [{ id: 'users' as const, href: '/admin/users', icon: UserGroupIcon }] : []),
    { id: 'settings', href: '/settings', icon: CogIcon },
  ]
}
