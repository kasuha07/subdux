const subscription = {
  "card": {
    "billingType": {
      "legacy": "旧買い切り"
    },
    "recurrence": {
      "interval": {
        "day": "{{count}}日ごと",
        "week": "{{count}}週間ごと",
        "month": "{{count}}か月ごと",
        "year": "{{count}}年ごと"
      },
      "monthlyCost": "月",
      "monthlyDate": "毎月{{day}}日",
      "yearlyDate": "毎年{{month}}月{{day}}日"
    },
    "dueIn": "{{count}}日後に更新",
    "dueToday": "今日更新",
    "overdue": "期限超過",
    "noNextBilling": "次回請求日なし",
    "holdingCost": "{{amount}}/日",
    "reminder": {
      "on": "{{days}}日前にリマインド",
      "off": "リマインドオフ"
    },
    "endsOn": "{{date}} に終了",
    "endedOn": "{{date}} に終了済み",
    "notes": "メモ: {{content}}",
    "cycleProgressAria": "現在のサイクル進捗: {{progress}}%",
    "status": {
      "active": "有効",
      "ended": "終了"
    },
    "renewalMode": {
      "auto_renew": "自動更新",
      "manual_renew": "手動更新",
      "cancel_at_period_end": "今期終了で終了"
    }
  },
  "detail": {
    "open": "詳細",
    "edit": "編集",
    "titleFallback": "サブスク詳細",
    "description": "履歴、通知、今後の請求を表示するサブスクリプション詳細ドロワーです。",
    "error": "詳細の読み込みに失敗しました",
    "errorTitle": "詳細を表示できません",
    "tabs": {
      "timeline": "履歴",
      "prices": "価格",
      "notifications": "ログ",
      "charges": "請求"
    },
    "summary": {
      "nextCharge": "次回請求",
      "lifecycle": "ライフサイクル",
      "latestActivity": "最新アクティビティ",
      "lastNotification": "直近: {{channel}}"
    },
    "info": {
      "title": "サブスク情報",
      "amount": "金額",
      "billingType": "課金タイプ",
      "recurrence": "繰り返し",
      "nextBillingDate": "次回請求日",
      "status": "ライフサイクル",
      "renewalMode": "更新方法",
      "endsAt": "終了日",
      "category": "カテゴリ",
      "paymentMethod": "支払い方法",
      "notification": "通知",
      "notificationDefault": "デフォルトポリシーを使用",
      "notificationEnabled": "有効",
      "notificationEnabledWithDays": "有効、{{days}}日前",
      "notificationDisabled": "無効",
      "createdAt": "作成日",
      "url": "URL",
      "notes": "メモ"
    },
    "empty": {
      "none": "なし",
      "noUpcomingCharges": "今後の請求なし",
      "noNotifications": "通知なし"
    },
    "timeline": {
      "emptyTitle": "履歴はまだありません",
      "emptyDescription": "このサブスクリプションの変更がここに表示されます。"
    },
    "prices": {
      "emptyTitle": "価格履歴なし",
      "emptyDescription": "価格を編集すると変更がここに表示されます。",
      "from": "{{amount}} から"
    },
    "notifications": {
      "emptyTitle": "通知ログなし",
      "emptyDescription": "このサブスクリプションの最近の通知がここに表示されます。",
      "statusSent": "送信済み",
      "statusFailed": "失敗"
    },
    "charges": {
      "emptyTitle": "今後の請求なし",
      "emptyDescription": "このサブスクリプションには今後の請求日がありません。"
    },
    "calendar": {
      "title": "カレンダー",
      "next": "次のカレンダー予定: {{date}}",
      "noEvent": "今後のカレンダー予定なし",
      "open": "カレンダーを開く"
    }
  },
  "form": {
    "editTitle": "サブスクリプションを編集",
    "addTitle": "サブスクリプションを追加",
    "editDescription": "以下のサブスクリプション情報を確認し、変更内容を保存してください。",
    "addDescription": "以下のサブスクリプション情報を入力して、トラッカーに追加してください。",
    "nameLabel": "名前",
    "namePlaceholder": "Netflix, Spotify...",
    "amountLabel": "金額",
    "amountPlaceholder": "9.99",
    "currencyLabel": "通貨",
    "billingTypeLabel": "課金タイプ",
    "billingType": {
      "recurring": "サブスク制",
      "one_time": "旧買い切り（非推奨）"
    },
    "statusLabel": "ライフサイクル",
    "status": {
      "active": "有効",
      "ended": "終了"
    },
    "renewalModeLabel": "更新方法",
    "renewalMode": {
      "auto_renew": "自動更新",
      "manual_renew": "手動更新",
      "cancel_at_period_end": "今期終了で終了"
    },
    "endsAtLabel": "終了日",
    "purchaseDateLabel": "購入日",
    "nextBillingDateLabel": "次回請求日",
    "recurrenceTypeLabel": "繰り返しルール",
    "recurrenceDetailLabel": "詳細",
    "recurrenceType": {
      "interval": "カスタム",
      "monthly_date": "毎月の特定日",
      "yearly_date": "毎年の特定日"
    },
    "intervalCountLabel": "N の値",
    "intervalUnitLabel": "単位",
    "intervalUnit": {
      "day": "日",
      "week": "週",
      "month": "月",
      "year": "年"
    },
    "monthlyDayLabel": "毎月の日付",
    "yearlyMonthLabel": "月",
    "yearlyDayLabel": "日",
    "categoryLabel": "カテゴリ",
    "categoryPlaceholder": "選択...",
    "paymentMethodLabel": "支払い方法",
    "paymentMethodPlaceholder": "選択...",
    "noPaymentMethod": "支払い方法なし",
    "categories": {
      "entertainment": "エンタメ",
      "productivity": "生産性",
      "development": "開発",
      "music": "音楽",
      "cloud": "クラウド",
      "finance": "金融",
      "health": "健康",
      "education": "教育",
      "news": "ニュース",
      "other": "その他"
    },
    "iconLabel": "アイコン / 絵文字",
    "iconPlaceholder": "🎬",
    "iconPicker": {
      "label": "アイコン",
      "tabs": {
        "emoji": "絵文字",
        "brand": "ブランド",
        "image": "画像"
      },
      "emojiPlaceholder": "🎬",
      "searchPlaceholder": "ブランドアイコンを検索...",
      "noResults": "アイコンが見つかりません",
      "urlLabel": "画像URL",
      "urlPlaceholder": "https://example.com/logo.png",
      "suggestions": {
        "title": "{{domain}} の候補アイコン",
        "google": "Google ファビコン",
        "iconHorse": "icon.horse"
      },
      "uploadLabel": "画像をアップロード",
      "uploadHint": "PNG / JPG / ICO、最大 {{size}}KB",
      "uploadButton": "ファイルを選択",
      "invalidType": "拡張子と内容が一致する有効な PNG/JPG/ICO のみ対応しています",
      "fileTooLarge": "ファイルが {{size}}KB の制限を超えています",
      "removeFile": "削除"
    },
    "iconUploadFailed": "アイコンのアップロードに失敗しましたが、サブスクリプションは保存されました",
    "urlLabel": "URL",
    "urlPlaceholder": "https://...",
    "notesLabel": "メモ",
    "notesPlaceholder": "メモ（任意）...",
    "cancel": "キャンセル",
    "saving": "保存中...",
    "update": "更新",
    "addButton": "サブスクリプションを追加",
    "error": "保存に失敗しました"
  }
} as const

export default subscription
