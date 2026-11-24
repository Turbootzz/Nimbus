import type { Metadata, Viewport } from 'next'
import { Geist, Geist_Mono } from 'next/font/google'
import localFont from 'next/font/local'
import { NextIntlClientProvider } from 'next-intl'
import { getLocale } from 'next-intl/server'
import '../styles/globals.css'
import '../styles/themes.css'
import { ThemeProvider } from '@/contexts/ThemeContext'
import { ThemeScript } from '@/components/ThemeScript'
import LocaleSync from '@/components/LocaleSync'
import ServiceWorkerRegistration from '@/components/ServiceWorkerRegistration'

const geistSans = Geist({
  variable: '--font-geist-sans',
  subsets: ['latin'],
})

const geistMono = Geist_Mono({
  variable: '--font-geist-mono',
  subsets: ['latin'],
})

// The fonts a user can pick in the theme settings (styles/themes.css). The
// files are in the repo under their OFL licences, so the build fetches
// nothing; not preloaded, so only users who picked one download it.
const inter = localFont({
  src: '../styles/fonts/inter/InterVariable.woff2',
  variable: '--font-inter',
  weight: '100 900',
  display: 'swap',
  preload: false,
})

const jetbrainsMono = localFont({
  src: '../styles/fonts/jetbrains-mono/JetBrainsMono-Variable.woff2',
  variable: '--font-jetbrains-mono',
  weight: '100 800',
  display: 'swap',
  preload: false,
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

// Colours the browser and installed-app chrome; matches the manifest
export const viewport: Viewport = {
  themeColor: '#0ea5e9',
}

export default async function RootLayout({
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
  const locale = await getLocale()

  return (
    <html lang={locale} suppressHydrationWarning>
      <head>
        <ThemeScript />
        {jsonLd && (
          <script
            type="application/ld+json"
            dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
          />
        )}
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} ${inter.variable} ${jetbrainsMono.variable} antialiased`}
      >
        <NextIntlClientProvider>
          <ThemeProvider>
            {children}
            <LocaleSync />
          </ThemeProvider>
        </NextIntlClientProvider>
        <ServiceWorkerRegistration />
      </body>
    </html>
  )
}
