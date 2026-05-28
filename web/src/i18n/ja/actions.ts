const actions = {
  "title": "アクションセンター",
  "nav": {
    "back": "ダッシュボードに戻る",
    "refresh": "アクションを更新"
  },
  "toast": {
    "markRenewed": "更新済みにしました",
    "cancelAtPeriodEnd": "期間終了でキャンセルにしました",
    "keepSubscription": "継続するように変更しました",
    "snoozed": "7日間スヌーズしました"
  },
  "error": {
    "title": "アクションセンターを表示できません",
    "description": "アクションセンターの読み込みに失敗しました",
    "actionFailed": "操作に失敗しました",
    "missingNextBilling": "期間終了でキャンセルする前に次回請求日を設定してください",
    "subscriptionMissing": "サブスクリプションは存在しません"
  },
  "summary": {
    "total": "対応待ち",
    "snoozed": "{{count}}件をスヌーズ中",
    "grouped": "{{actions}}件のアクション · {{snoozed}}件をスヌーズ中",
    "upcoming": "近日の請求",
    "window": "{{urgent}} / {{days}} 日の範囲",
    "repair": "修正が必要",
    "decision": "{{count}}件は判断が必要"
  },
  "empty": {
    "title": "対応が必要な項目はありません",
    "description": "近日の更新、通知失敗、データ修正がここに表示されます。"
  },
  "type": {
    "upcoming_renewal": "近日の請求",
    "manual_renewal_due": "手動更新",
    "ending_soon": "まもなく終了",
    "notification_failed": "通知失敗",
    "missing_next_billing": "請求日未設定",
    "price_increase": "値上げ"
  },
  "severity": {
    "critical": "緊急",
    "high": "高",
    "medium": "中",
    "low": "低"
  },
  "message": {
    "upcoming_renewal": "次回の自動請求前に継続するか確認してください。",
    "manual_renewal_due": "手動支払い後に更新を確認してください。",
    "ending_soon": "このサブスクリプションは近日終了予定です。",
    "notification_failed": "最近のリマインダーを配信できませんでした。",
    "missing_next_billing": "次回請求日を追加して通知、レポート、カレンダーを復旧してください。",
    "price_increase": "新しい価格でも継続する価値があるか確認してください。"
  },
  "priceDelta": "+{{amount}}/月",
  "group": {
    "itemCount": "{{count}}件のアクション"
  },
  "command": {
    "markRenewed": "手動更新",
    "keepSubscription": "継続する",
    "cancelAtPeriodEnd": "キャンセル予定",
    "snooze": "スヌーズ"
  },
  "date": {
    "today": "今日",
    "overdue": "{{count}}日超過",
    "inDays": "{{count}}日後 · {{date}}",
    "event": "{{date}} に変更",
    "none": "日付なし"
  }
} as const

export default actions
