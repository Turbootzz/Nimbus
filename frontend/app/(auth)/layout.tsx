export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-blue-50 via-indigo-50 to-purple-50">
      <div className="w-full max-w-md px-6">
        {/* Logo/Branding */}
        <div className="mb-8 text-center">
          <h1 className="mb-2 text-4xl font-bold text-gray-900">☁️ Nimbus</h1>
          <p className="text-gray-600">Your Personal Homelab Dashboard</p>
        </div>

        {/* Auth Form Container */}
        {children}
      </div>
    </div>
  )
}
