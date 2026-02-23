import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const NETEASE_UU_PATH_1 =
  "M571.567985 168.76155v89.937365a210.435969 210.435969 0 1 1-127.007752 2.540155V170.587287a297.436279 297.436279 0 1 0 127.007752-1.825737z"

const NETEASE_UU_PATH_2 =
  "M571.567985 0v88.111628a377.292403 377.292403 0 1 1-127.007752 1.428837V1.031938A462.070078 462.070078 0 0 0 425.50907 912.868217c28.021085 5.23907 56.994729 111.131783 86.524031 111.131783 27.068527 0 53.581395-105.416434 79.379845-109.861705A462.070078 462.070078 0 0 0 571.567985 0z"

export const SXNeteaseUu: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 1024 1024",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", { d: NETEASE_UU_PATH_1, fill: "#36ECD4" }),
    createElement("path", { d: NETEASE_UU_PATH_2, fill: "#36ECD4" }),
  )
