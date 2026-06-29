import i18n from "i18next"
import type { BackendModule, ReadCallback, ResourceLanguage } from "i18next"
import { initReactI18next } from "react-i18next"

type LocaleCode = "en" | "zh-CN" | "ja"

const localeLoaders: Record<LocaleCode, () => Promise<ResourceLanguage>> = {
  en: () => import("./en").then((module) => module.default),
  "zh-CN": () => import("./zh-CN").then((module) => module.default),
  ja: () => import("./ja").then((module) => module.default),
}

const fallbackLocale = "en"
const supportedLocales: readonly LocaleCode[] = ["en", "zh-CN", "ja"]
const localeCache = new Map<LocaleCode, Promise<ResourceLanguage>>()

function readStoredLanguage(): LocaleCode | null {
  if (typeof window === "undefined") {
    return null
  }

  try {
    const stored = window.localStorage.getItem("language")
    if (stored === "en" || stored === "zh-CN" || stored === "ja") {
      return stored
    }
  } catch {
    void 0
  }

  return null
}

function detectBrowserLanguage(): LocaleCode {
  const stored = readStoredLanguage()
  if (stored) {
    return stored
  }

  if (typeof navigator === "undefined") {
    return fallbackLocale
  }

  const preferred = navigator.languages?.find((language) => supportedLocales.includes(language as LocaleCode))
  if (preferred === "en" || preferred === "zh-CN" || preferred === "ja") {
    return preferred
  }

  const language = navigator.language
  if (language === "en" || language === "zh-CN" || language === "ja") {
    return language
  }

  return fallbackLocale
}

function loadLocale(language: LocaleCode): Promise<ResourceLanguage> {
  const cached = localeCache.get(language)
  if (cached) {
    return cached
  }

  const loader = localeLoaders[language] ?? localeLoaders[fallbackLocale]
  const promise = loader()
  localeCache.set(language, promise)
  return promise
}

export function preloadLocale(language: string): Promise<ResourceLanguage> {
  const normalized = (supportedLocales.includes(language as LocaleCode) ? language : fallbackLocale) as LocaleCode
  return loadLocale(normalized)
}

export function preloadSupportedLocales(excludeLanguage?: string): void {
  for (const language of supportedLocales) {
    if (language === excludeLanguage) {
      continue
    }
    void loadLocale(language)
  }
}

const dynamicLocaleBackend: BackendModule = {
  type: "backend",
  init() {},
  read(language: string, _namespace: string, callback: ReadCallback) {
    const normalized = (supportedLocales.includes(language as LocaleCode) ? language : fallbackLocale) as LocaleCode

    loadLocale(normalized)
      .then((locale) => callback(null, locale))
      .catch((error: unknown) => callback(error instanceof Error ? error : new Error(String(error)), null))
  },
}

i18n.use(dynamicLocaleBackend).use(initReactI18next)

export const i18nReady = i18n.init({
  fallbackLng: fallbackLocale,
  supportedLngs: supportedLocales,
  ns: ["translation"],
  defaultNS: "translation",
  interpolation: {
    escapeValue: false,
  },
  lng: detectBrowserLanguage(),
})

i18n.on("languageChanged", (language) => {
  if (typeof window === "undefined") {
    return
  }

  try {
    window.localStorage.setItem("language", language)
  } catch {
    void 0
  }
})

export function getInitialLanguage(): LocaleCode {
  return i18n.language === "zh-CN" || i18n.language === "ja" ? i18n.language : "en"
}

export default i18n
