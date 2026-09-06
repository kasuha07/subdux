import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it, vi } from "vitest"

import DashboardFiltersToolbar from "./dashboard-filters-toolbar"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === "dashboard.filters.resultCount") {
        return `Showing ${params?.shown} / ${params?.total}`
      }
      return key
    },
    i18n: { language: "en" },
  }),
}))

describe("DashboardFiltersToolbar responsive design", () => {
  const defaultProps = {
    batchMode: false,
    onToggleBatchMode: vi.fn(),
    categoryOptions: ["Streaming", "Cloud"],
    getSortFieldLabel: (field: string) => `Field:${field}`,
    hasActiveFilters: false,
    includeNoCategory: false,
    includeNoPaymentMethod: false,
    onResetFiltersAndSorting: vi.fn(),
    onSearchTermChange: vi.fn(),
    onSortFieldSelect: vi.fn(),
    onToggleCategory: vi.fn(),
    onToggleRenewalMode: vi.fn(),
    onToggleStatus: vi.fn(),
    onToggleNoCategory: vi.fn(),
    onToggleNoPaymentMethod: vi.fn(),
    onTogglePaymentMethod: vi.fn(),
    paymentMethodLabelMap: new Map(),
    paymentMethods: [],
    searchTerm: "",
    shownCount: 5,
    subscriptionView: "list" as const,
    selectedCategories: new Set<string>(),
    selectedPaymentMethodIDs: new Set<number>(),
    selectedRenewalModes: new Set<never>(),
    selectedStatuses: new Set<"active">(["active"]),
    totalCount: 10,
    onToggleSubscriptionView: vi.fn(),
    viewToggleDisabled: false,
    sortDirection: "asc" as const,
    sortField: "name" as const,
  }

  it("renders flexible search input and responsive toolbar structure", () => {
    const markup = renderToStaticMarkup(<DashboardFiltersToolbar {...defaultProps} />)

    // Variable-length search input wrapper with max-width limit
    expect(markup).toContain("relative w-full min-w-0 flex-1 max-w-[260px]")
    // Responsive outer container with flex-wrap
    expect(markup).toContain("flex flex-wrap items-center justify-between gap-3")
    // Responsive button container
    expect(markup).toContain("flex items-center gap-2 shrink-0")
  })

  it("renders filter and sort triggers with responsive icon-only classes and tooltips", () => {
    const markup = renderToStaticMarkup(<DashboardFiltersToolbar {...defaultProps} />)

    // Filter button contains icon-only classes and tooltip content trigger
    expect(markup).toContain("w-8 px-0 md:w-auto md:px-3")
    expect(markup).toContain("dashboard.filters.filterButton")

    // Sort button contains sort field label and tooltip trigger
    expect(markup).toContain("Field:name")
    expect(markup).toContain("dashboard.filters.sortBy")
  })

  it("displays active filter count badge and tooltip count when filters are applied", () => {
    const propsWithFilters = {
      ...defaultProps,
      hasActiveFilters: true,
      selectedCategories: new Set(["Streaming", "Cloud"]),
    }
    const markup = renderToStaticMarkup(<DashboardFiltersToolbar {...propsWithFilters} />)

    // Mobile badge showing active count (2)
    expect(markup).toContain("md:hidden")
    expect(markup).toContain(">2<")
    expect(markup).toContain("dashboard.filters.filterButton (2)")
  })

  it("hides result counter on very small screens below 480px", () => {
    const markup = renderToStaticMarkup(<DashboardFiltersToolbar {...defaultProps} />)

    // Result counter is hidden on extra small viewports and displayed from 480px
    expect(markup).toContain("hidden cursor-default select-none text-sm tabular-nums text-muted-foreground shrink-0 min-[480px]:block")
    expect(markup).toContain("5 / 10")
  })

  it("renders keyboard shortcut hint across breakpoints when search term is empty", () => {
    const markup = renderToStaticMarkup(<DashboardFiltersToolbar {...defaultProps} searchTerm="" />)

    // Shortcut badge is inline-flex without hidden/sm: breakpoint gating
    expect(markup).toContain("<kbd")
    expect(markup).toContain("inline-flex")
    expect(markup).not.toContain("hidden -translate-y-1/2 select-none items-center rounded border border-border/80 bg-muted/60 px-1.5 font-mono text-[10px] font-medium text-muted-foreground sm:inline-flex")
    expect(markup).toContain(">/</kbd>")
  })

  it("does not render keyboard shortcut hint on mobile devices", () => {
    vi.stubGlobal("navigator", {
      userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)",
      maxTouchPoints: 5,
    })
    vi.stubGlobal("window", {})

    const markup = renderToStaticMarkup(<DashboardFiltersToolbar {...defaultProps} searchTerm="" />)
    expect(markup).not.toContain("<kbd")
    expect(markup).not.toContain(">/</kbd>")
    expect(markup).toContain("pr-3")

    vi.unstubAllGlobals()
  })

  it("calculates active filter count across status, renewal mode, category, and payment methods", () => {
    const props = {
      ...defaultProps,
      hasActiveFilters: true,
      selectedStatuses: new Set<"active" | "paused">(["active", "paused"]),
      selectedRenewalModes: new Set<"auto_renewal">(["auto_renewal"]),
      includeNoCategory: true,
      selectedCategories: new Set(["Streaming"]),
      includeNoPaymentMethod: true,
      selectedPaymentMethodIDs: new Set([1]),
    }
    const markup = renderToStaticMarkup(<DashboardFiltersToolbar {...props} />)

    // Total active count: status (1) + renewal (1) + noCategory (1) + category (1) + noPayment (1) + payment (1) = 6
    expect(markup).toContain(">6<")
    expect(markup).toContain("dashboard.filters.filterButton (6)")
  })

  it("renders sort direction indicators for both ascending and descending orders", () => {
    const ascMarkup = renderToStaticMarkup(
      <DashboardFiltersToolbar {...defaultProps} sortDirection="asc" />
    )
    expect(ascMarkup).toContain("lucide-arrow-up")

    const descMarkup = renderToStaticMarkup(
      <DashboardFiltersToolbar {...defaultProps} sortDirection="desc" />
    )
    expect(descMarkup).toContain("lucide-arrow-down")
  })
})
