import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const ADOBE_CREATIVE_CLOUD_PATH =
  "M15.1 2H24v20L15.1 2zM8.9 2H0v20L8.9 2zM12 9.4 17.6 22h-3.8l-1.6-4H8.1L12 9.4z"

export const SXAdobeCreativeCloud: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "3 2 18 20",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: ADOBE_CREATIVE_CLOUD_PATH,
      fill: "red",
    }),
  )
