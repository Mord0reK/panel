import { NextResponse } from 'next/server'
import { backendFetch } from '@/lib/backend'

export async function GET() {
  const res = await backendFetch('/api/services/cloudflare/dns')

  const text = await res.text()
  let data: unknown
  try {
    data = JSON.parse(text)
  } catch {
    data = { error: text.trim() }
  }

  return NextResponse.json(data, { status: res.status })
}
