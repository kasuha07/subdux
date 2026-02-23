import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const BITWARDEN_SHIELD_PATH =
  "M372 297V131H256v294c47-28 115-74 116-128zm49-198v198c0 106-152 181-165 181S91 403 91 297V99s0-17 17-17h296s17 0 17 17z"

export const SXBitwarden: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 512 512",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("rect", {
      width: 512,
      height: 512,
      rx: "15%",
      fill: "#175DDC",
    }),
    createElement("path", {
      d: BITWARDEN_SHIELD_PATH,
      fill: "#fff",
    }),
  )
