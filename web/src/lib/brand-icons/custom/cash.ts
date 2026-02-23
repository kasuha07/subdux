import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const CASH_BAG_PATH =
  "M7 10.02v1.01m0-6.02v.94m0 7.54c3.5 0 6-1.24 6-4c0-3-1.5-5-4.5-6.5l1.18-1.52a.66.66 0 0 0-.56-1H4.88a.66.66 0 0 0-.56 1L5.5 3C2.5 4.51 1 6.51 1 9.51c0 2.74 2.5 3.98 6 3.98Z"

const CASH_DOLLAR_PATH =
  "M6 9.56A1.24 1.24 0 0 0 7 10a1.12 1.12 0 0 0 1.19-1A1.12 1.12 0 0 0 7 8a1.12 1.12 0 0 1-1.19-1A1.11 1.11 0 0 1 7 6a1.26 1.26 0 0 1 1 .4"

export const SXCash: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 14 14",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: CASH_BAG_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("path", {
      d: CASH_DOLLAR_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
  )
