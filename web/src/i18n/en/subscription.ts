const subscription = {
  "card": {
    "billingType": {
      "legacy": "Legacy purchase"
    },
    "recurrence": {
      "interval": {
        "day": "Every {{count}} day(s)",
        "week": "Every {{count}} week(s)",
        "month": "Every {{count}} month(s)",
        "year": "Every {{count}} year(s)"
      },
      "monthlyCost": "month",
      "monthlyDate": "Monthly on day {{day}}",
      "yearlyDate": "Yearly on {{month}}/{{day}}"
    },
    "dueIn": "Due in {{count}}d",
    "dueToday": "Due today",
    "overdue": "Overdue",
    "noNextBilling": "No next billing date",
    "holdingCost": "{{amount}}/day",
    "reminder": {
      "on": "Reminder {{days}}d before",
      "off": "Reminder off"
    },
    "endsOn": "Ends on {{date}}",
    "endedOn": "Ended on {{date}}",
    "notes": "Note: {{content}}",
    "cycleProgressAria": "Current cycle progress: {{progress}}%",
    "status": {
      "active": "active",
      "ended": "ended"
    },
    "renewalMode": {
      "auto_renew": "Auto renew",
      "manual_renew": "Manual renew",
      "cancel_at_period_end": "End at period end"
    }
  },
  "detail": {
    "open": "Details",
    "edit": "Edit",
    "titleFallback": "Subscription details",
    "description": "Subscription detail drawer with history, notifications, and upcoming charges.",
    "error": "Failed to load details",
    "errorTitle": "Details unavailable",
    "tabs": {
      "timeline": "Timeline",
      "prices": "Prices",
      "notifications": "Logs",
      "charges": "Charges"
    },
    "summary": {
      "nextCharge": "Next charge",
      "periodEnd": "Period ends",
      "endingAtPeriodEnd": "Ends at period end",
      "lifecycle": "Lifecycle",
      "latestActivity": "Latest activity",
      "lastNotification": "Last via {{channel}}"
    },
    "info": {
      "title": "Subscription information",
      "amount": "Amount",
      "billingType": "Billing type",
      "recurrence": "Recurrence",
      "nextBillingDate": "Next billing date",
      "periodEndDate": "Period end date",
      "status": "Lifecycle",
      "renewalMode": "Renewal mode",
      "endsAt": "Ended on",
      "category": "Category",
      "paymentMethod": "Payment method",
      "notification": "Notification",
      "notificationDefault": "Use default policy",
      "notificationEnabled": "Enabled",
      "notificationEnabledWithDays": "Enabled, {{days}} day(s) before",
      "notificationDisabled": "Disabled",
      "createdAt": "Created",
      "updatedAt": "Updated",
      "url": "URL",
      "notes": "Notes"
    },
    "empty": {
      "none": "None",
      "noUpcomingCharges": "No upcoming charge",
      "noNotifications": "No notifications"
    },
    "timeline": {
      "emptyTitle": "No timeline yet",
      "emptyDescription": "Changes to this subscription will appear here."
    },
    "prices": {
      "emptyTitle": "No price history",
      "emptyDescription": "Price changes will appear here after edits.",
      "from": "from {{amount}}"
    },
    "notifications": {
      "emptyTitle": "No notification logs",
      "emptyDescription": "Recent notifications for this subscription will appear here.",
      "statusSent": "Sent",
      "statusFailed": "Failed"
    },
    "charges": {
      "emptyTitle": "No upcoming charges",
      "emptyDescription": "This subscription has no future billing dates."
    },
    "calendar": {
      "title": "Calendar",
      "next": "Next calendar event: {{date}}",
      "noEvent": "No upcoming calendar event",
      "open": "Open calendar"
    }
  },
  "form": {
    "editTitle": "Edit subscription",
    "addTitle": "Add subscription",
    "editDescription": "Review the subscription details below and save your changes.",
    "addDescription": "Fill in the subscription details below to add it to your tracker.",
    "nameLabel": "Name",
    "namePlaceholder": "Netflix, Spotify...",
    "amountLabel": "Amount",
    "amountPlaceholder": "9.99",
    "currencyLabel": "Currency",
    "billingTypeLabel": "Billing type",
    "billingType": {
      "recurring": "Subscription-based",
      "one_time": "Legacy purchase (unsupported)"
    },
    "statusLabel": "Lifecycle",
    "status": {
      "active": "Active",
      "ended": "Ended"
    },
    "renewalModeLabel": "Renewal mode",
    "renewalMode": {
      "auto_renew": "Auto renew",
      "manual_renew": "Manual renew",
      "cancel_at_period_end": "End at period end"
    },
    "endsAtLabel": "Ended on",
    "purchaseDateLabel": "Purchase date",
    "nextBillingDateLabel": "Next billing date",
    "periodEndDateLabel": "Period end date",
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
      "suggestions": {
        "title": "Suggested icons for {{domain}}",
        "google": "Google favicon",
        "iconHorse": "icon.horse"
      },
      "uploadLabel": "Upload image",
      "uploadHint": "PNG, JPG, or ICO, max {{size}}KB",
      "uploadButton": "Choose file",
      "invalidType": "Only valid PNG/JPG/ICO files with matching extension are allowed",
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
