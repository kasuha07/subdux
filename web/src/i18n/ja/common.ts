const common = {
  "loading": "読み込み中...",
  "cancel": "キャンセル",
  "unauthorized": "認証エラー",
  "requestFailed": "リクエストに失敗しました",
  "passkeyErrors": {
    "notAllowed": "Passkey リクエストはキャンセルされたかタイムアウトしました",
    "notSupported": "この端末またはブラウザは Passkey に対応していません",
    "invalidState": "この Passkey はこの端末ですでに登録されています",
    "security": "このサイトでは Passkey を利用できません。ドメインと HTTPS 設定を確認してください",
    "aborted": "Passkey リクエストが中断されました",
    "notFound": "一致する Passkey が見つかりませんでした"
  }
} as const

export default common
