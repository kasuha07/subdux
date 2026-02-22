const subscription = {
  "card": {
    "billingType": {
      "one_time": "Buyout"
    },
    "recurrence": {
      "interval": {
        "day": "Every {{count}} day(s)",
        "week": "Every {{count}} week(s)",
        "month": "Every {{count}} month(s)",
        "year": "Every {{count}} year(s)"
      },
      "monthlyDate": "Monthly on day {{day}}",
      "yearlyDate": "Yearly on {{month}}/{{day}}"
    },
    "dueIn": "Due in {{count}}d",
    "overdue": "Overdue",
    "noNextBilling": "No next billing date",
    "trial": {
      "startsIn": "Trial starts in {{count}}d",
      "active": "Trial active",
      "endsIn": "Trial ends in {{count}}d",
      "endedOn": "Trial ended on {{date}}"
    },
    "reminder": {
      "on": "Reminder {{days}}d before",
      "off": "Reminder off"
    },
    "anchorDate": "Anchor {{date}}",
    "notes": "Note: {{content}}",
    "status": {
      "enabled": "enabled",
      "disabled": "disabled"
    }
  },
  "form": {
    "editTitle": "Edit subscription",
    "addTitle": "Add subscription",
    "nameLabel": "Name",
    "namePlaceholder": "Netflix, Spotify...",
    "amountLabel": "Amount",
    "amountPlaceholder": "9.99",
    "currencyLabel": "Currency",
    "billingTypeLabel": "Billing type",
    "billingType": {
      "recurring": "Subscription-based",
      "one_time": "Buyout"
    },
    "enabledLabel": "Enabled",
    "enabled": "Enabled",
    "disabled": "Disabled",
    "purchaseDateLabel": "Purchase date",
    "anchorDateLabel": "Billing anchor date",
    "recurrenceTypeLabel": "Recurrence rule",
    "recurrenceDetailLabel": "Rule details",
    "recurrenceType": {
      "interval": "Custom",
      "monthly_date": "Specific day each month",
      "yearly_date": "Specific date each year"
    },
    "intervalCountLabel": "Every N",
    "intervalUnitLabel": "Unit",
    "intervalUnit": {
      "day": "Day",
      "week": "Week",
      "month": "Month",
      "year": "Year"
    },
    "monthlyDayLabel": "Day of month",
    "yearlyMonthLabel": "Month",
    "yearlyDayLabel": "Day",
    "trialLabel": "Trial period",
    "trialStartLabel": "Trial start",
    "trialEndLabel": "Trial end",
    "categoryLabel": "Category",
    "categoryPlaceholder": "Select...",
    "paymentMethodLabel": "Payment method",
    "paymentMethodPlaceholder": "Select...",
    "noPaymentMethod": "No payment method",
    "categories": {
      "entertainment": "Entertainment",
      "productivity": "Productivity",
      "development": "Development",
      "music": "Music",
      "cloud": "Cloud",
      "finance": "Finance",
      "health": "Health",
      "education": "Education",
      "news": "News",
      "other": "Other"
    },
    "iconLabel": "Icon / Emoji",
    "iconPlaceholder": "🎬",
    "iconPicker": {
      "label": "Icon",
      "tabs": {
        "emoji": "Emoji",
        "brand": "Brand",
        "image": "Image"
      },
      "emojiPlaceholder": "🎬",
      "searchPlaceholder": "Search brand icons...",
      "noResults": "No icons found",
      "urlLabel": "Image URL",
      "urlPlaceholder": "https://example.com/logo.png",
      "uploadLabel": "Upload image",
      "uploadHint": "PNG or JPG, max {{size}}KB",
      "uploadButton": "Choose file",
      "invalidType": "Only PNG and JPG images are allowed",
      "fileTooLarge": "File exceeds {{size}}KB limit",
      "removeFile": "Remove"
    },
    "iconUploadFailed": "Icon upload failed, but subscription was saved",
    "urlLabel": "URL",
    "urlPlaceholder": "https://...",
    "notesLabel": "Notes",
    "notesPlaceholder": "Optional notes...",
    "cancel": "Cancel",
    "saving": "Saving...",
    "update": "Update",
    "addButton": "Add subscription",
    "error": "Failed to save"
  }
} as const

export default subscription
