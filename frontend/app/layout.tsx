import type { Metadata } from 'next'
import { Geist, Geist_Mono } from 'next/font/google'
import '../styles/globals.css'
import { ThemeProvider } from '@/contexts/ThemeContext'
import { ThemeScript } from '@/components/ThemeScript'

const geistSans = Geist({
  variable: '--font-geist-sans',
  subsets: ['latin'],
})

const geistMono = Geist_Mono({
  variable: '--font-geist-mono',
  subsets: ['latin'],
})

// SEO: Set NEXT_PUBLIC_SITE_URL to enable SEO features (e.g., https://nimbus.example.com)
const siteUrl = process.env.NEXT_PUBLIC_SITE_URL
const SEO_TITLE = 'Nimbus - Modern Self-Hosted Homelab Dashboard'
const SEO_DESCRIPTION =
  'A beautiful, open-source alternative to Homarr and Dashy. Organize your homelab services with a drag-and-drop dashboard. Docker ready.'

export const metadata: Metadata = {
  title: 'Nimbus',
  icons: {
    icon: [
      { url: '/favicon/favicon-16x16.png', sizes: '16x16', type: 'image/png' },
      { url: '/favicon/favicon-32x32.png', sizes: '32x32', type: 'image/png' },
      { url: '/favicon/favicon.ico', sizes: 'any' },
    ],
    apple: '/favicon/apple-touch-icon.png',
  },
  manifest: '/favicon/site.webmanifest',
  // SEO metadata only when NEXT_PUBLIC_SITE_URL is configured
  ...(siteUrl && {
    title: SEO_TITLE,
    description: SEO_DESCRIPTION,
    keywords: [
      'homelab',
      'dashboard',
      'self-hosted',
      'docker',
      'startpage',
      'nimbus',
      'open source',
      'server monitoring',
    ],
    authors: [{ name: 'Nimbus' }],
    metadataBase: new URL(siteUrl),
    openGraph: {
      type: 'website',
      locale: 'en_US',
      url: siteUrl,
      siteName: 'Nimbus',
      title: SEO_TITLE,
      description: SEO_DESCRIPTION,
      images: [
        {
          url: '/og-image.png',
          width: 1200,
          height: 630,
          alt: 'Nimbus - Homelab Dashboard',
        },
      ],
    },
    twitter: {
      card: 'summary_large_image',
      title: SEO_TITLE,
      description: SEO_DESCRIPTION,
      images: ['/og-image.png'],
    },
  }),
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  // Only include JSON-LD when site URL is configured (for public instances)
  const jsonLd = siteUrl
    ? {
        '@context': 'https://schema.org',
        '@type': 'SoftwareApplication',
        name: 'Nimbus',
        description: SEO_DESCRIPTION,
        operatingSystem: 'Docker, Linux',
        applicationCategory: 'UtilitiesApplication',
        offers: {
          '@type': 'Offer',
          price: '0',
          priceCurrency: 'USD',
        },
        url: siteUrl,
      }
    : null

  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <ThemeScript />
        {jsonLd && (
          <script
            type="application/ld+json"
            dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
          />
        )}
      </head>
      <body className={`${geistSans.variable} ${geistMono.variable} antialiased`}>
        <ThemeProvider>{children}</ThemeProvider>
      </body>
    </html>
  )
}
