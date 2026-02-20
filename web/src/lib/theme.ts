export type Theme = "light" | "dark" | "system"

const THEME_KEY = "theme"

export function getTheme(): Theme {
  const stored = localStorage.getItem(THEME_KEY)
  if (stored === "light" || stored === "dark" || stored === "system") {
    return stored
  }
  return "system"
}

export function applyTheme(theme: Theme): void {
  localStorage.setItem(THEME_KEY, theme)

  const isDark =
    theme === "dark" ||
    (theme === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches)

  document.documentElement.classList.toggle("dark", isDark)
}

export function initTheme(): void {
  applyTheme(getTheme())
}

export function watchSystemTheme(): void {
  const media = window.matchMedia("(prefers-color-scheme: dark)")
  media.addEventListener("change", () => {
    if (getTheme() === "system") {
      applyTheme("system")
    }
  })
}
