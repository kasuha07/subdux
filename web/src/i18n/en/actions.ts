const actions = {
  "title": "Action center",
  "nav": {
    "back": "Back to dashboard",
    "refresh": "Refresh actions"
  },
  "toast": {
    "markRenewed": "Marked as renewed",
    "cancelAtPeriodEnd": "Set to end at period end",
    "keepSubscription": "Subscription will keep renewing",
    "snoozed": "Reminder snoozed for 7 days"
  },
  "error": {
    "title": "Action center unavailable",
    "description": "Failed to load action center",
    "actionFailed": "Action failed",
    "missingNextBilling": "Set a next billing date before canceling at period end",
    "subscriptionMissing": "Subscription no longer exists"
  },
  "summary": {
    "total": "To handle",
    "snoozed": "{{count}} snoozed",
    "grouped": "{{actions}} actions · {{snoozed}} snoozed",
    "upcoming": "Upcoming charges",
    "window": "{{urgent}} / {{days}} day window",
    "repair": "Needs repair",
    "decision": "{{count}} need a decision"
  },
  "empty": {
    "title": "Nothing needs attention",
    "description": "Upcoming renewals, failed reminders, and data fixes will appear here."
  },
  "type": {
    "upcoming_renewal": "Upcoming charge",
    "manual_renewal_due": "Manual renewal",
    "ending_soon": "Ending soon",
    "notification_failed": "Notification failed",
    "missing_next_billing": "Missing billing date",
    "price_increase": "Price increase"
  },
  "severity": {
    "critical": "Critical",
    "high": "High",
    "medium": "Medium",
    "low": "Low"
  },
  "message": {
    "upcoming_renewal": "Review before the next automatic charge.",
    "manual_renewal_due": "Confirm payment after renewing manually.",
    "ending_soon": "This subscription is scheduled to end soon.",
    "notification_failed": "A recent reminder could not be delivered.",
    "missing_next_billing": "Add a next billing date to restore reminders and reports.",
    "price_increase": "Review whether the new price is still worth keeping."
  },
  "priceDelta": "+{{amount}}/mo",
  "group": {
    "itemCount": "{{count}} actions"
  },
  "command": {
    "markRenewed": "Mark renewed",
    "keepSubscription": "Keep subscription",
    "cancelAtPeriodEnd": "Cancel later",
    "snooze": "Snooze"
  },
  "date": {
    "today": "Today",
    "overdue": "{{count}}d overdue",
    "inDays": "In {{count}}d · {{date}}",
    "event": "Changed {{date}}",
    "none": "No date"
  }
} as const

export default actions
