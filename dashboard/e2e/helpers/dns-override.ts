/**
 * DNS Override for E2E Tests
 *
 * Resolves *.local.etalbaas.dev to 127.0.0.1 via two mechanisms:
 *
 * 1. Monkey-patch dns.lookup (covers Node http/https modules)
 * 2. Set undici global dispatcher with custom DNS lookup (covers
 *    globalThis.fetch / Supabase JS client)
 *
 * Required because Windows doesn't support wildcard hosts file entries,
 * and each test creates a project with a random subdomain under
 * *.api.local.etalbaas.dev.
 *
 * Import this module at the top of test files that need DNS resolution
 * for project-specific subdomains.
 */

import * as dns from "node:dns";

export const RESOLVED_IP = "127.0.0.1";
export const DOMAIN_SUFFIX = ".local.etalbaas.dev";

/* ------------------------------------------------------------------ */
/*  1. Patch dns.lookup (http/https modules)                           */
/* ------------------------------------------------------------------ */

const origLookup = dns.lookup;

function patchedLookup(
  hostname: string,
  options: dns.LookupOptions | number | undefined | null,
  callback?: dns.LookupCallback,
): void;
function patchedLookup(
  hostname: string,
  callback: dns.LookupCallback,
): void;
function patchedLookup(
  hostname: string,
  optionsOrCallback?: dns.LookupOptions | number | null | dns.LookupCallback,
  maybeCallback?: dns.LookupCallback,
): void {
  const cb =
    typeof optionsOrCallback === "function"
      ? optionsOrCallback
      : maybeCallback;

  if (
    typeof hostname === "string" &&
    hostname.endsWith(DOMAIN_SUFFIX)
  ) {
    if (typeof cb === "function") {
      process.nextTick(() => (cb as Function)(null, RESOLVED_IP, 4));
      return;
    }
  }

  // Fall through to the original lookup
  if (typeof optionsOrCallback === "function") {
    origLookup.call(dns, hostname, optionsOrCallback as dns.LookupCallback);
  } else {
    origLookup.call(
      dns,
      hostname,
      optionsOrCallback as dns.LookupOptions,
      maybeCallback as dns.LookupCallback,
    );
  }
}

(dns as any).lookup = patchedLookup;

/* ------------------------------------------------------------------ */
/*  2. Patch undici global dispatcher (fetch / Supabase client)        */
/* ------------------------------------------------------------------ */

try {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const undici = require("undici");

  const agent = new undici.Agent({
    connect: {
      rejectUnauthorized: false,
      lookup: (
        hostname: string,
        options: object,
        callback: (
          err: Error | null,
          address: string,
          family: number,
        ) => void,
      ) => {
        if (
          typeof hostname === "string" &&
          hostname.endsWith(DOMAIN_SUFFIX)
        ) {
          callback(null, RESOLVED_IP, 4);
          return;
        }
        origLookup(hostname, options as dns.LookupOptions, callback as dns.LookupCallback);
      },
    },
  });

  undici.setGlobalDispatcher(agent);
} catch {
  // undici not available (Node < 18); DNS override for fetch may not work
}

/* ------------------------------------------------------------------ */
/*  Helper: rewrite a domain URL to 127.0.0.1 + return Host header     */
/*  For use with Playwright APIRequestContext which runs in a separate  */
/*  process and doesn't see our dns.lookup patch.                       */
/* ------------------------------------------------------------------ */

/**
 * Given a URL like `https://abc.api.local.etalbaas.dev/rest/v1`,
 * returns `{ url: 'https://127.0.0.1/rest/v1', host: 'abc.api.local.etalbaas.dev' }`.
 * If the hostname doesn't match the wildcard domain, returns the original URL.
 */
export function rewriteUrl(originalUrl: string): { url: string; host: string } {
  const parsed = new URL(originalUrl);
  if (parsed.hostname.endsWith(DOMAIN_SUFFIX)) {
    const host = parsed.hostname;
    parsed.hostname = RESOLVED_IP;
    return { url: parsed.toString(), host };
  }
  return { url: originalUrl, host: parsed.hostname };
}
