import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const MICROSOFT365_PATH_1 =
  "M20.507 3.241L8.443 9.74a8.6 8.6 0 0 0-4.508 7.563V30.7a8.6 8.6 0 0 0 4.508 7.564m15.668-5.597l-2.826-1.557a8.6 8.6 0 0 1-4.45-7.532v-4.072"

const MICROSOFT365_PATH_2 =
  "M31.166 19.275v4.45a8.6 8.6 0 0 1-4.508 7.564l-11.466 6.202a8.6 8.6 0 0 1-7.435.358q.33.222.687.414l11.465 6.202a8.6 8.6 0 0 0 8.182 0l11.465-6.202a8.6 8.6 0 0 0 4.508-7.563"

const MICROSOFT365_PATH_3 =
  "M39.557 9.739L28.092 3.536a8.6 8.6 0 0 0-7.585-.295a8.6 8.6 0 0 0-3.673 7.048v8.986l3.288-1.661a8.6 8.6 0 0 1 7.756 0l11.465 5.793a8.6 8.6 0 0 1 4.72 7.484l.001-.191V17.302a8.6 8.6 0 0 0-4.507-7.563"

export const SXMicrosoft365: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 48 48",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: MICROSOFT365_PATH_1,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("path", {
      d: MICROSOFT365_PATH_2,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("path", {
      d: MICROSOFT365_PATH_3,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
  )
