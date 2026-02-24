import admin from "./admin"
import calendar from "./calendar"
import auth from "./auth"
import common from "./common"
import dashboard from "./dashboard"
import presets from "./presets"
import settings from "./settings"
import subscription from "./subscription"

const locale = {
  admin,
  calendar,
  auth,
  common,
  dashboard,
  presets,
  settings,
  subscription,
} as const

export default locale
