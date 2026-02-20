import { NextResponse } from 'next/server'

// Wylogowanie — usuwa httpOnly cookie z tokenem
export async function POST() {
  const response = NextResponse.json({ success: true })
  response.cookies.set('token', '', {
    httpOnly: true,
    sameSite: 'strict',
    path: '/',
    maxAge: 0,
  })
  return response
}
