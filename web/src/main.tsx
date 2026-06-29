import { createRoot } from "react-dom/client"
import "./index.css"
import { getInitialLanguage, i18nReady, preloadLocale } from "@/i18n"
import { initTheme, watchSystemTheme } from "@/lib/theme"
import App from "./App"

initTheme()
watchSystemTheme()
void preloadLocale(getInitialLanguage())

if (import.meta.env.DEV) {
  void import("react-grep")
}

void i18nReady.catch((error: unknown) => {
  console.error("Failed to initialize i18n", error)
})

createRoot(document.getElementById("root")!).render(<App />)
