export type Theme = "light" | "dark" | "system"
export type ThemeColorScheme = "default" | "ocean" | "sunset" | "forest" | "custom"

export interface CustomThemeColors {
  light_primary: string
  light_accent: string
  dark_primary: string
  dark_accent: string
}

type ThemeTone = "light" | "dark"

interface ThemeBaseColors {
  primary: string
  accent: string
}

interface ThemeColorPreset {
  light: ThemeBaseColors
  dark: ThemeBaseColors
}

const THEME_KEY = "theme"
const THEME_COLOR_SCHEME_KEY = "theme_color_scheme"
const THEME_CUSTOM_COLORS_KEY = "theme_custom_colors"
const HEX_COLOR_PATTERN = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i

const DEFAULT_CUSTOM_THEME_COLORS: CustomThemeColors = {
  light_primary: "#2563eb",
  light_accent: "#14b8a6",
  dark_primary: "#60a5fa",
  dark_accent: "#2dd4bf",
}

const THEME_COLOR_PRESETS: Record<
  Exclude<ThemeColorScheme, "default" | "custom">,
  ThemeColorPreset
> = {
  ocean: {
    light: { primary: "#2563eb", accent: "#0ea5e9" },
    dark: { primary: "#60a5fa", accent: "#22d3ee" },
  },
  sunset: {
    light: { primary: "#ea580c", accent: "#ec4899" },
    dark: { primary: "#fb923c", accent: "#f472b6" },
  },
  forest: {
    light: { primary: "#166534", accent: "#0f766e" },
    dark: { primary: "#4ade80", accent: "#2dd4bf" },
  },
}

export const THEME_COLOR_SCHEME_OPTIONS: readonly ThemeColorScheme[] = [
  "default",
  "ocean",
  "sunset",
  "forest",
  "custom",
]

const THEME_COLOR_VARIABLES = [
  "--primary",
  "--primary-foreground",
  "--accent",
  "--accent-foreground",
  "--ring",
  "--sidebar-primary",
  "--sidebar-primary-foreground",
  "--sidebar-ring",
  "--chart-1",
  "--chart-2",
  "--chart-3",
  "--chart-4",
  "--chart-5",
] as const

type ThemeColorVariable = (typeof THEME_COLOR_VARIABLES)[number]
type ThemeColorVariableMap = Record<ThemeColorVariable, string>

function normalizeHexColor(value: string, fallback: string): string {
  const next = value.trim().toLowerCase()
  if (!HEX_COLOR_PATTERN.test(next)) {
    return fallback
  }

  if (next.length === 4) {
    const [hash, r, g, b] = next
    return `${hash}${r}${r}${g}${g}${b}${b}`
  }

  return next
}

function hexToRgb(hex: string): [number, number, number] | null {
  const normalized = normalizeHexColor(hex, "")
  if (normalized.length !== 7) {
    return null
  }

  const r = Number.parseInt(normalized.slice(1, 3), 16)
  const g = Number.parseInt(normalized.slice(3, 5), 16)
  const b = Number.parseInt(normalized.slice(5, 7), 16)

  if ([r, g, b].some((value) => Number.isNaN(value))) {
    return null
  }

  return [r, g, b]
}

function rgbToHex(r: number, g: number, b: number): string {
  const clamp = (value: number): number => Math.min(255, Math.max(0, Math.round(value)))
  const channel = (value: number): string => clamp(value).toString(16).padStart(2, "0")
  return `#${channel(r)}${channel(g)}${channel(b)}`
}

function mixHex(firstHex: string, secondHex: string, secondRatio: number): string {
  const firstRgb = hexToRgb(firstHex)
  const secondRgb = hexToRgb(secondHex)
  if (!firstRgb || !secondRgb) {
    return firstHex
  }

  const ratio = Math.min(1, Math.max(0, secondRatio))
  const inverse = 1 - ratio

  return rgbToHex(
    firstRgb[0] * inverse + secondRgb[0] * ratio,
    firstRgb[1] * inverse + secondRgb[1] * ratio,
    firstRgb[2] * inverse + secondRgb[2] * ratio
  )
}

function getContrastTextColor(backgroundHex: string, lightText: string, darkText: string): string {
  const rgb = hexToRgb(backgroundHex)
  if (!rgb) {
    return lightText
  }

  const toLinear = (channel: number): number => {
    const normalized = channel / 255
    if (normalized <= 0.03928) {
      return normalized / 12.92
    }
    return ((normalized + 0.055) / 1.055) ** 2.4
  }

  const luminance = 0.2126 * toLinear(rgb[0]) + 0.7152 * toLinear(rgb[1]) + 0.0722 * toLinear(rgb[2])
  return luminance > 0.45 ? darkText : lightText
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null
}

function normalizeCustomThemeColors(colors: CustomThemeColors): CustomThemeColors {
  return {
    light_primary: normalizeHexColor(colors.light_primary, DEFAULT_CUSTOM_THEME_COLORS.light_primary),
    light_accent: normalizeHexColor(colors.light_accent, DEFAULT_CUSTOM_THEME_COLORS.light_accent),
    dark_primary: normalizeHexColor(colors.dark_primary, DEFAULT_CUSTOM_THEME_COLORS.dark_primary),
    dark_accent: normalizeHexColor(colors.dark_accent, DEFAULT_CUSTOM_THEME_COLORS.dark_accent),
  }
}

function parseCustomThemeColors(raw: string | null): CustomThemeColors {
  if (!raw) {
    return { ...DEFAULT_CUSTOM_THEME_COLORS }
  }

  try {
    const parsed: unknown = JSON.parse(raw)
    if (!isRecord(parsed)) {
      return { ...DEFAULT_CUSTOM_THEME_COLORS }
    }

    return normalizeCustomThemeColors({
      light_primary:
        typeof parsed.light_primary === "string"
          ? parsed.light_primary
          : DEFAULT_CUSTOM_THEME_COLORS.light_primary,
      light_accent:
        typeof parsed.light_accent === "string"
          ? parsed.light_accent
          : DEFAULT_CUSTOM_THEME_COLORS.light_accent,
      dark_primary:
        typeof parsed.dark_primary === "string"
          ? parsed.dark_primary
          : DEFAULT_CUSTOM_THEME_COLORS.dark_primary,
      dark_accent:
        typeof parsed.dark_accent === "string"
          ? parsed.dark_accent
          : DEFAULT_CUSTOM_THEME_COLORS.dark_accent,
    })
  } catch {
    return { ...DEFAULT_CUSTOM_THEME_COLORS }
  }
}

function getResolvedThemeTone(theme: Theme): ThemeTone {
  if (theme === "light" || theme === "dark") {
    return theme
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
}

function getThemeColorPreset(
  scheme: ThemeColorScheme,
  customColors: CustomThemeColors
): ThemeColorPreset | null {
  if (scheme === "default") {
    return null
  }

  if (scheme === "custom") {
    return {
      light: {
        primary: customColors.light_primary,
        accent: customColors.light_accent,
      },
      dark: {
        primary: customColors.dark_primary,
        accent: customColors.dark_accent,
      },
    }
  }

  return THEME_COLOR_PRESETS[scheme]
}

function createThemeColorVariables(baseColors: ThemeBaseColors, tone: ThemeTone): ThemeColorVariableMap {
  const surface = tone === "light" ? "#ffffff" : "#111827"
  const text = tone === "light" ? "#111827" : "#f9fafb"
  const ringTarget = tone === "light" ? "#ffffff" : "#000000"
  const accentSurface = mixHex(surface, baseColors.accent, tone === "light" ? 0.14 : 0.22)

  return {
    "--primary": baseColors.primary,
    "--primary-foreground": getContrastTextColor(baseColors.primary, "#ffffff", "#111827"),
    "--accent": accentSurface,
    "--accent-foreground": text,
    "--ring": mixHex(baseColors.primary, ringTarget, tone === "light" ? 0.28 : 0.34),
    "--sidebar-primary": baseColors.primary,
    "--sidebar-primary-foreground": getContrastTextColor(baseColors.primary, "#ffffff", "#111827"),
    "--sidebar-ring": mixHex(baseColors.primary, ringTarget, tone === "light" ? 0.28 : 0.34),
    "--chart-1": baseColors.primary,
    "--chart-2": baseColors.accent,
    "--chart-3": mixHex(baseColors.primary, baseColors.accent, 0.5),
    "--chart-4": mixHex(baseColors.accent, tone === "light" ? "#f59e0b" : "#fbbf24", 0.35),
    "--chart-5": mixHex(baseColors.primary, tone === "light" ? "#10b981" : "#34d399", 0.35),
  }
}

function clearThemeColorOverrides(): void {
  for (const variable of THEME_COLOR_VARIABLES) {
    document.documentElement.style.removeProperty(variable)
  }
}

function applyThemeColorStyles(
  theme: Theme,
  scheme: ThemeColorScheme,
  customColors: CustomThemeColors
): void {
  const preset = getThemeColorPreset(scheme, customColors)
  if (!preset) {
    clearThemeColorOverrides()
    return
  }

  const tone = getResolvedThemeTone(theme)
  const variables = createThemeColorVariables(preset[tone], tone)

  for (const variable of THEME_COLOR_VARIABLES) {
    document.documentElement.style.setProperty(variable, variables[variable])
  }
}

export function getTheme(): Theme {
  const stored = localStorage.getItem(THEME_KEY)
  if (stored === "light" || stored === "dark" || stored === "system") {
    return stored
  }
  return "system"
}

export function getThemeColorScheme(): ThemeColorScheme {
  const stored = localStorage.getItem(THEME_COLOR_SCHEME_KEY)
  if (
    stored === "default" ||
    stored === "ocean" ||
    stored === "sunset" ||
    stored === "forest" ||
    stored === "custom"
  ) {
    return stored
  }
  return "default"
}

export function getDefaultCustomThemeColors(): CustomThemeColors {
  return { ...DEFAULT_CUSTOM_THEME_COLORS }
}

export function getCustomThemeColors(): CustomThemeColors {
  return parseCustomThemeColors(localStorage.getItem(THEME_CUSTOM_COLORS_KEY))
}

export function getThemeColorSchemeSwatch(
  scheme: ThemeColorScheme,
  customColors: CustomThemeColors = getCustomThemeColors()
): readonly [string, string] {
  if (scheme === "default") {
    return ["#111827", "#6b7280"] as const
  }

  if (scheme === "custom") {
    return [customColors.light_primary, customColors.light_accent] as const
  }

  const preset = THEME_COLOR_PRESETS[scheme]
  return [preset.light.primary, preset.light.accent] as const
}

export function applyTheme(theme: Theme): void {
  localStorage.setItem(THEME_KEY, theme)

  const isDark =
    theme === "dark" ||
    (theme === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches)

  document.documentElement.classList.toggle("dark", isDark)
  applyThemeColorStyles(theme, getThemeColorScheme(), getCustomThemeColors())
}

export function applyThemeColorScheme(
  scheme: ThemeColorScheme,
  customColors: CustomThemeColors = getCustomThemeColors()
): void {
  const normalizedColors = normalizeCustomThemeColors(customColors)
  localStorage.setItem(THEME_COLOR_SCHEME_KEY, scheme)
  localStorage.setItem(THEME_CUSTOM_COLORS_KEY, JSON.stringify(normalizedColors))
  applyThemeColorStyles(getTheme(), scheme, normalizedColors)
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

export function useTheme(): "light" | "dark" {
  if (typeof window === "undefined") return "light"

  const theme = getTheme()
  if (theme === "light" || theme === "dark") {
    return theme
  }

  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
}
