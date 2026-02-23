import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const SIMCARD_OUTLINE_PATH =
  "M384 40H230.627A31.791 31.791 0 0 0 208 49.373L97.373 160A31.791 31.791 0 0 0 88 182.627V448a32.036 32.036 0 0 0 32 32h264a32.036 32.036 0 0 0 32-32V72a32.036 32.036 0 0 0-32-32m0 408H120V182.627L230.627 72H384Z"

const SIMCARD_CORE_PATH = "M208 416h144V216H208Zm32-168h80v136h-80Z"

export const SXSimCard: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 512 512",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: SIMCARD_OUTLINE_PATH,
      fill: "currentColor",
    }),
    createElement("path", {
      d: SIMCARD_CORE_PATH,
      fill: "currentColor",
    }),
  )
