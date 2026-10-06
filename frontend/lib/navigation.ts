import {
  ChartBarIcon,
  CogIcon,
  HomeIcon,
  ServerIcon,
  UserGroupIcon,
} from '@heroicons/react/24/outline'

// The app's main pages, shared by the sidebar and the canvas user menu
export function mainNavigation(isAdmin: boolean) {
  return [
    { name: 'Dashboard', href: '/dashboard', icon: HomeIcon },
    { name: 'Services', href: '/services', icon: ServerIcon },
    { name: 'Metrics', href: '/metrics', icon: ChartBarIcon },
    ...(isAdmin ? [{ name: 'Users', href: '/admin/users', icon: UserGroupIcon }] : []),
    { name: 'Settings', href: '/settings', icon: CogIcon },
  ]
}
