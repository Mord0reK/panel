import { redirect } from 'next/navigation'

// Redirect na metryki
export default async function ServerPage({
  params,
}: {
  params: Promise<{ uuid: string }>
}) {
  const { uuid } = await params
  redirect(`/servers/${uuid}/metrics`)
}
