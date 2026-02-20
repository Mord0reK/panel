import { NextResponse } from 'next/server'
import { backendFetch } from '@/lib/backend'

export async function GET() {
  const res = await backendFetch('/api/servers')
  const data = await res.json()
  return NextResponse.json(data, { status: res.status })
}
