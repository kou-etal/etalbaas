// Transport configuration for Connect RPC
// This module is preserved for when proper proto codegen is set up.
// Currently, lib/api/clients.ts uses direct fetch calls.

export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
