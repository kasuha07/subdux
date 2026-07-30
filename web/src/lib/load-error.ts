import { isBackendAPIError } from "@/lib/api"

export interface LoadErrorCopy {
  description: string
  retry: string
  title: string
}

type LoadErrorCategory = "default" | "exchangeRate"
type Translate = (key: string) => string

const backendErrorCategories: Readonly<Record<string, LoadErrorCategory>> = {
  exchange_rate_unavailable: "exchangeRate",
}

function classifyLoadError(error: unknown): LoadErrorCategory {
  if (!isBackendAPIError(error) || !error.code) {
    return "default"
  }

  return backendErrorCategories[error.code] ?? "default"
}

export function buildLoadErrorCopy(
  error: unknown,
  t: Translate,
  translationNamespace: string
): LoadErrorCopy {
  const category = classifyLoadError(error)
  const titleKey = category === "default" ? "title" : `${category}Title`
  const descriptionKey =
    category === "default" ? "description" : `${category}Description`

  return {
    title: t(`${translationNamespace}.error.${titleKey}`),
    description: t(`${translationNamespace}.error.${descriptionKey}`),
    retry: t(`${translationNamespace}.error.retry`),
  }
}
