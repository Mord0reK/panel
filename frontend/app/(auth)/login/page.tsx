import { redirect } from 'next/navigation'
import { backendFetch } from '@/lib/backend'
import LoginForm from '@/components/auth/LoginForm'

export default async function LoginPage() {
  try {
    const res = await backendFetch('/api/auth/status')
    if (res.ok) {
      const data = await res.json() as { setup_required?: boolean }
      if (data.setup_required) {
        redirect('/setup')
      }
    }
  } catch {
    // backend niedostępny — renderuj formularz
  }

  return <LoginForm />
}
