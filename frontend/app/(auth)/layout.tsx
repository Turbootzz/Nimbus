export default function AuthLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 via-indigo-50 to-purple-50">
      <div className="w-full max-w-md px-6">
        {/* Logo/Branding */}
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">
            ☁️ Nimbus
          </h1>
          <p className="text-gray-600">
            Your Personal Homelab Dashboard
          </p>
        </div>

        {/* Auth Form Container */}
        {children}
      </div>
    </div>
  )
}
