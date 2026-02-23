import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const EMBY_PATH =
  "M469.333 85.333L256 298.667l42.667 42.666L85.333 554.667 298.667 768l42.666-42.667 213.334 213.334L768 725.333l-42.667-42.666 213.334-213.334L725.333 256l-42.666 42.667L469.333 85.333m-42.666 277.334l256 149.333-256 149.333V362.667z"

export const SXEmby: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 1024 1024",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: EMBY_PATH,
      fill: "#06B831",
    }),
  )
