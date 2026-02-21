import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

const PUBLIC_PATHS = ['/login', '/setup']

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:8080'

async function checkSetupRequired(): Promise<boolean> {
  try {
    const res = await fetch(`${BACKEND_URL}/api/auth/status`)
    const data = await res.json() as { setup_required?: boolean }
    return data.setup_required ?? false
  } catch {
    return false
  }
}

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl
  const token = request.cookies.get('token')?.value

  // Przepuść publiczne ścieżki i zasoby Next.js
  if (
    PUBLIC_PATHS.some((p) => pathname.startsWith(p)) ||
    pathname.startsWith('/_next') ||
    pathname.startsWith('/favicon')
  ) {
    // Jeśli zalogowany i wchodzi na /login lub /setup → redirect na dashboard
    if (token) {
      return NextResponse.redirect(new URL('/dashboard', request.url))
    }
    return NextResponse.next()
  }

  // API routes obsługują własną autentykację — nie blokuj middleware
  if (pathname.startsWith('/api')) {
    return NextResponse.next()
  }

  // Brak tokena → sprawdź czy setup wymagany
  if (!token) {
    const setupRequired = await checkSetupRequired()
    if (setupRequired) {
      return NextResponse.redirect(new URL('/setup', request.url))
    }
    const loginUrl = new URL('/login', request.url)
    loginUrl.searchParams.set('from', pathname)
    return NextResponse.redirect(loginUrl)
  }

  return NextResponse.next()
}

export const config = {
  matcher: [
    /*
     * Dopasuj wszystkie ścieżki oprócz:
     * - _next/static
     * - _next/image
     * - favicon.ico
     */
    '/((?!_next/static|_next/image|favicon.ico).*)',
  ],
}