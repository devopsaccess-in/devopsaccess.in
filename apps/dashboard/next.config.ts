import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Node server behind nginx (app.devopsaccess.in / → 127.0.0.1:3001).
  // standalone bundles server + deps so Ansible rsyncs one directory.
  output: "standalone",
  images: { unoptimized: true },
};

export default nextConfig;
