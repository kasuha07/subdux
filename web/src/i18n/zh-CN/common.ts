import backendMessages from "./backend-messages"

const common = {
  "loading": "加载中...",
  "cancel": "取消",
  "close": "关闭",
  "clear": "清空",
  "back": "返回",
  "copy": "复制",
  "copied": "已复制",
  "showPassword": "显示密码",
  "hidePassword": "隐藏密码",
  "unauthorized": "未授权",
  "requestFailed": "请求失败",
  "backendMessages": backendMessages,
  "passkeyErrors": {
    "notAllowed": "Passkey 请求已取消或超时",
    "notSupported": "当前设备或浏览器不支持 Passkey",
    "invalidState": "该 Passkey 已在此设备上注册",
    "security": "当前站点无法使用 Passkey，请检查域名和 HTTPS 配置",
    "aborted": "Passkey 请求被中断",
    "notFound": "未找到可用的 Passkey"
  }
} as const

export default common
