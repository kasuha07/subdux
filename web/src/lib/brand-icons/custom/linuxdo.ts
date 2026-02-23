import { createElement, useId } from "react"

import type { SvgIconComponent } from "../types"

export const SXLinuxDo: SvgIconComponent = (props) => {
  const clipId = `linuxdo-${useId().replaceAll(":", "")}`

  return createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 120 120",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement(
      "defs",
      undefined,
      createElement(
        "clipPath",
        { id: clipId },
        createElement("circle", { cx: 60, cy: 60, r: 47 }),
      ),
    ),
    createElement("circle", {
      cx: 60,
      cy: 60,
      r: 50,
      fill: "#F0F0F0",
    }),
    createElement("rect", {
      x: 10,
      y: 10,
      width: 100,
      height: 30,
      fill: "#1C1C1E",
      clipPath: `url(#${clipId})`,
    }),
    createElement("rect", {
      x: 10,
      y: 40,
      width: 100,
      height: 40,
      fill: "#F0F0F0",
      clipPath: `url(#${clipId})`,
    }),
    createElement("rect", {
      x: 10,
      y: 80,
      width: 100,
      height: 30,
      fill: "#FFB003",
      clipPath: `url(#${clipId})`,
    }),
  )
}
