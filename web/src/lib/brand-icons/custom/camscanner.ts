import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const CAMSCANNER_FRAME_PATH =
  "M11 5a6 6 0 0 0-6 6v26a6 6 0 0 0 6 6h26a6 6 0 0 0 6-6V11a6 6 0 0 0-6-6ZM5 34.53h38"

const CAMSCANNER_MAIN_PATH =
  "M26.81 27A4.75 4.75 0 0 0 31 28.83h2.51a4.24 4.24 0 0 0 4.24-4.24h0a4.24 4.24 0 0 0-4.24-4.25h-2.8a4.25 4.25 0 0 1-4.24-4.25h0a4.24 4.24 0 0 1 4.24-4.25h2.52a4.73 4.73 0 0 1 4.16 1.86m-15.86 9.43v.07a5.63 5.63 0 0 1-5.63 5.63h0a5.63 5.63 0 0 1-5.63-5.63v-5.73a5.63 5.63 0 0 1 5.63-5.63h0a5.63 5.63 0 0 1 5.63 5.63v.07"

export const SXCamscanner: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 48 48",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: CAMSCANNER_FRAME_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("path", {
      d: CAMSCANNER_MAIN_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
  )
