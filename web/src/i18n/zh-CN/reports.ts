const reports = {
  "title": "报表分析",
  "error": {
    "title": "报表暂不可用",
    "description": "请刷新页面或稍后重试。"
  },
  "empty": {
    "title": "暂无报表数据",
    "description": "添加活跃订阅后即可查看分析。"
  },
  "kpis": {
    "monthly": "月度支出",
    "yearlyDetail": "年度 {{amount}}",
    "committed": "自动续费支出",
    "autoRenewDetail": "{{count}} 个自动续费订阅",
    "next30Days": "未来 30 天",
    "renewalDetail": "{{count}} 次续费",
    "active": "活跃订阅",
    "modeDetail": "{{auto}} 自动 / {{manual}} 手动 / {{canceling}} 将结束"
  },
  "yearlyStats": {
    "title": "年度统计",
    "totalYearly": "年度支出",
    "committedYearly": "自动续费年支出",
    "forecast12Months": "12 个月预测支出",
    "forecastPayments": "12 个月付款次数",
    "monthlyAverage": "月均 {{amount}}",
    "autoRenewCount": "{{count}} 个自动续费订阅",
    "forecastMonths": "覆盖 {{count}} 个月",
    "paymentDetail": "预测付款次数"
  },
  "forecast": {
    "title": "12 个月扣费预测",
    "occurrences": "{{count}} 次付款"
  },
  "categories": {
    "title": "分类支出拆分",
    "none": "未分类"
  },
  "paymentMethods": {
    "title": "支付方式拆分",
    "none": "无支付方式"
  },
  "renewalModes": {
    "title": "续费方式拆分",
    "auto_renew": "自动续费",
    "manual_renew": "手动续费",
    "cancel_at_period_end": "周期结束后停止"
  },
  "topSubscriptions": {
    "title": "最高月成本订阅",
    "monthly": "月均"
  },
  "upcoming": {
    "title": "即将续费",
    "daysUntil": "{{count}} 天后",
    "emptyTitle": "近期无续费",
    "emptyDescription": "未来 30 天没有活跃订阅需要续费。"
  },
  "priceIncreases": {
    "title": "近期涨价",
    "delta": "月均增加 {{amount}}",
    "emptyTitle": "暂无涨价记录",
    "emptyDescription": "订阅金额上涨后会显示在这里。"
  },
  "annualGrowth": {
    "title": "年度增长",
    "fromTo": "{{from}} -> {{to}}",
    "emptyTitle": "暂无增长数据",
    "emptyDescription": "订阅有历史变更后会显示月均成本增长。"
  },
  "recentChanges": {
    "title": "近期变更",
    "emptyTitle": "暂无近期变更",
    "emptyDescription": "创建、更新、续费和删除记录会显示在这里。",
    "types": {
      "created": "已创建",
      "updated": "已更新",
      "manual_renewed": "手动续费",
      "deleted": "已删除",
      "system_change": "系统变更"
    },
    "fields": {
      "created": "创建",
      "deleted": "删除",
      "amount": "金额",
      "currency": "货币",
      "monthly_amount": "月均金额",
      "next_billing_date": "下次计费",
      "status": "状态",
      "renewal_mode": "续费方式",
      "category": "分类",
      "payment_method": "支付方式"
    }
  }
} as const

export default reports
