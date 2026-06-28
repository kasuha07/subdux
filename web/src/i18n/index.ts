import i18n from "i18next"
import type { BackendModule, ReadCallback, ResourceLanguage } from "i18next"
import { initReactI18next } from "react-i18next"
import LanguageDetector from "i18next-browser-languagedetector"

const localeLoaders: Record<string, () => Promise<ResourceLanguage>> = {
  en: () => import("./en").then((module) => module.default),
  "zh-CN": () => import("./zh-CN").then((module) => module.default),
  ja: () => import("./ja").then((module) => module.default),
}

const dynamicLocaleBackend: BackendModule = {
  type: "backend",
  init() {},
  read(language: string, _namespace: string, callback: ReadCallback) {
    const loadLocale = localeLoaders[language] ?? localeLoaders.en

    loadLocale()
      .then((locale) => callback(null, locale))
      .catch((error: unknown) => callback(error instanceof Error ? error : new Error(String(error)), null))
  },
}

export const i18nReady = i18n
  .use(dynamicLocaleBackend)
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    fallbackLng: "en",
    supportedLngs: ["en", "zh-CN", "ja"],
    ns: ["translation"],
    defaultNS: "translation",
    interpolation: {
      escapeValue: false,
    },
    detection: {
      order: ["localStorage", "navigator"],
      caches: ["localStorage"],
      lookupLocalStorage: "language",
    },
  })

export default i18n
