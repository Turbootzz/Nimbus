import { MetadataRoute } from 'next'

export default function robots(): MetadataRoute.Robots {
  const siteUrl = process.env.NEXT_PUBLIC_SITE_URL

  // Disable indexing when SEO is not configured
  if (!siteUrl) {
    return {
      rules: { userAgent: '*', disallow: '/' },
    }
  }

  return {
    rules: [
      {
        userAgent: '*',
        allow: ['/', '/images/', '/favicon/'],
        disallow: ['/dashboard/', '/api/', '/settings/', '/admin/'],
      },
    ],
    sitemap: `${siteUrl}/sitemap.xml`,
  }
}
