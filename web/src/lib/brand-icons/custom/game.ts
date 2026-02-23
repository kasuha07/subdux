import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const GAME_OUTLINE_PATH =
  "M4 3h8a4 4 0 0 1 4 4v3a4 4 0 0 1-4 4H4a4 4 0 0 1-4-4V7a4 4 0 0 1 4-4m0 1a3 3 0 0 0-3 3v3a3 3 0 0 0 3 3h8a3 3 0 0 0 3-3V7a3 3 0 0 0-3-3z"

const GAME_BUTTONS_PATH =
  "M5.5 6a.5.5 0 0 0-.5.5V8H3.5a.5.5 0 0 0 0 1H5v1.5a.5.5 0 0 0 1 0V9h1.5a.5.5 0 0 0 0-1H6V6.5a.5.5 0 0 0-.5-.5M13 7a1 1 0 1 1-2 0a1 1 0 0 1 2 0m-1 3a1 1 0 1 1-2 0a1 1 0 0 1 2 0"

export const SXGame: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 16 16",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: GAME_OUTLINE_PATH,
      fill: "currentColor",
      fillRule: "evenodd",
      clipRule: "evenodd",
    }),
    createElement("path", {
      d: GAME_BUTTONS_PATH,
      fill: "currentColor",
    }),
  )
