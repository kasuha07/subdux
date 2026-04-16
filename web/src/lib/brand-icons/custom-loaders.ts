import type { SvgIconComponent } from "./types"

const customIconModules = import.meta.glob("./custom/*.ts") as Record<
  string,
  () => Promise<Record<string, unknown>>
>

export function loadCustomSvgIcon(
  moduleName: string,
  exportName: string
): () => Promise<SvgIconComponent> {
  return async () => {
    const modulePath = `./custom/${moduleName}.ts`
    const loadModule = customIconModules[modulePath]

    if (!loadModule) {
      throw new Error(`custom brand icon module not found: ${moduleName}`)
    }

    const loadedModule = await loadModule()
    const loadedIcon = loadedModule[exportName]

    if (typeof loadedIcon !== "function") {
      throw new Error(`custom brand icon export not found: ${moduleName}.${exportName}`)
    }

    return loadedIcon as SvgIconComponent
  }
}
