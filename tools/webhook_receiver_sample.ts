// Reference Sharecrop webhook receiver.
//
// Executable documentation for the delivery contract described in
// docs/api_reference.md ("Signature Verification"). It verifies each
// delivery the way every production receiver should:
//
// 1. Recompute `v1=` + hex(HMAC-SHA256(secret, "<timestamp>.<raw body>"))
//    over the Sharecrop-Webhook-Timestamp header, a literal dot, and the raw
//    request body, and compare against Sharecrop-Webhook-Signature in
//    constant time (crypto.subtle.verify).
// 2. Reject timestamps more than five minutes from the local clock, bounding
//    replay of a captured delivery.
// 3. Deduplicate by Sharecrop-Webhook-Id: the dispatcher retries on non-2xx
//    answers and timeouts, so the same delivery id can legitimately arrive
//    more than once. A duplicate is acknowledged with 200 (the dispatcher
//    should stop retrying) but not re-processed.
//
// Run it with the secret returned once by POST /api/webhook-subscriptions:
//
//   SHARECROP_WEBHOOK_SECRET=scrop_whsec_... \
//     deno run --allow-net --allow-env tools/webhook_receiver_sample.ts
//
// The dispatcher only posts to https:// URLs on public addresses, so expose
// the receiver through a TLS-terminating tunnel or host when testing against
// a real Sharecrop instance.

interface DeliveryEvent {
  kind: string;
  task_id: string;
  cursor: string;
}

interface DeliveryBody {
  event: DeliveryEvent;
  subscription_id: string;
}

const timestampSkewLimitSeconds = 5 * 60;
const dedupeCapacity = 10_000;
const encoder = new TextEncoder();

function decodeHex(raw: string): Uint8Array<ArrayBuffer> | undefined {
  if (raw.length % 2 !== 0 || /[^0-9a-f]/.test(raw)) {
    return undefined;
  }
  const bytes = new Uint8Array(raw.length / 2);
  for (let index = 0; index < bytes.length; index++) {
    bytes[index] = Number.parseInt(raw.slice(index * 2, index * 2 + 2), 16);
  }
  return bytes;
}

async function importSecret(secret: string): Promise<CryptoKey> {
  return await crypto.subtle.importKey(
    "raw",
    encoder.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["verify"],
  );
}

// verifySignature checks the documented recipe: the signed message is the
// timestamp header, a literal ".", and the raw body bytes.
async function verifySignature(
  key: CryptoKey,
  timestampHeader: string,
  signatureHeader: string,
  body: Uint8Array<ArrayBuffer>,
): Promise<boolean> {
  if (!signatureHeader.startsWith("v1=")) {
    return false;
  }
  const signature = decodeHex(signatureHeader.slice("v1=".length));
  if (signature === undefined) {
    return false;
  }
  const prefix = encoder.encode(`${timestampHeader}.`);
  const message = new Uint8Array(prefix.length + body.length);
  message.set(prefix, 0);
  message.set(body, prefix.length);
  return await crypto.subtle.verify("HMAC", key, signature, message);
}

function timestampIsFresh(timestampHeader: string): boolean {
  if (!/^[0-9]+$/.test(timestampHeader)) {
    return false;
  }
  const sentAtSeconds = Number.parseInt(timestampHeader, 10);
  const nowSeconds = Math.floor(Date.now() / 1000);
  return Math.abs(nowSeconds - sentAtSeconds) <= timestampSkewLimitSeconds;
}

const secret = Deno.env.get("SHARECROP_WEBHOOK_SECRET");
if (secret === undefined || secret === "") {
  console.error("SHARECROP_WEBHOOK_SECRET is required");
  Deno.exit(2);
}
const key = await importSecret(secret);
const port = Number.parseInt(Deno.env.get("PORT") ?? "8787", 10);

// Delivery ids already processed. Bounded so a long-running sample cannot
// grow without limit; a real receiver would persist processed ids instead.
const processedIDs = new Set<string>();

Deno.serve({ port }, async (request: Request): Promise<Response> => {
  if (request.method !== "POST") {
    return new Response("webhook deliveries are POSTs\n", { status: 405 });
  }
  const id = request.headers.get("Sharecrop-Webhook-Id") ?? "";
  const timestamp = request.headers.get("Sharecrop-Webhook-Timestamp") ?? "";
  const signature = request.headers.get("Sharecrop-Webhook-Signature") ?? "";
  if (id === "" || timestamp === "" || signature === "") {
    return new Response("missing delivery headers\n", { status: 400 });
  }
  if (!timestampIsFresh(timestamp)) {
    return new Response("stale or malformed timestamp\n", { status: 400 });
  }
  const body = new Uint8Array(await request.arrayBuffer());
  if (!(await verifySignature(key, timestamp, signature, body))) {
    return new Response("signature mismatch\n", { status: 400 });
  }
  if (processedIDs.has(id)) {
    // Acknowledged but not re-processed: the dispatcher already delivered it.
    console.log(`duplicate delivery ${id} acknowledged`);
    return new Response("ok (duplicate)\n", { status: 200 });
  }
  if (processedIDs.size >= dedupeCapacity) {
    processedIDs.clear();
  }
  processedIDs.add(id);

  const delivery = JSON.parse(
    new TextDecoder().decode(body),
  ) as DeliveryBody;
  console.log(
    `verified delivery ${id}: ${delivery.event.kind}` +
      ` task=${delivery.event.task_id} cursor=${delivery.event.cursor}` +
      ` subscription=${delivery.subscription_id}`,
  );
  return new Response("ok\n", { status: 200 });
});
