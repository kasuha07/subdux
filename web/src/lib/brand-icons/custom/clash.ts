import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const CLASH_LEFT_PATH =
  "M27.19 42.5a89 89 0 0 1-14.681-1.572S13.94 12.372 17.92 5.535c-.13-.297 2.992 1.212 4.422 6.266a25.6 25.6 0 0 1 4.847-.47"

const CLASH_RIGHT_PATH =
  "M27.19 42.5a89 89 0 0 0 14.681-1.572S40.44 12.372 36.458 5.535c.03-.2-3.59 1.755-4.421 6.266a25.6 25.6 0 0 0-4.848-.47"

const CLASH_BOTTOM_PATH =
  "M12.508 40.927c-1.93-.327-4.948-.31-6.04-3.487c-1.067-3.107.438-6.67 3.742-7.045m15.253-4.008a1.467 1.467 0 0 0 1.473-1.472"

const CLASH_BOTTOM_RIGHT_PATH = "M28.41 26.387a1.467 1.467 0 0 1-1.474-1.472"

export const SXClash: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 48 48",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: CLASH_LEFT_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("ellipse", {
      cx: 21.24,
      cy: 20.309,
      rx: 1.671,
      ry: 2.13,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("path", {
      d: CLASH_RIGHT_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("ellipse", {
      cx: 33.14,
      cy: 20.309,
      rx: 1.671,
      ry: 2.13,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("path", {
      d: CLASH_BOTTOM_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeMiterlimit: 5.714,
    }),
    createElement("path", {
      d: CLASH_BOTTOM_RIGHT_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeMiterlimit: 5.714,
    }),
  )
