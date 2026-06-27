import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// fs.allow lets us import the canonical shared/styles.css from outside the app root.
export default defineConfig({
  plugins: [react()],
  server: { fs: { allow: [".", "../../shared"] } },
});
