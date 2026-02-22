export const CHANNEL_TYPES = [
  "smtp",
  "resend",
  "telegram",
  "webhook",
  "gotify",
  "ntfy",
  "bark",
  "serverchan",
  "feishu",
  "wecom",
  "dingtalk",
  "pushdeer",
  "pushplus",
  "pushover",
  "napcat",
]

export const TEMPLATE_FORMATS = ["plaintext", "markdown", "html"] as const

export const TEMPLATE_VARIABLES = [
  { name: "{{.SubscriptionName}}", key: "varSubscriptionName" },
  { name: "{{.BillingDate}}", key: "varBillingDate" },
  { name: "{{.Amount}}", key: "varAmount" },
  { name: "{{.Currency}}", key: "varCurrency" },
  { name: "{{.DaysUntil}}", key: "varDaysUntil" },
  { name: "{{.Category}}", key: "varCategory" },
  { name: "{{.PaymentMethod}}", key: "varPaymentMethod" },
  { name: "{{.URL}}", key: "varURL" },
  { name: "{{.Remark}}", key: "varRemark" },
  { name: "{{.UserEmail}}", key: "varUserEmail" },
] as const
