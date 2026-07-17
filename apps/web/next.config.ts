import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Static export served by nginx — mirrors Astro's build.format "directory"
  // (each page emitted as dir/index.html, URLs keep their trailing slash).
  output: "export",
  trailingSlash: true,
  // No image-optimization server under export; images are hand-sized assets.
  images: { unoptimized: true },
};

export default nextConfig;
