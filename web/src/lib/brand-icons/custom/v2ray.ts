import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const V2RAY_PATH =
  "M9.619 41.487V10.614H5.5V6.542h10.264v14.394c.696-.64 1.287-1.18 1.873-1.725L28.18 9.385c.942-.878 1.877-1.763 2.83-2.627c.144-.13.371-.238.56-.239c3.567-.013 10.93 0 10.93 0C31.545 18.156 20.628 29.793 9.619 41.488"

export const SXV2ray: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 48 48",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: V2RAY_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
  )
