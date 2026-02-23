import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const SAMSCLUB_PATH =
  "M512 364.8l-140.8 144 140.8 153.6 140.8-153.6-140.8-144z m41.6-44.8l182.4 188.8-182.4 195.2-41.6-44.8-41.6 44.8L288 508.8l182.4-188.8 41.6 44.8 41.6-44.8zM800 192H224c-19.2 0-32 12.8-32 32v576c0 19.2 12.8 32 32 32h576c19.2 0 32-12.8 32-32V224c0-19.2-12.8-32-32-32z m0-64c54.4 0 96 41.6 96 96v576c0 54.4-41.6 96-96 96H224c-54.4 0-96-41.6-96-96V224c0-54.4 41.6-96 96-96h576z"

export const SXSamsClub: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 1024 1024",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", {
      d: SAMSCLUB_PATH,
      fill: "#5F6165",
    }),
  )
