import { useSyncExternalStore } from "react"

const HOVER_CAPABLE_POINTER_QUERY = "(hover: hover) and (pointer: fine)"

function getHoverCapablePointer(): boolean {
  if (typeof window === "undefined" || !window.matchMedia) {
    return false
  }

  return window.matchMedia(HOVER_CAPABLE_POINTER_QUERY).matches
}

function subscribeHoverCapablePointer(onStoreChange: () => void): () => void {
  if (typeof window === "undefined" || !window.matchMedia) {
    return () => {}
  }

  const media = window.matchMedia(HOVER_CAPABLE_POINTER_QUERY)
  media.addEventListener("change", onStoreChange)
  return () => media.removeEventListener("change", onStoreChange)
}

export function useHoverCapablePointer(): boolean {
  return useSyncExternalStore(subscribeHoverCapablePointer, getHoverCapablePointer, () => false)
}
