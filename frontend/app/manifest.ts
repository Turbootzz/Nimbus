import type { MetadataRoute } from 'next'

// Served at /manifest.webmanifest and linked from every page by Next.js
export default function manifest(): MetadataRoute.Manifest {
  // || and not ??: the Docker build sets an empty string when the arg is unset
  const name = process.env.NEXT_PUBLIC_APP_NAME || 'Nimbus'

  return {
    name,
    short_name: name,
    description: 'Your personal homelab dashboard',
    id: '/',
    start_url: '/dashboard',
    scope: '/',
    display: 'standalone',
    theme_color: '#0ea5e9',
    background_color: '#ffffff',
    icons: [
      { src: '/favicon/android-chrome-192x192.png', sizes: '192x192', type: 'image/png' },
      { src: '/favicon/android-chrome-512x512.png', sizes: '512x512', type: 'image/png' },
      {
        src: '/favicon/maskable-512x512.png',
        sizes: '512x512',
        type: 'image/png',
        purpose: 'maskable',
      },
    ],
  }
}
