import path from "path"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("/node_modules/")) {
            return undefined
          }

          if (id.includes("/node_modules/react/") || id.includes("/node_modules/react-dom/")) {
            return "vendor-react"
          }

          if (id.includes("/node_modules/react-router/") || id.includes("/node_modules/react-router-dom/")) {
            return "vendor-router"
          }

          if (
            id.includes("/node_modules/i18next/") ||
            id.includes("/node_modules/react-i18next/") ||
            id.includes("/node_modules/i18next-browser-languagedetector/")
          ) {
            return "vendor-i18n"
          }

          if (id.includes("/node_modules/lucide-react/")) {
            return "vendor-icons"
          }
        },
      },
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      sonner: path.resolve(__dirname, "./src/lib/sonner.ts"),
    },
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      "/mcp": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
})
