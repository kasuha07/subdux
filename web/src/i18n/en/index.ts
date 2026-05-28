import actions from "./actions"
import admin from "./admin"
import calendar from "./calendar"
import auth from "./auth"
import common from "./common"
import dashboard from "./dashboard"
import presets from "./presets"
import reports from "./reports"
import settings from "./settings"
import subscription from "./subscription"

const locale = {
  actions,
  admin,
  calendar,
  auth,
  common,
  dashboard,
  presets,
  reports,
  settings,
  subscription,
} as const

export default locale
