import admin from "./admin"
import auth from "./auth"
import common from "./common"
import dashboard from "./dashboard"
import presets from "./presets"
import settings from "./settings"
import subscription from "./subscription"

const locale = {
  admin,
  auth,
  common,
  dashboard,
  presets,
  settings,
  subscription,
} as const

export default locale
