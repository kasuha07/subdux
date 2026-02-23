import { createElement } from "react"

import type { SvgIconComponent } from "../types"

const INFUSE_PATH_1 =
  "M8.509 14.279V7.01c-1.038-.7-2.32-.7-3.465 1.313l-.039 5.956c-.129 1.736 3.061 2.56 3.504 0"

const INFUSE_PATH_2 =
  "M18.91 11c-.405-1.709-1.148-1.733-2.954-3.182c0 0 1.6 2.732-.526 3.968c.039-.023-3.968 2.931-3.968 2.931L7.369 17.7a1.742 1.742 0 0 0 1.947 2.849l7.984-5.581a3.858 3.858 0 0 0 1.3-1.489A3.809 3.809 0 0 0 18.91 11"

const INFUSE_PATH_3 =
  "M8.509 6.8c-.034-.031 4.022 2.877 4.022 2.877l2.662 1.959c1.3-.517 1.427-2.127.494-4.028L10 3.575c-.882-.456-5.277-1.25-4.987 4.58c.001 0 1.153-2.869 3.496-1.355"

export const SXInfuse: SvgIconComponent = (props) =>
  createElement(
    "svg",
    {
      ...props,
      viewBox: "0 0 24 24",
      xmlns: "http://www.w3.org/2000/svg",
    },
    createElement("path", { d: INFUSE_PATH_1, fill: "currentColor" }),
    createElement("path", { d: INFUSE_PATH_2, fill: "currentColor" }),
    createElement("path", { d: INFUSE_PATH_3, fill: "currentColor" }),
  )
