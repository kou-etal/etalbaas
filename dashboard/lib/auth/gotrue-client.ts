import { GoTrueClient } from "@supabase/gotrue-js";

export const gotrue = new GoTrueClient({
  url: process.env.NEXT_PUBLIC_GOTRUE_URL || "http://localhost:9999",
  autoRefreshToken: true,
  persistSession: true,
  storageKey: "etalbaas-auth",
});
