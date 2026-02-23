import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const DLSITE_PATH =
  "M5 15h16c14 0 24 10 24 22.5S35 60 21 60H5Zm11.5 10v25h4c7 0 12-5 12-12.5S27.5 25 20.5 25ZM49 15v45h32v-10h-21v-35zM110 33c-3-5-6-6-10-6-8 0-13 4-13 11 0 8 19 7 19 14 0 5-5 9-11 9-5 0-10-3-12-7l6-6c2 3 4 4 6 4 3 0 5-2 5-5 0-5-6-6-6-12 0-3 3-4 6-4 3 0 5 2 6 5zM112 27h11v33h-11zM130 18v9h-5v10h5v14c0 6 3 9 9 9h7v-10h-4c-2 0-3-1-3-3v-10h7v-10h-7v-9zM148 44c0-11 8-17 19-17c11 0 19 6 19 16v4h-26c1 6 6 9 11 9c5 0 9-2 11-5l6 6c-4 4-10 5-17 5c-12 0-23-6-23-18zm12-4h21c-1-4-4-7-10-7-6 0-10 3-11 7z"

export const SXDlsite: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 190 75",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: DLSITE_PATH,
      fill: "#0b3687",
    }),
    createElement("circle", {
      cx: 117.5,
      cy: 16,
      r: 6,
      fill: "#ffb800",
    }),
  )
