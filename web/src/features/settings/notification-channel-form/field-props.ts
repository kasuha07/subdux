import type { TFunction } from "i18next"

import type { NotificationChannelFormValues } from "./types"

export interface BaseChannelConfigFieldProps {
  t: TFunction
  values: NotificationChannelFormValues
  isSecretFieldConfigured: (field: keyof NotificationChannelFormValues) => boolean
  onValueChange: <K extends keyof NotificationChannelFormValues>(
    key: K,
    value: NotificationChannelFormValues[K]
  ) => void
}
