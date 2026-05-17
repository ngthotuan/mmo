import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  images: {
    remotePatterns: [
      { protocol: "https", hostname: "**.r2.cloudflarestorage.com" },
      { protocol: "https", hostname: "images.pexels.com" },
      { protocol: "https", hostname: "pixabay.com" },
    ],
  },
};

export default nextConfig;
