import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const GEMINI_PATH =
  "M960 512.896A477.248 477.248 0 0 0 512.896 960h-1.792A477.184 477.184 0 0 0 64 512.896v-1.792A477.184 477.184 0 0 0 511.104 64h1.792A477.248 477.248 0 0 0 960 511.104z"

export const SXGemini: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 1024 1024",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: GEMINI_PATH,
      fill: "#448AFF",
    }),
  )
