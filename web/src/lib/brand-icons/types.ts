import type { ComponentType, SVGProps } from "react"

export type SvgIconComponent = ComponentType<SVGProps<SVGSVGElement>>
export type IconValuePrefix = "bl" | "lg" | "custom"

export type BrandIconComponent = ComponentType<{ size?: number | string; color?: string; className?: string }>

export interface BrandIconSpec {
  prefix: IconValuePrefix
  slug: string
  title: string
  hex: string
  keywords?: string[]
  loadIcon: () => Promise<SvgIconComponent>
}

export interface BrandIconRuntime {
  slug: string
  value: string
  Icon: BrandIconComponent
}

export interface BrandIcon extends BrandIconRuntime {
  title: string
  hex: string
  keywords: string[]
}
