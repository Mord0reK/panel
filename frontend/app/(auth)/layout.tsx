export default function AuthLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <h1 className="text-2xl font-bold text-foreground tracking-tight">
            Panel
          </h1>
          <p className="text-muted-foreground text-sm mt-1">
            Server Dashboard &amp; Monitor
          </p>
        </div>
        {children}
      </div>
    </div>
  )
}
