// TODO: przyszłość — widok logów
export default async function LogsPage({
  params,
}: {
  params: Promise<{ uuid: string }>
}) {
  const { uuid } = await params
  return (
    <main className="p-8">
      <h1 className="text-zinc-100 text-2xl font-semibold">
        Logi serwera {uuid} — Coming soon
      </h1>
    </main>
  )
}
