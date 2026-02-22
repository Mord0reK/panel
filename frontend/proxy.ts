import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

const PUBLIC_PATHS = ['/login', '/setup']

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:8080'

type AuthStatus = {
  setup_required: boolean
  authenticated: boolean
}

async function checkAuthStatus(token?: string): Promise<AuthStatus> {
  try {
    const headers: Record<string, string> = {}
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }
    const res = await fetch(`${BACKEND_URL}/api/auth/status`, { headers })
    const data = await res.json() as AuthStatus
    return {
      setup_required: data.setup_required ?? false,
      authenticated: data.authenticated ?? false,
    }
  } catch {
    return { setup_required: false, authenticated: false }
  }
}

export async function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl
  const token = request.cookies.get('token')?.value

  // Przepuść publiczne ścieżki i zasoby Next.js
  if (
    PUBLIC_PATHS.some((p) => pathname.startsWith(p)) ||
    pathname.startsWith('/_next') ||
    pathname.startsWith('/favicon')
  ) {
    // Jeśli zalogowany i wchodzi na /login lub /setup → zweryfikuj token
    if (token) {
      const authStatus = await checkAuthStatus(token)

      // Token prawidłowy → dashboard
      if (authStatus.authenticated) {
        // Ale jeśli setup wymagany → /setup
        if (authStatus.setup_required) {
          return NextResponse.redirect(new URL('/setup', request.url))
        }
        return NextResponse.redirect(new URL('/dashboard', request.url))
      }

      // Token nieprawidłowy → wyczyść cookie i zostań na stronie
      const response = NextResponse.next()
      response.cookies.set('token', '', { expires: new Date(0) })
      return response
    }
    return NextResponse.next()
  }

  // API routes obsługują własną autentykację — nie blokuj middleware
  if (pathname.startsWith('/api')) {
    return NextResponse.next()
  }

  // Brak tokena lub nieprawidłowy token → sprawdź czy setup wymagany
  if (!token) {
    const authStatus = await checkAuthStatus()
    if (authStatus.setup_required) {
      return NextResponse.redirect(new URL('/setup', request.url))
    }
    const loginUrl = new URL('/login', request.url)
    loginUrl.searchParams.set('from', pathname)
    return NextResponse.redirect(loginUrl)
  }

  // Jest token w cookies → zweryfikuj go
  const authStatus = await checkAuthStatus(token)

  // Token nieprawidłowy → wyczyść cookie i przekieruj na login
  if (!authStatus.authenticated) {
    const response = NextResponse.redirect(new URL('/login', request.url))
    response.cookies.set('token', '', { expires: new Date(0) })
    return response
  }

  // Token prawidłowy, ale setup wymagany → /setup
  if (authStatus.setup_required) {
    return NextResponse.redirect(new URL('/setup', request.url))
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