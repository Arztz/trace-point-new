import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, ".", "");
  
  return {
    plugins: [react(), tailwindcss()],
    server: {
      allowedHosts: ["trace.arztz.top", "localhost"],
      host: "0.0.0.0",
      port: 5173,
      proxy: {
        "/api": {
          target: env.VITE_BACKEND_URL || "http://localhost:8088",
          changeOrigin: true,
          secure: false,
        },
        "/health": {
          target: env.VITE_BACKEND_URL || "http://localhost:8088",
          changeOrigin: true,
          secure: false,
        },
      },
    },
  };
});
