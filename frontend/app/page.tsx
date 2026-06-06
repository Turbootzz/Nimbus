import Link from 'next/link'
import Image from 'next/image'
import {
  ShieldCheckIcon,
  ChartBarIcon,
  SwatchIcon,
  CpuChipIcon,
  ArrowRightIcon,
} from '@heroicons/react/24/outline'

const NimbusLogo = ({ size = 32 }: { size?: number }) => (
  <Image
    src="/images/logo.png"
    alt="Nimbus"
    width={272}
    height={202}
    style={{ width: size, height: 'auto' }}
  />
)

const features = [
  {
    name: 'Authentication & Security',
    description:
      'Local accounts with JWT, OAuth2 support (Google, GitHub, Discord), role-based access control, and admin panel for user management.',
    icon: ShieldCheckIcon,
  },
  {
    name: 'Service Monitoring',
    description:
      'Real-time health checks, response time tracking, smart self-signed cert handling, status history and uptime graphs.',
    icon: ChartBarIcon,
  },
  {
    name: 'Personalization',
    description:
      'Custom backgrounds per user, light/dark mode toggle, accent color themes, and drag & drop service tiles.',
    icon: SwatchIcon,
  },
  {
    name: 'Metrics & Integration',
    description:
      'Configurable check intervals, Prometheus metrics export, and mobile responsive design.',
    icon: CpuChipIcon,
  },
]

const screenshots = [
  { src: '/images/dashboard-preview.png', alt: 'Nimbus Dashboard', label: 'Dashboard' },
  { src: '/images/services.png', alt: 'Service Management', label: 'Services' },
  { src: '/images/metrics.png', alt: 'Metrics View', label: 'Metrics' },
  { src: '/images/theme.png', alt: 'Theme Settings', label: 'Themes' },
]

export default function LandingPage() {
  const isNimbusCloud = process.env.NEXT_PUBLIC_NIMBUS_CLOUD === 'true'

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Navigation */}
      <nav className="border-b border-gray-200 bg-white">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
          <div className="flex items-center gap-3">
            <NimbusLogo size={32} />
            <span className="text-xl font-semibold text-gray-900">Nimbus</span>
          </div>
          <div className="flex items-center gap-4">
            <a
              href="https://nimbusapp.dev"
              target="_blank"
              rel="noopener noreferrer"
              className="text-sm font-medium text-sky-500 hover:text-sky-600"
            >
              Nimbus Cloud
            </a>
            <a
              href="https://github.com/Turbootzz/Nimbus"
              target="_blank"
              rel="noopener noreferrer"
              className="text-sm text-gray-600 hover:text-gray-900"
            >
              GitHub
            </a>
            <Link
              href="/login"
              className="rounded-lg bg-sky-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-sky-600"
            >
              Launch demo
            </Link>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="relative overflow-hidden bg-white py-20 sm:py-32">
        <div className="absolute inset-0 bg-linear-to-br from-sky-50 via-white to-white" />
        <div className="relative mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <h1 className="text-4xl font-bold tracking-tight text-gray-900 sm:text-5xl lg:text-6xl">
              Your Homelab,
              <br />
              <span className="text-sky-500">Beautifully Organized</span>
            </h1>
            <p className="mx-auto mt-6 max-w-2xl text-lg text-gray-600">
              The open-source homelab dashboard, hosted for you. No Docker, no maintenance — just
              organize your services.
            </p>
            <div className="mt-10 flex flex-col items-center justify-center gap-4">
              <div className="flex flex-col items-center justify-center gap-4 sm:flex-row">
                <a
                  href="https://nimbusapp.dev"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="group flex items-center gap-2 rounded-lg bg-sky-500 px-6 py-3 text-base font-medium text-white shadow-lg transition-all hover:bg-sky-600 hover:shadow-xl"
                >
                  Try Nimbus Cloud
                  <ArrowRightIcon className="h-4 w-4 transition-transform group-hover:translate-x-1" />
                </a>
                <Link
                  href="/login"
                  className="rounded-lg border border-gray-300 bg-white px-6 py-3 text-base font-medium text-gray-700 transition-colors hover:bg-gray-50"
                >
                  Launch demo
                </Link>
              </div>
              <a
                href="https://github.com/Turbootzz/Nimbus#-quick-start"
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-gray-500 transition-colors hover:text-gray-700"
              >
                or self-host in 2 minutes →
              </a>
            </div>
          </div>

          {/* Hero Screenshot */}
          <div className="mt-16 sm:mt-20">
            <div className="relative mx-auto max-w-5xl">
              <div className="overflow-hidden rounded-xl border border-gray-200 bg-gray-900 shadow-2xl">
                <Image
                  src="/images/dashboard-preview.png"
                  alt="Nimbus Dashboard Preview"
                  width={2361}
                  height={1025}
                  className="w-full"
                  priority
                  unoptimized
                />
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="bg-white py-20 sm:py-28">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <h2 className="text-3xl font-bold tracking-tight text-gray-900 sm:text-4xl">
              Everything you need for your homelab
            </h2>
            <p className="mx-auto mt-4 max-w-2xl text-lg text-gray-600">
              Built for the homelab community with the features that matter most.
            </p>
          </div>

          <div className="mt-16 grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
            {features.map((feature) => (
              <div
                key={feature.name}
                className="rounded-xl border border-gray-200 bg-gray-50 p-6 transition-shadow hover:shadow-lg"
              >
                <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-sky-500 text-white">
                  <feature.icon className="h-6 w-6" />
                </div>
                <h3 className="mt-4 text-lg font-semibold text-gray-900">{feature.name}</h3>
                <p className="mt-2 text-sm text-gray-600">{feature.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Screenshots Section */}
      <section className="bg-gray-50 py-20 sm:py-28">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <h2 className="text-3xl font-bold tracking-tight text-gray-900 sm:text-4xl">
              See it in action
            </h2>
            <p className="mx-auto mt-4 max-w-2xl text-lg text-gray-600">
              A clean, intuitive interface that puts your services front and center.
            </p>
          </div>

          <div className="mt-16 grid gap-6 sm:grid-cols-2">
            {screenshots.map((screenshot) => (
              <div key={screenshot.label} className="group relative">
                <div className="relative aspect-5/2 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-lg transition-shadow group-hover:shadow-xl">
                  <Image
                    src={screenshot.src}
                    alt={screenshot.alt}
                    fill
                    className="object-cover object-top"
                    unoptimized
                  />
                </div>
                <p className="mt-3 text-center text-sm font-medium text-gray-600">
                  {screenshot.label}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="bg-sky-500 py-16 sm:py-20">
        <div className="mx-auto max-w-7xl px-4 text-center sm:px-6 lg:px-8">
          <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
            Ready to organize your homelab?
          </h2>
          <p className="mx-auto mt-4 max-w-xl text-lg text-sky-100">
            Deploy Nimbus in under 2 minutes with Docker. Only 2 environment variables required!
          </p>
          <div className="mt-8 flex flex-col items-center justify-center gap-4 sm:flex-row">
            <Link
              href="/login"
              className="rounded-lg bg-white px-6 py-3 text-base font-medium text-sky-600 shadow-lg transition-colors hover:bg-gray-50"
            >
              Launch demo
            </Link>
            <a
              href="https://github.com/Turbootzz/Nimbus"
              target="_blank"
              rel="noopener noreferrer"
              className="rounded-lg border-2 border-white px-6 py-3 text-base font-medium text-white transition-colors hover:bg-sky-600"
            >
              View on GitHub
            </a>
          </div>
        </div>
      </section>

      {/* Built on Open Source */}
      <section className="bg-white py-20 sm:py-28">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <h2 className="text-3xl font-bold tracking-tight text-gray-900 sm:text-4xl">
              Built on Open Source
            </h2>
            <p className="mx-auto mt-4 max-w-2xl text-lg text-gray-600">
              Nimbus Cloud runs the exact same open-source Nimbus that you can self-host for free.
              No vendor lock-in — your data is yours.
            </p>
          </div>

          <div className="mt-16 grid gap-8 sm:grid-cols-3">
            <div className="text-center">
              <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-lg bg-sky-500 text-white">
                <svg
                  className="h-6 w-6"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth={1.5}
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M17.25 6.75 22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3-4.5 16.5"
                  />
                </svg>
              </div>
              <h3 className="mt-4 text-lg font-semibold text-gray-900">100% Open Source</h3>
              <p className="mt-2 text-sm text-gray-600">
                Same code on GitHub, same code in the cloud.
              </p>
            </div>
            <div className="text-center">
              <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-lg bg-sky-500 text-white">
                <svg
                  className="h-6 w-6"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth={1.5}
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M13.5 10.5V6.75a4.5 4.5 0 1 1 9 0v3.75M3.75 21.75h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H3.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z"
                  />
                </svg>
              </div>
              <h3 className="mt-4 text-lg font-semibold text-gray-900">No Vendor Lock-in</h3>
              <p className="mt-2 text-sm text-gray-600">Export your data and self-host anytime.</p>
            </div>
            <div className="text-center">
              <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-lg bg-sky-500 text-white">
                <svg
                  className="h-6 w-6"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth={1.5}
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M18 18.72a9.094 9.094 0 0 0 3.741-.479 3 3 0 0 0-4.682-2.72m.94 3.198.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0 1 12 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 0 1 6 18.719m12 0a5.971 5.971 0 0 0-.941-3.197m0 0A5.995 5.995 0 0 0 12 12.75a5.995 5.995 0 0 0-5.058 2.772m0 0a3 3 0 0 0-4.681 2.72 8.986 8.986 0 0 0 3.74.477m.94-3.197a5.971 5.971 0 0 0-.94 3.197M15 6.75a3 3 0 1 1-6 0 3 3 0 0 1 6 0Zm6 3a2.25 2.25 0 1 1-4.5 0 2.25 2.25 0 0 1 4.5 0Zm-13.5 0a2.25 2.25 0 1 1-4.5 0 2.25 2.25 0 0 1 4.5 0Z"
                  />
                </svg>
              </div>
              <h3 className="mt-4 text-lg font-semibold text-gray-900">Community Driven</h3>
              <p className="mt-2 text-sm text-gray-600">Built by and for the homelab community.</p>
            </div>
          </div>

          <div className="mt-12 flex flex-col items-center gap-4">
            <a
              href="https://github.com/Turbootzz/Nimbus"
              target="_blank"
              rel="noopener noreferrer"
              className="group flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-6 py-3 text-base font-medium text-gray-700 transition-colors hover:bg-gray-50"
            >
              <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
                <path
                  fillRule="evenodd"
                  d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0 1 12 6.844a9.59 9.59 0 0 1 2.504.337c1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.02 10.02 0 0 0 22 12.017C22 6.484 17.522 2 12 2Z"
                  clipRule="evenodd"
                />
              </svg>
              View on GitHub →
            </a>
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src="https://img.shields.io/github/stars/Turbootzz/Nimbus?style=social"
              alt="GitHub stars"
              className="h-5"
            />
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-gray-200 bg-white py-12">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col items-center justify-between gap-4 sm:flex-row">
            <div className="flex items-center gap-2">
              <NimbusLogo size={24} />
              <span className="font-semibold text-gray-900">Nimbus</span>
              <span className="text-sm text-gray-500">— Made for the homelab community</span>
            </div>
            <div className="flex items-center gap-6">
              {isNimbusCloud && (
                <a
                  href="https://nimbusapp.dev"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm font-medium hover:opacity-80"
                  style={{ color: 'var(--color-primary, #6366f1)' }}
                >
                  Hosted by Nimbus Cloud
                </a>
              )}
              <a
                href="https://github.com/Turbootzz/Nimbus"
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-gray-600 hover:text-gray-900"
              >
                GitHub
              </a>
              <a
                href="https://github.com/Turbootzz/Nimbus/blob/main/LICENSE"
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-gray-600 hover:text-gray-900"
              >
                AGPL-3.0 License
              </a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  )
}
