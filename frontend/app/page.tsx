import { redirect } from 'next/navigation'

// Middleware (proxy.ts) obsługuje auth redirect.
// Zalogowani → /dashboard (middleware), niezalogowani → /login (middleware).
// Ten redirect obsługuje bezpośrednie wejście na /.
export default function RootPage() {
  redirect('/dashboard')
}
