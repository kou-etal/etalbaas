/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080",
    NEXT_PUBLIC_GOTRUE_URL: process.env.NEXT_PUBLIC_GOTRUE_URL || "http://localhost:9999",
  },
};

export default nextConfig;
