import { bankBrandSpecs } from "./specs/banks"
import { coreBrandSpecs } from "./specs/core"
import { entertainmentBrandSpecs } from "./specs/entertainment"
import { serviceBrandSpecs } from "./specs/services"

import type { BrandIconSpec } from "./types"

export const brandSpecs: BrandIconSpec[] = [
  ...coreBrandSpecs,
  ...serviceBrandSpecs,
  ...entertainmentBrandSpecs,
  ...bankBrandSpecs,
]
