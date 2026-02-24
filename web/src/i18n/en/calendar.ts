const calendar = {
  "title": "Calendar",
  "back": "Back",
  "today": "Today",
  "weekdays": {
    "sun": "Sun",
    "mon": "Mon",
    "tue": "Tue",
    "wed": "Wed",
    "thu": "Thu",
    "fri": "Fri",
    "sat": "Sat"
  },
  "months": {
    "1": "January",
    "2": "February",
    "3": "March",
    "4": "April",
    "5": "May",
    "6": "June",
    "7": "July",
    "8": "August",
    "9": "September",
    "10": "October",
    "11": "November",
    "12": "December"
  },
  "subscriptionsDue": "{{count}} subscription(s) due",
  "noSubscriptions": "No subscriptions due this day",
  "editSuccess": "Subscription updated",
  "token": {
    "title": "Calendar Subscription",
    "description": "Generate a subscription link to sync your billing dates with external calendar apps (Google Calendar, Apple Calendar, Outlook, etc.)",
    "createLink": "Create Link",
    "create": "Create",
    "creating": "Creating...",
    "name": "Link Name",
    "namePlaceholder": "e.g. My Calendar",
    "delete": "Delete",
    "deleteConfirm": "Delete this calendar link?",
    "empty": {
      "title": "No calendar links yet",
      "description": "Create a calendar subscription link to sync your billing dates with external apps."
    },
    "copyLink": "Copy Link",
    "copied": "Link copied to clipboard",
    "copyWarning": "Copy this URL now. You won't be able to see the full token again.",
    "urlLabel": "Subscription URL",
    "usageTitle": "How to use",
    "usageDescription": "Paste this URL into your calendar app's \"Subscribe to calendar\" or \"Add by URL\" feature.",
    "createSuccess": "Calendar link created",
    "deleteSuccess": "Calendar link deleted",
    "limitReached": "Maximum of 5 calendar links reached"
  }
} as const

export default calendar
