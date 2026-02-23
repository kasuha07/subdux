import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const GIFFGAFF_DOT_PATH = "M24 22.64v-8.58a1.82 1.82 0 0 1 1.83-1.82h0a2.09 2.09 0 0 1 1.83.76"
const GIFFGAFF_LINE_PATH = "M20.45 15.73h5.8"
const GIFFGAFF_SMALL_PATH = "M22.86 12.22a4 4 0 0 0-.59 0h0a1.81 1.81 0 0 0-1.82 1.82v8.58h-2.88v-6.93H14.7v7.8a2.59 2.59 0 0 1-2.6 2.6h0a2.6 2.6 0 0 1-1.84-.76"
const GIFFGAFF_BOTTOM_PATH = "M34.84 32.84v-8.59a1.82 1.82 0 0 1 1.82-1.82h0a2.13 2.13 0 0 1 1.84.76"
const GIFFGAFF_BOTTOM_LINE_PATH = "M31.25 25.94h5.81"
const GIFFGAFF_MID_PATH = "M20.45 25.94v7.81a2.61 2.61 0 0 1-2.6 2.6h0a2.6 2.6 0 0 1-1.85-.76m12.38-5.35a2.6 2.6 0 0 1-2.6 2.6h0a2.61 2.61 0 0 1-2.6-2.6v-1.69a2.61 2.61 0 0 1 2.6-2.61h0a2.6 2.6 0 0 1 2.6 2.61"
const GIFFGAFF_MID2_PATH = "M33.66 22.48a3.4 3.4 0 0 0-.59-.05h0a1.83 1.83 0 0 0-1.82 1.82v8.59h-2.87v-6.9"
const GIFFGAFF_OUTER_PATH = "M40.5 5.5h-33a2 2 0 0 0-2 2v33a2 2 0 0 0 2 2h33a2 2 0 0 0 2-2v-33a2 2 0 0 0-2-2"

export const SXGiffgaff: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 48 48",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("circle", {
      cx: 17.57,
      cy: 12.56,
      r: 0.75,
      fill: "currentColor",
    }),
    createElement("path", {
      d: GIFFGAFF_DOT_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("path", {
      d: GIFFGAFF_LINE_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("path", {
      d: GIFFGAFF_SMALL_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("rect", {
      x: 9.5,
      y: 15.75,
      width: 5.2,
      height: 6.89,
      rx: 2.6,
      transform: "rotate(180 12.1 19.195)",
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("rect", {
      x: 15.25,
      y: 25.94,
      width: 5.2,
      height: 6.89,
      rx: 2.6,
      transform: "rotate(-180 17.845 29.39)",
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("path", {
      d: GIFFGAFF_BOTTOM_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("path", {
      d: GIFFGAFF_BOTTOM_LINE_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("path", {
      d: GIFFGAFF_MID_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("path", {
      d: GIFFGAFF_MID2_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
    createElement("path", {
      d: GIFFGAFF_OUTER_PATH,
      fill: "none",
      stroke: "currentColor",
      strokeLinecap: "round",
      strokeLinejoin: "round",
    }),
  )
