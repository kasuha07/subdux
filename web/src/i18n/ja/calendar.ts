const calendar = {
  "title": "カレンダー",
  "back": "戻る",
  "today": "今日",
  "weekdays": {
    "sun": "日",
    "mon": "月",
    "tue": "火",
    "wed": "水",
    "thu": "木",
    "fri": "金",
    "sat": "土"
  },
  "months": {
    "1": "1月",
    "2": "2月",
    "3": "3月",
    "4": "4月",
    "5": "5月",
    "6": "6月",
    "7": "7月",
    "8": "8月",
    "9": "9月",
    "10": "10月",
    "11": "11月",
    "12": "12月"
  },
  "subscriptionsDue": "{{count}} 件のサブスクリプションが期限",
  "noSubscriptions": "この日に期限のサブスクリプションはありません",
  "editSuccess": "サブスクリプションを更新しました",
  "token": {
    "title": "カレンダー購読",
    "description": "購読リンクを生成して、請求日を外部カレンダーアプリ（Google カレンダー、Apple カレンダー、Outlook など）と同期できます",
    "createLink": "リンクを作成",
    "create": "作成",
    "creating": "作成中...",
    "name": "リンク名",
    "namePlaceholder": "例：マイカレンダー",
    "delete": "削除",
    "deleteConfirm": "このカレンダーリンクを削除しますか？",
    "empty": {
      "title": "カレンダーリンクはまだありません",
      "description": "カレンダー購読リンクを作成して、請求日を外部アプリと同期しましょう。"
    },
    "copyLink": "リンクをコピー",
    "copied": "リンクをクリップボードにコピーしました",
    "copyWarning": "この URL を今すぐコピーしてください。完全なトークンは再表示できません。",
    "urlLabel": "購読URL",
    "usageTitle": "使い方",
    "usageDescription": "この URL をカレンダーアプリの「カレンダーを購読」または「URL で追加」機能に貼り付けてください。",
    "createSuccess": "カレンダーリンクを作成しました",
    "deleteSuccess": "カレンダーリンクを削除しました",
    "limitReached": "カレンダーリンクは最大5つまでです"
  }
} as const

export default calendar
