const dashboard = {
  "title": "Subdux",
  "add": "添加",
  "loading": "加载中...",
  "stats": {
    "monthly": "月度",
    "yearly": "年度",
    "enabled": "启用",
    "upcoming": "即将到期"
  },
  "empty": {
    "title": "暂无订阅",
    "description": "添加您的第一个订阅开始追踪",
    "addButton": "添加订阅"
  },
  "filters": {
    "searchPlaceholder": "按名称、分类或备注搜索...",
    "filter": "筛选",
    "status": "状态",
    "category": "分类",
    "noCategory": "无分类",
    "paymentMethod": "支付方式",
    "noPaymentMethod": "无支付方式",
    "noCategories": "暂无可筛选分类",
    "noPaymentMethods": "暂无可筛选支付方式",
    "clear": "重置",
    "clearFilters": "清空筛选",
    "sort": "排序",
    "sortBy": "排序字段",
    "order": "排序方向",
    "sortFields": {
      "nextBillingDate": "下次扣费日",
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
