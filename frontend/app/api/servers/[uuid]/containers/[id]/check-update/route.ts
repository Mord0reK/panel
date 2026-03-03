import { NextRequest, NextResponse } from 'next/server'
import { backendFetch } from '@/lib/backend'

type RouteParams = {
  params: Promise<{
    uuid: string
    id: string
  }>
}

export async function POST(
  request: NextRequest,
  { params }: RouteParams
) {
  const { uuid, id } = await params

  try {
    const response = await backendFetch(
      `/api/servers/${uuid}/containers/${id}/check-update`,
      {
        method: 'POST',
      }
    )

    if (!response.ok) {
      const errorText = await response.text()
      return NextResponse.json(
        { error: errorText || 'Failed to check for updates' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Failed to check for updates' },
      { status: 500 }
    )
  }
}
