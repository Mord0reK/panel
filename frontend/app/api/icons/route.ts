import fs from 'fs'
import path from 'path'
import { NextResponse } from 'next/server'
import type { CustomIcon } from '@/types'

const ICONS_DIR = path.join(process.cwd(), 'public', 'icons')

export async function GET() {
  try {
    if (!fs.existsSync(ICONS_DIR)) {
      return NextResponse.json([] as CustomIcon[])
    }

    const files = fs.readdirSync(ICONS_DIR).filter((f) => {
      const ext = path.extname(f).toLowerCase()
      return ext === '.svg' || ext === '.png' || ext === '.webp'
    })

    const icons: CustomIcon[] = files.map((name) => ({
      name,
      url: `/icons/${name}`,
    }))

    return NextResponse.json(icons)
  } catch {
    return NextResponse.json([] as CustomIcon[])
  }
}
