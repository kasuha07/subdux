const subscription = {
  "card": {
    "billingType": {
      "one_time": "買い切り制"
    },
    "recurrence": {
      "interval": {
        "day": "{{count}}日ごと",
        "week": "{{count}}週間ごと",
        "month": "{{count}}か月ごと",
        "year": "{{count}}年ごと"
      },
      "monthlyDate": "毎月{{day}}日",
      "yearlyDate": "毎年{{month}}月{{day}}日"
    },
    "dueIn": "{{count}}日後に更新",
    "overdue": "期限超過",
    "noNextBilling": "次回請求日なし",
    "trial": {
      "startsIn": "トライアル開始まで{{count}}日",
      "active": "トライアル中",
      "endsIn": "トライアル終了まで{{count}}日",
      "endedOn": "トライアル終了日: {{date}}"
    },
    "reminder": {
      "on": "{{days}}日前にリマインド",
      "off": "リマインドオフ"
    },
    "anchorDate": "起点日 {{date}}",
    "notes": "メモ: {{content}}",
    "status": {
      "enabled": "有効",
      "disabled": "無効"
    }
  },
  "form": {
    "editTitle": "サブスクリプションを編集",
    "addTitle": "サブスクリプションを追加",
    "nameLabel": "名前",
    "namePlaceholder": "Netflix, Spotify...",
    "amountLabel": "金額",
    "amountPlaceholder": "9.99",
    "currencyLabel": "通貨",
    "billingTypeLabel": "課金タイプ",
    "billingType": {
      "recurring": "サブスク制",
      "one_time": "買い切り制"
    },
    "enabledLabel": "有効状態",
    "enabled": "有効",
    "disabled": "無効",
    "purchaseDateLabel": "購入日",
    "anchorDateLabel": "課金起点日",
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
    "trialLabel": "トライアル期間",
    "trialStartLabel": "トライアル開始",
    "trialEndLabel": "トライアル終了",
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
      "uploadLabel": "画像をアップロード",
      "uploadHint": "PNG または JPG、最大 {{size}}KB",
      "uploadButton": "ファイルを選択",
      "invalidType": "PNG と JPG 形式のみ対応しています",
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
