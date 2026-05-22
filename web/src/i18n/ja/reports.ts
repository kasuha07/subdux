const reports = {
  "title": "レポート分析",
  "error": {
    "title": "レポートを表示できません",
    "description": "ページを更新するか、しばらくしてから再試行してください。"
  },
  "empty": {
    "title": "レポートデータがありません",
    "description": "有効なサブスクリプションを追加すると分析を表示できます。"
  },
  "kpis": {
    "monthly": "月額支出",
    "yearlyDetail": "年額 {{amount}}",
    "committed": "自動更新の支出",
    "autoRenewDetail": "{{count}} 件の自動更新",
    "next30Days": "今後 30 日",
    "renewalDetail": "{{count}} 件の更新",
    "active": "有効なサブスクリプション",
    "modeDetail": "{{auto}} 自動 / {{manual}} 手動 / {{canceling}} 終了予定"
  },
  "yearlyStats": {
    "title": "年間統計",
    "totalYearly": "年額支出",
    "committedYearly": "自動更新の年額支出",
    "forecast12Months": "12か月予測支出",
    "forecastPayments": "12か月の支払い回数",
    "monthlyAverage": "月平均 {{amount}}",
    "autoRenewCount": "{{count}} 件の自動更新",
    "forecastMonths": "{{count}} か月分",
    "paymentDetail": "予測支払い回数"
  },
  "forecast": {
    "title": "12 か月の請求予測",
    "occurrences": "{{count}} 件の支払い"
  },
  "categories": {
    "title": "カテゴリ別内訳",
    "none": "未分類"
  },
  "paymentMethods": {
    "title": "支払い方法別内訳",
    "none": "支払い方法なし"
  },
  "renewalModes": {
    "title": "更新方法別内訳",
    "auto_renew": "自動更新",
    "manual_renew": "手動更新",
    "cancel_at_period_end": "期間終了時に終了"
  },
  "topSubscriptions": {
    "title": "月額コスト上位",
    "monthly": "月額"
  },
  "upcoming": {
    "title": "今後の更新",
    "daysUntil": "{{count}} 日後",
    "emptyTitle": "近日中の更新はありません",
    "emptyDescription": "今後 30 日以内に更新される有効なサブスクリプションはありません。"
  }
} as const

export default reports
