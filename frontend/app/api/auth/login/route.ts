import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:8080'
const COOKIE_MAX_AGE = 60 * 60 * 24 * 7 // 7 dni

export async function POST(request: NextRequest) {
  const body = await request.json()

  const res = await fetch(`${BACKEND_URL}/api/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })

  if (!res.ok) {
    const text = await res.text().catch(() => 'Login failed')
    return NextResponse.json({ error: text }, { status: res.status })
  }

  const { token } = (await res.json()) as { token: string }

  const response = NextResponse.json({ success: true })
  response.cookies.set('token', token, {
    httpOnly: true,
    sameSite: 'strict',
    path: '/',
    maxAge: COOKIE_MAX_AGE,
    secure: process.env.NODE_ENV === 'production',
  })

  return response
}
