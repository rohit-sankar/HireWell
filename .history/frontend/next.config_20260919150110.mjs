import { PHASE_DEVELOPMENT_SERVER } from "next/constants";

/** @type {import('next').NextConfig} */
const nextConfig = (phase) => ({
  distDir: phase === PHASE_DEVELOPMENT_SERVER ? ".next-dev" : ".next",
  reactStrictMode: true,
  eslint: {
    // linting is run separately via `npm run lint` in CI; don't block prod
    // builds on it here so `next build` failures are always real type/build
    // errors, not lint nits.
    ignoreDuringBuilds: false,
  },
});

export default nextConfig;
