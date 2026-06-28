export interface PreviewCurrencyChange {
  code: string
  symbol: string
  is_new: boolean
}

export interface PreviewPaymentMethodChange {
  name: string
  is_new: boolean
  matched?: string
}

export interface PreviewCategoryChange {
  name: string
  is_new: boolean
}

export interface PreviewSubscriptionChange {
  name: string
  amount: number
  currency: string
  billing_type: string
  category?: string
  skipped: boolean
  skip_reason?: string
}

export interface ImportPreview {
  currencies: PreviewCurrencyChange[]
  payment_methods: PreviewPaymentMethodChange[]
  categories: PreviewCategoryChange[]
  subscriptions: PreviewSubscriptionChange[]
}

export interface SubduxPreviewChannelChange {
  type: string
  is_new: boolean
  config: string
}

export interface SubduxPreviewTemplateChange {
  channel_type: string
  format: string
  is_new: boolean
}

export interface SubduxPreviewPreferenceChange {
  will_create: boolean
  will_update: boolean
  current: string
  incoming: string
}

export interface SubduxPreviewPolicyChange {
  will_create: boolean
  will_update: boolean
  current_days_before: number
  incoming_days_before: number
  current_notify_on_due_day: boolean
  incoming_notify_on_due_day: boolean
  current_notify_manual_renew_daily: boolean
  incoming_notify_manual_renew_daily: boolean
}

export interface SubduxImportPreview {
  currencies: PreviewCurrencyChange[]
  payment_methods: PreviewPaymentMethodChange[]
  categories: PreviewCategoryChange[]
  subscriptions: PreviewSubscriptionChange[]
  channels: SubduxPreviewChannelChange[]
  templates: SubduxPreviewTemplateChange[]
  preference?: SubduxPreviewPreferenceChange
  policy?: SubduxPreviewPolicyChange
}
