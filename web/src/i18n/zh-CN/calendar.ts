const calendar = {
  "title": "日历",
  "back": "返回",
  "today": "今天",
  "weekdays": {
    "sun": "日",
    "mon": "一",
    "tue": "二",
    "wed": "三",
    "thu": "四",
    "fri": "五",
    "sat": "六"
  },
  "months": {
    "1": "一月",
    "2": "二月",
    "3": "三月",
    "4": "四月",
    "5": "五月",
    "6": "六月",
    "7": "七月",
    "8": "八月",
    "9": "九月",
    "10": "十月",
    "11": "十一月",
    "12": "十二月"
  },
  "subscriptionsDue": "{{count}} 个订阅到期",
  "noSubscriptions": "当天没有订阅到期",
  "editSuccess": "订阅已更新",
  "token": {
    "title": "日历订阅",
    "description": "生成订阅链接，将账单日期同步到外部日历应用（Google 日历、Apple 日历、Outlook 等）",
    "createLink": "创建链接",
    "create": "创建",
    "creating": "创建中...",
    "name": "链接名称",
    "namePlaceholder": "例如：我的日历",
    "delete": "删除",
    "deleteConfirm": "确定删除此日历链接？",
    "empty": {
      "title": "暂无日历链接",
      "description": "创建日历订阅链接，将账单日期同步到外部应用。"
    },
    "copyLink": "复制链接",
    "copied": "链接已复制到剪贴板",
    "copyWarning": "请立即复制此链接，之后将无法再次查看完整令牌。",
    "urlLabel": "订阅链接",
    "usageTitle": "使用方法",
    "usageDescription": "将此链接粘贴到日历应用的「订阅日历」或「通过 URL 添加」功能中。",
    "createSuccess": "日历链接已创建",
    "deleteSuccess": "日历链接已删除",
    "limitReached": "最多创建 5 个日历链接"
  }
} as const

export default calendar
