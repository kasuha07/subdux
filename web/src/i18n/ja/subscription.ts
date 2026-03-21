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
    "markRenewed": "更新済みにする",
    "saving": "保存中...",
    "update": "更新",
    "addButton": "サブスクリプションを追加",
    "error": "保存に失敗しました"
  }
} as const

export default subscription
