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
})
