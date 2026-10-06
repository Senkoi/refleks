import react from "@vitejs/plugin-react";
import path from "path";
import { defineConfig } from "vite";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  define: {
    "import.meta.env.VITE_AIMMEOW_BUILD": JSON.stringify(process.env.GITHUB_SHA?.slice(0, 8) || "本地构建"),
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "@wails": path.resolve(__dirname, "./wailsjs"),
    },
  },
  build: {
    sourcemap: false,
  },
});
