import { useCallback, useEffect, useRef, useState } from "react"

import { api } from "@/lib/api"
import type { Category } from "@/types"
import type { JevCategorySuggestion, JevSettings } from "@/types/jev"

interface Options {
  open: boolean
  isEditing: boolean
  name: string
  url: string
  categories: Category[]
  onSuggestion: (categoryId: string) => void
}

export function useJevCategory({ open, isEditing, name, url, categories, onSuggestion }: Options) {
  const [enabled, setEnabled] = useState(false)
  const manual = useRef(false)
  const hasAutofilled = useRef(false)
  const pending = useRef<AbortController | null>(null)
  // Depend on category content, not the array identity of a parent rerender.
  const categorySignature = JSON.stringify(categories.map(({ id, name }) => [id, name]))

  const stop = useCallback(() => {
    manual.current = true
    pending.current?.abort()
  }, [])

  useEffect(() => {
    manual.current = false
    hasAutofilled.current = false
    if (!open || isEditing) return
    const controller = new AbortController()
    api.get<JevSettings>("/jev/settings", { signal: controller.signal }).then((settings) => {
      if (!controller.signal.aborted) setEnabled(settings.enabled && settings.api_key_configured)
    }).catch(() => {
      if (!controller.signal.aborted) setEnabled(false)
    })
    return () => {
      controller.abort()
      pending.current?.abort()
      setEnabled(false)
    }
  }, [open, isEditing])

  useEffect(() => {
    if (!open || isEditing || !enabled || manual.current) return
    const controller = new AbortController()
    pending.current = controller
    // An automatic category belongs to its input. Discard it when the name,
    // domain or available categories change, but never clear a manual choice.
    if (hasAutofilled.current) {
      hasAutofilled.current = false
      onSuggestion("")
    }
    if (!name.trim() || categorySignature === "[]") return
    const timer = setTimeout(() => {
      if (controller.signal.aborted || manual.current) return
      void api.post<JevCategorySuggestion>("/jev/category-suggestion", { name: name.trim(), url }, {
        signal: controller.signal,
        errorHandling: "silent",
      }).then((suggestion) => {
        if (controller.signal.aborted || manual.current || suggestion.category_id === null) return
        const ids: Array<[number, string]> = JSON.parse(categorySignature)
        if (!ids.some(([id]) => id === suggestion.category_id)) return
        hasAutofilled.current = true
        onSuggestion(String(suggestion.category_id))
      }).catch(() => { /* Optional suggestions must never interrupt entry. */ })
    }, 600)
    return () => {
      clearTimeout(timer)
      controller.abort()
    }
  }, [open, isEditing, enabled, name, url, categorySignature, onSuggestion])

  return stop
}
