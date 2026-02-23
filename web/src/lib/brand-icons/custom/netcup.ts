import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const NETCUP_PATH =
  "M5.25 0A5.24 5.24 0 0 0 0 5.25v13.5A5.24 5.24 0 0 0 5.25 24h13.5A5.24 5.24 0 0 0 24 18.75V5.25A5.24 5.24 0 0 0 18.75 0zm-.045 5.102h9.482c1.745 0 2.631.907 2.631 2.753v8.352h1.477v2.691h-4.666V8.58c0-.514-.298-.785-.889-.785H9.873v11.103H6.682V7.795H5.205z"

export const SXNetcup: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 24 24",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: NETCUP_PATH,
      fill: "#00646E",
    }),
  )
