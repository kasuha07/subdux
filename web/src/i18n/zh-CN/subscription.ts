const subscription = {
  "card": {
    "billingType": {
      "one_time": "买断制"
    },
    "recurrence": {
      "interval": {
        "day": "每 {{count}} 天",
        "week": "每 {{count}} 周",
        "month": "每 {{count}} 个月",
        "year": "每 {{count}} 年"
      },
      "monthlyCost": "月",
      "monthlyDate": "每月 {{day}} 号",
      "yearlyDate": "每年 {{month}}/{{day}}"
    },
    "dueIn": "{{count}}天后到期",
    "overdue": "已逾期",
    "noNextBilling": "无下次计费日",
    "holdingCost": "{{amount}}/日",
    "trial": {
      "startsIn": "试用将在 {{count}} 天后开始",
      "active": "试用中",
      "endsIn": "试用剩余 {{count}} 天",
      "endedOn": "试用于 {{date}} 结束"
    },
    "reminder": {
      "on": "提前 {{days}} 天提醒",
      "off": "提醒已关闭"
    },
    "anchorDate": "锚点日 {{date}}",
    "notes": "备注：{{content}}",
    "status": {
      "enabled": "启用",
      "disabled": "停用"
    }
  },
  "form": {
    "editTitle": "编辑订阅",
    "addTitle": "添加订阅",
    "nameLabel": "名称",
    "namePlaceholder": "Netflix, Spotify...",
    "amountLabel": "金额",
    "amountPlaceholder": "9.99",
    "currencyLabel": "货币",
    "billingTypeLabel": "计费类型",
    "billingType": {
      "recurring": "订阅制",
      "one_time": "买断制"
    },
    "enabledLabel": "启用状态",
    "enabled": "启用",
    "disabled": "停用",
    "purchaseDateLabel": "购买日期",
    "anchorDateLabel": "计费锚点日期",
    "recurrenceTypeLabel": "重复规则",
    "recurrenceDetailLabel": "具体内容",
    "recurrenceType": {
      "interval": "时间间隔",
      "monthly_date": "每月固定日期",
      "yearly_date": "每年固定日期"
    },
    "intervalCountLabel": "每 N",
    "intervalUnitLabel": "单位",
    "intervalUnit": {
      "day": "天",
      "week": "周",
      "month": "月",
      "year": "年"
    },
    "monthlyDayLabel": "每月日期",
    "yearlyMonthLabel": "月份",
    "yearlyDayLabel": "日期",
    "trialLabel": "试用期",
    "trialStartLabel": "试用开始",
    "trialEndLabel": "试用结束",
    "categoryLabel": "分类",
    "categoryPlaceholder": "选择...",
    "paymentMethodLabel": "支付方式",
    "paymentMethodPlaceholder": "选择...",
    "noPaymentMethod": "不设",
    "categories": {
      "entertainment": "娱乐",
      "productivity": "效率",
      "development": "开发",
      "music": "音乐",
      "cloud": "云服务",
      "finance": "财务",
      "health": "健康",
      "education": "教育",
      "news": "新闻",
      "other": "其他"
    },
    "iconLabel": "图标 / 表情",
    "iconPlaceholder": "🎬",
    "iconPicker": {
      "label": "图标",
      "tabs": {
        "emoji": "表情",
        "brand": "品牌图标",
        "image": "图片"
      },
      "emojiPlaceholder": "🎬",
      "searchPlaceholder": "搜索品牌图标...",
      "noResults": "未找到图标",
      "urlLabel": "图片链接",
      "urlPlaceholder": "https://example.com/logo.png",
      "uploadLabel": "上传图片",
      "uploadHint": "PNG 或 JPG，最大 {{size}}KB",
      "uploadButton": "选择文件",
      "invalidType": "仅支持 PNG 和 JPG 格式",
      "fileTooLarge": "文件超过 {{size}}KB 限制",
      "removeFile": "移除"
    },
    "iconUploadFailed": "图标上传失败，但订阅已保存",
    "urlLabel": "网址",
    "urlPlaceholder": "https://...",
    "notesLabel": "备注",
    "notesPlaceholder": "可选备注...",
    "cancel": "取消",
    "saving": "保存中...",
    "update": "更新",
    "addButton": "添加订阅",
    "error": "保存失败"
  }
} as const

export default subscription
