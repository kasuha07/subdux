const reports = {
  "title": "Reports",
  "error": {
    "title": "Report unavailable",
    "description": "Refresh the page or try again later."
  },
  "empty": {
    "title": "No report data",
    "description": "Add active subscriptions to see analytics."
  },
  "kpis": {
    "monthly": "Monthly spend",
    "yearlyDetail": "{{amount}} yearly",
    "committed": "Auto-renew spend",
    "autoRenewDetail": "{{count}} auto-renew subscriptions",
    "next30Days": "Next 30 days",
    "renewalDetail": "{{count}} renewals",
    "active": "Active subscriptions",
    "modeDetail": "{{auto}} auto / {{manual}} manual / {{canceling}} ending"
  },
  "yearlyStats": {
    "title": "Yearly stats",
    "totalYearly": "Yearly spend",
    "committedYearly": "Auto-renew yearly spend",
    "forecast12Months": "12-month forecast spend",
    "forecastPayments": "12-month payments",
    "monthlyAverage": "{{amount}} monthly average",
    "autoRenewCount": "{{count}} auto-renew subscriptions",
    "forecastMonths": "Covers {{count}} months",
    "paymentDetail": "Forecast payment count"
  },
  "forecast": {
    "title": "12-month billing forecast",
    "occurrences": "{{count}} payments"
  },
  "categories": {
    "title": "Category breakdown",
    "none": "Uncategorized"
  },
  "paymentMethods": {
    "title": "Payment method breakdown",
    "none": "No payment method"
  },
  "renewalModes": {
    "title": "Renewal mode breakdown",
    "auto_renew": "Auto renew",
    "manual_renew": "Manual renew",
    "cancel_at_period_end": "Ends at period end"
  },
  "topSubscriptions": {
    "title": "Highest monthly cost",
    "monthly": "monthly"
  },
  "upcoming": {
    "title": "Upcoming renewals",
    "daysUntil": "in {{count}} days",
    "emptyTitle": "No renewals soon",
    "emptyDescription": "No active subscriptions renew in the next 30 days."
  }
} as const

export default reports
