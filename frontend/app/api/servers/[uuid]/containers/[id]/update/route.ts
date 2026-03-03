import { NextRequest, NextResponse } from 'next/server'

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
  const backendUrl = process.env.BACKEND_URL || 'http://localhost:8080'

  try {
    const response = await fetch(
      `${backendUrl}/api/servers/${uuid}/containers/${id}/update`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      }
    )

    if (!response.ok) {
      const errorText = await response.text()
      return NextResponse.json(
        { error: errorText || 'Failed to update container' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Failed to update container' },
      { status: 500 }
    )
  }
}
