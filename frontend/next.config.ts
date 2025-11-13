import type { NextConfig } from 'next'

const nextConfig: NextConfig = {
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
