const dashboard = {
  "title": "Subdux",
  "add": "追加",
  "loading": "読み込み中...",
  "stats": {
    "monthly": "月額",
    "yearly": "年額",
    "enabled": "有効",
    "upcoming": "更新予定"
  },
  "views": {
    "current": "現在の表示: {{view}}",
    "list": "リスト表示",
    "cards": "カード表示",
    "toggleToList": "リスト表示に切替",
    "toggleToCards": "カード表示に切替"
  },
  "empty": {
    "title": "サブスクリプションがありません",
    "description": "最初のサブスクリプションを追加して追跡を始めましょう",
    "addButton": "サブスクリプションを追加"
  },
  "filters": {
    "searchPlaceholder": "名前・カテゴリ・メモで検索...",
    "filterButton": "フィルター",
    "status": "ステータス",
    "category": "カテゴリ",
    "noCategory": "カテゴリ未設定",
    "paymentMethod": "支払い方法",
    "noPaymentMethod": "支払い方法未設定",
    "noCategories": "選択可能なカテゴリがありません",
    "noPaymentMethods": "選択可能な支払い方法がありません",
    "clear": "リセット",
    "clearFilters": "フィルターをクリア",
    "sortBy": "並び替え項目",
    "order": "順序",
    "sortFields": {
      "nextBillingDate": "次回請求日",
      "name": "名前",
      "createdAt": "追加日時",
      "amount": "金額"
    },
    "orders": {
      "asc": "昇順",
      "desc": "降順"
    },
    "resultCount": "{{total}} 件中 {{shown}} 件を表示",
    "empty": {
      "title": "一致するサブスクリプションがありません",
      "description": "検索キーワードまたはフィルター条件を調整してください"
    }
  },
  "deleteConfirm": "このサブスクリプションを削除しますか？",
  "createSuccess": "サブスクリプションを作成しました",
  "updateSuccess": "サブスクリプションを更新しました",
  "deleteSuccess": "サブスクリプションを削除しました"
} as const

export default dashboard
