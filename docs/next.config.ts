import { NextConfig } from "next";
import nextra from "nextra";

const withNextra = nextra({
  latex: true,
  defaultShowCopyCode: true,
  contentDirBasePath: "/",
  unstable_shouldAddLocaleToLinks: true,
});

const isGithubPages = process.env.NEXT_PUBLIC_GITHUB_PAGES === "true";

const nextConfig: NextConfig = withNextra({
  reactStrictMode: true,
  output: "export",
  experimental: {
    cpus: 1,
  },
  i18n: {
    locales: ["en", "it", "es", "fr", "de"],
    defaultLocale: "en",
  },
  ...(isGithubPages && {
    basePath: "/LWN-Simulator/docs",
    assetPrefix: "/LWN-Simulator/docs/",
  }),

  images: {
    unoptimized: true,
  },
});

export default nextConfig;
