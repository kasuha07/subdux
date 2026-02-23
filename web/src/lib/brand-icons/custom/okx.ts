import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const OKX_PATH =
  "M30.167 17.833H17.833v12.334h12.334zM42.5 30.167H30.167V42.5H42.5zm0-24.667H30.167v12.333H42.5zM17.833 30.167H5.5V42.5h12.333zm0-24.667H5.5v12.333h12.333z"

export const SXOkx: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 48 48",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: OKX_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
  )
