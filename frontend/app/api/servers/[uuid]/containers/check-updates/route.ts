import { NextRequest, NextResponse } from 'next/server'
import { backendFetch } from '@/lib/backend'

type RouteParams = {
  params: Promise<{
    uuid: string
  }>
}

export async function POST(
  request: NextRequest,
  { params }: RouteParams
) {
  const { uuid } = await params

  try {
    const body = await request.json()

    const response = await backendFetch(
      `/api/servers/${uuid}/containers/check-updates`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
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
