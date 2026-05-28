const subscription = {
  "card": {
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
    "dueToday": "今天到期",
    "overdue": "已逾期",
    "noNextBilling": "无下次计费日",
    "holdingCost": "{{amount}}/日",
    "reminder": {
      "on": "提前 {{days}} 天提醒",
      "off": "提醒已关闭"
    },
    "endsOn": "{{date}} 结束",
    "endedOn": "已于 {{date}} 结束",
    "notes": "备注：{{content}}",
    "cycleProgressAria": "当前周期进度：{{progress}}%",
    "status": {
      "active": "有效",
      "ended": "已结束"
    },
    "renewalMode": {
      "auto_renew": "自动续费",
      "manual_renew": "手动续费",
      "cancel_at_period_end": "到期终止"
    }
  },
  "detail": {
    "open": "详情",
    "edit": "编辑",
    "titleFallback": "订阅详情",
    "description": "订阅详情抽屉，包含历史、通知和未来扣费点。",
    "error": "加载详情失败",
    "errorTitle": "详情不可用",
    "tabs": {
      "timeline": "时间线",
      "prices": "价格",
      "notifications": "日志",
      "charges": "扣费"
    },
    "summary": {
      "nextCharge": "下次扣费",
      "periodEnd": "本期结束",
      "endingAtPeriodEnd": "到期终止",
      "lifecycle": "生命周期",
      "latestActivity": "最近动态",
      "lastNotification": "最近通过 {{channel}}"
    },
    "info": {
      "title": "订阅信息",
      "amount": "金额",
      "recurrence": "重复规则",
      "nextBillingDate": "下次计费日期",
      "periodEndDate": "本期结束日期",
      "status": "生命周期",
      "renewalMode": "续费方式",
      "endsAt": "结束日期",
      "category": "分类",
      "paymentMethod": "支付方式",
      "notification": "通知",
      "notificationDefault": "使用默认策略",
      "notificationEnabled": "已开启",
      "notificationEnabledWithDays": "已开启，提前 {{days}} 天",
      "notificationDisabled": "已关闭",
      "createdAt": "创建时间",
      "updatedAt": "更新时间",
      "url": "网址",
      "notes": "备注"
    },
    "empty": {
      "none": "无",
      "noUpcomingCharges": "暂无未来扣费",
      "noNotifications": "暂无通知"
    },
    "timeline": {
      "emptyTitle": "暂无时间线",
      "emptyDescription": "此订阅的变更会显示在这里。"
    },
    "prices": {
      "emptyTitle": "暂无价格历史",
      "emptyDescription": "编辑价格后，变化会显示在这里。",
      "from": "原为 {{amount}}"
    },
    "notifications": {
      "emptyTitle": "暂无通知日志",
      "emptyDescription": "此订阅的最近通知会显示在这里。",
      "statusSent": "已发送",
      "statusFailed": "失败"
    },
    "charges": {
      "emptyTitle": "暂无未来扣费",
      "emptyDescription": "此订阅没有未来计费日期。"
    },
    "calendar": {
      "title": "日历",
      "next": "下个日历事件：{{date}}",
      "noEvent": "暂无未来日历事件",
      "open": "打开日历"
    }
  },
  "form": {
    "editTitle": "编辑订阅",
    "addTitle": "添加订阅",
    "editDescription": "请检查下方订阅信息，并保存你的修改。",
    "addDescription": "请填写下方订阅信息，将其添加到你的追踪列表中。",
    "nameLabel": "名称",
    "namePlaceholder": "Netflix, Spotify...",
    "amountLabel": "金额",
    "amountPlaceholder": "9.99",
    "currencyLabel": "货币",
    "statusLabel": "生命周期",
    "status": {
      "active": "有效",
      "ended": "已结束"
    },
    "renewalModeLabel": "续费方式",
    "renewalMode": {
      "auto_renew": "自动续费",
      "manual_renew": "手动续费",
      "cancel_at_period_end": "到期终止"
    },
    "endsAtLabel": "结束日期",
    "purchaseDateLabel": "购买日期",
    "nextBillingDateLabel": "下次计费日期",
    "periodEndDateLabel": "本期结束日期",
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
      "suggestions": {
        "title": "{{domain}} 的图标建议",
        "google": "Google 网站图标",
        "iconHorse": "icon.horse"
      },
      "uploadLabel": "上传图片",
      "uploadHint": "PNG/JPG/ICO，最大 {{size}}KB",
      "uploadButton": "选择文件",
      "invalidType": "仅支持扩展名与内容匹配的有效 PNG/JPG/ICO 文件",
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
