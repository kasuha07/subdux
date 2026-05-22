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
  }
} as const

export default reports
