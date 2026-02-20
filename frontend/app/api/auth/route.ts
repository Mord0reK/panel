import { NextResponse } from 'next/server'

// Bazowy endpoint /api/auth — używaj podścieżek: /status, /login, /setup, /logout
export async function GET() {
  return NextResponse.json({ error: 'Not Found' }, { status: 404 })
}
