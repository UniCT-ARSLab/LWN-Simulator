import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "export",
  assetPrefix: process.env.ELECTRON_BUILD === "true" ? "./" : undefined,
  trailingSlash: true,

  images: {
    unoptimized: true,
  },
};

export default nextConfig;
