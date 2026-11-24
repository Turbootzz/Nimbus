import type { NextConfig } from 'next'

// The service worker must be revalidated on every load, or a deploy would not
// reach installed apps
const workerHeaders = [
  { key: 'Content-Type', value: 'application/javascript; charset=utf-8' },
  { key: 'Cache-Control', value: 'no-cache, no-store, must-revalidate' },
]

const nextConfig: NextConfig = {
  env: {
    // Names the service worker's static cache, so each build gets a fresh one
    NEXT_PUBLIC_BUILD_ID: Date.now().toString(36),
  },
  async headers() {
    return [
      { source: '/sw.js', headers: workerHeaders },
      { source: '/sw-routing.js', headers: workerHeaders },
    ]
  },
  // Don't let `next dev` generate AGENTS.md / CLAUDE.md in the repo
  agentRules: false,
  images: {
    // Allow external images from any domain for user-provided icon URLs
    remotePatterns: [
      {
        protocol: 'https',
        hostname: '**', // Wildcard to allow any HTTPS domain
      },
      {
        protocol: 'http',
        hostname: '**', // Wildcard to allow any HTTP domain (for local development)
      },
    ],
    // Reasonable size limits for icon images
    deviceSizes: [64, 96, 128, 256],
    imageSizes: [16, 32, 48, 64, 96, 128],
  },
}

export default nextConfig
