// TODO: przyszłość — widok logów
export default async function LogsPage({
  params,
}: {
  params: Promise<{ uuid: string }>
}) {
  const { uuid } = await params
  return (
    <section className="space-y-3 rounded-lg border border-zinc-800 bg-zinc-900/40 px-4 py-6 sm:px-6 sm:py-8">
      <h1 className="text-2xl font-semibold text-zinc-100">
        Logi serwera {uuid} — Coming soon
      </h1>
    </section>
  )
}
