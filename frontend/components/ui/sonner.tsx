'use client'

import { Toaster as Sonner } from 'sonner'

type ToasterProps = React.ComponentProps<typeof Sonner>

const Toaster = ({ ...props }: ToasterProps) => {
  return (
    <Sonner
      theme="dark"
      className="toaster group"
      position="top-right"
      toastOptions={{
        classNames: {
          toast:
            'group toast group-[.toaster]:bg-zinc-900 group-[.toaster]:text-zinc-100 group-[.toaster]:border-zinc-800 group-[.toaster]:shadow-lg',
          description: 'group-[.toast]:text-zinc-400',
          actionButton:
            'group-[.toast]:bg-zinc-100 group-[.toast]:text-zinc-900',
          cancelButton:
            'group-[.toast]:bg-zinc-800 group-[.toast]:text-zinc-100',
          success: 'group-[.toaster]:!bg-emerald-950 group-[.toaster]:!border-emerald-800 group-[.toaster]:!text-emerald-100',
          error: 'group-[.toaster]:!bg-red-950 group-[.toaster]:!border-red-800 group-[.toaster]:!text-red-100',
          warning: 'group-[.toaster]:!bg-amber-950 group-[.toaster]:!border-amber-800 group-[.toaster]:!text-amber-100',
          info: 'group-[.toaster]:!bg-blue-950 group-[.toaster]:!border-blue-800 group-[.toaster]:!text-blue-100',
        },
      }}
      {...props}
    />
  )
}

export { Toaster }
