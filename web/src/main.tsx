import { createRoot } from "react-dom/client"
import "./index.css"
import "@/i18n"
import { initTheme, watchSystemTheme } from "@/lib/theme"
import App from "./App"

initTheme()
watchSystemTheme()

createRoot(document.getElementById("root")!).render(<App />)
