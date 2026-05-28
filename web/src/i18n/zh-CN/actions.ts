const actions = {
  "title": "行动中心",
  "nav": {
    "back": "返回仪表盘",
    "refresh": "刷新行动项"
  },
  "toast": {
    "markRenewed": "已标记为已续费",
    "cancelAtPeriodEnd": "已设置为到期终止",
    "keepSubscription": "已改为继续保留",
    "snoozed": "已暂时忽略 7 天"
  },
  "error": {
    "title": "行动中心不可用",
    "description": "加载行动中心失败",
    "actionFailed": "操作失败",
    "missingNextBilling": "请先设置下次扣费日，再设置到期终止",
    "subscriptionMissing": "订阅已不存在"
  },
  "summary": {
    "total": "待处理",
    "snoozed": "已暂时忽略 {{count}} 项",
    "grouped": "{{actions}} 项行动 · 已暂时忽略 {{snoozed}} 项",
    "upcoming": "即将扣费",
    "window": "{{urgent}} / {{days}} 天窗口",
    "repair": "需要修复",
    "decision": "{{count}} 项需要决策"
  },
  "empty": {
    "title": "暂无需要处理的事项",
    "description": "即将续费、通知失败和数据修复项会显示在这里。"
  },
  "type": {
    "upcoming_renewal": "即将扣费",
    "manual_renewal_due": "手动续费",
    "ending_soon": "即将结束",
    "notification_failed": "通知失败",
    "missing_next_billing": "缺少扣费日",
    "price_increase": "价格上涨"
  },
  "severity": {
    "critical": "紧急",
    "high": "高",
    "medium": "中",
    "low": "低"
  },
  "message": {
    "upcoming_renewal": "下次自动扣费前检查是否还要保留。",
    "manual_renewal_due": "手动付款后确认续费状态。",
    "ending_soon": "该订阅已设置为近期结束。",
    "notification_failed": "最近一次提醒没有成功送达。",
    "missing_next_billing": "补上下次扣费日以恢复提醒、报表和日历。",
    "price_increase": "检查涨价后是否仍值得继续保留。"
  },
  "priceDelta": "每月 +{{amount}}",
  "group": {
    "itemCount": "{{count}} 项行动"
  },
  "command": {
    "markRenewed": "标记已续费",
    "keepSubscription": "继续保留",
    "cancelAtPeriodEnd": "到期终止",
    "snooze": "暂时忽略"
  },
  "date": {
    "today": "今天",
    "overdue": "已逾期 {{count}} 天",
    "inDays": "{{count}} 天后 · {{date}}",
    "event": "{{date}} 变更",
    "none": "无日期"
  }
} as const

export default actions
