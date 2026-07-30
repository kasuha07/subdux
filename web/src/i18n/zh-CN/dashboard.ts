const dashboard = {
  "title": "Subdux",
  "add": "添加",
  "loading": "加载中...",
  "stats": {
    "activeMonthly": "月度费用",
    "activeYearly": "年度费用",
    "activeSubscriptions": "活跃订阅",
    "thisMonth": "本月",
    "upcoming": "即将到期"
  },
  "views": {
    "current": "当前视图：{{view}}",
    "list": "列表视图",
    "cards": "卡片视图",
    "toggleToList": "切换为列表视图",
    "toggleToCards": "切换为卡片视图"
  },
  "empty": {
    "title": "暂无订阅",
    "description": "添加您的第一个订阅开始追踪",
    "addButton": "添加订阅"
  },
  "error": {
    "title": "仪表盘暂不可用",
    "description": "无法加载仪表盘，请重试。",
    "exchangeRateTitle": "汇率暂不可用",
    "exchangeRateDescription": "当前汇率获取失败，为避免显示不准确的总额，数据暂不展示。请稍后重试。",
    "retry": "重试"
  },
  "filters": {
    "searchPlaceholder": "按名称、分类或备注搜索...",
    "filterButton": "筛选",
    "status": "状态",
    "renewalMode": "续费方式",
    "category": "分类",
    "noCategory": "无分类",
    "paymentMethod": "支付方式",
    "noPaymentMethod": "无支付方式",
    "noCategories": "暂无可筛选分类",
    "noPaymentMethods": "暂无可筛选支付方式",
    "clear": "重置",
    "clearFilters": "清空筛选",
    "sortBy": "排序字段",
    "order": "排序方向",
    "sortFields": {
      "nextBillingDate": "扣费/终止",
      "name": "名称",
      "createdAt": "添加时间",
      "amount": "金额"
    },
    "orders": {
      "asc": "升序",
      "desc": "降序"
    },
    "resultCount": "显示 {{shown}} / {{total}}",
    "empty": {
      "title": "没有符合条件的订阅",
      "description": "请调整搜索关键词或筛选条件"
    }
  },
  "deleteConfirm": "确定删除此订阅？",
  "createSuccess": "订阅已创建",
  "updateSuccess": "订阅已更新",
  "deleteSuccess": "订阅已删除"
} as const

export default dashboard
