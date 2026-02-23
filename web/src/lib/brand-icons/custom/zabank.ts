import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const ZA_BANK_PATH = "M25 31h28l-19 38m41 0H47l19-38"

export const SXZaBank: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 100 100",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("circle", {
      cx: 50,
      cy: 50,
      r: 50,
      fill: "#0c8",
    }),
    createElement("path", {
      d: ZA_BANK_PATH,
      fill: "none",
      stroke: "#fff",
      strokeWidth: 8,
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
  )
