const dashboard = {
  "title": "Subdux",
  "add": "Add",
  "loading": "Loading...",
  "stats": {
    "monthly": "Monthly",
    "yearly": "Yearly",
    "enabled": "Enabled",
    "upcoming": "Upcoming"
  },
  "views": {
    "current": "Current view: {{view}}",
    "list": "List view",
    "cards": "Card view",
    "toggleToList": "Switch to list view",
    "toggleToCards": "Switch to card view"
  },
  "empty": {
    "title": "No subscriptions yet",
    "description": "Start tracking by adding your first subscription",
    "addButton": "Add subscription"
  },
  "filters": {
    "searchPlaceholder": "Search by name, category, or notes...",
    "filter": "Filter",
    "status": "Status",
    "category": "Category",
    "noCategory": "No category",
    "paymentMethod": "Payment method",
    "noPaymentMethod": "No payment method",
    "noCategories": "No categories available",
    "noPaymentMethods": "No payment methods available",
    "clear": "Reset",
    "clearFilters": "Clear filters",
    "sort": "Sort",
    "sortBy": "Sort by",
    "order": "Order",
    "sortFields": {
      "nextBillingDate": "Next billing date",
      "name": "Name",
      "createdAt": "Added time",
      "amount": "Amount"
    },
    "orders": {
      "asc": "Ascending",
      "desc": "Descending"
    },
    "resultCount": "Showing {{shown}} / {{total}}",
    "empty": {
      "title": "No matching subscriptions",
      "description": "Try adjusting your search or filters"
    }
  },
  "deleteConfirm": "Delete this subscription?",
  "createSuccess": "Subscription created",
  "updateSuccess": "Subscription updated",
  "deleteSuccess": "Subscription deleted"
} as const

export default dashboard
