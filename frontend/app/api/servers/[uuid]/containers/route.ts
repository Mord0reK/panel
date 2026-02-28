import { NextRequest, NextResponse } from 'next/server'
import { backendFetch } from '@/lib/backend'

export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ uuid: string }> },
) {
  const { uuid } = await params
  const body = await request.json().catch(() => ({}))

  const res = await backendFetch(`/api/servers/${uuid}/containers`, {
    method: 'DELETE',
    body: JSON.stringify(body),
  })

  const text = await res.text()
  let data: unknown
  try {
    data = JSON.parse(text)
  } catch {
    data = { error: text.trim() }
  }
  return NextResponse.json(data, { status: res.status })
}
