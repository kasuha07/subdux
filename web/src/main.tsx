import { createRoot } from "react-dom/client"
import "./index.css"
import { i18nReady } from "@/i18n"
import { initTheme, watchSystemTheme } from "@/lib/theme"
import App from "./App"

initTheme()
watchSystemTheme()

if (import.meta.env.DEV) {
  void import("react-grep")
}

i18nReady
  .catch((error: unknown) => {
    console.error("Failed to initialize i18n", error)
  })
  .finally(() => {
    createRoot(document.getElementById("root")!).render(<App />)
  })
