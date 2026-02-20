import { redirect } from 'next/navigation'
import { backendFetch } from '@/lib/backend'
import SetupForm from '@/components/auth/SetupForm'

export default async function SetupPage() {
  try {
    const res = await backendFetch('/api/auth/status')
    if (res.ok) {
      const data = await res.json() as { setup_required?: boolean }
      if (!data.setup_required) {
        redirect('/login')
      }
    }
  } catch {
    // backend niedostępny — renderuj formularz
  }

  return <SetupForm />
}
