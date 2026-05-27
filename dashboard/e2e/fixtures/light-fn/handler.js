export default async function(req) {
  const body = await req.json().catch(() => ({}));
  return new Response(JSON.stringify({
    kind: "light",
    echo: body,
    ts: Date.now(),
  }), {
    headers: { "Content-Type": "application/json" },
  });
}
