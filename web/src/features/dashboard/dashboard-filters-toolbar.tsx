import { useEffect, useRef, useState, useSyncExternalStore } from "react"
import { useTranslation } from "react-i18next"
import {
  CheckSquare,
  ChevronDown,
  X,
  ArrowDown,
  ArrowUp,
  ArrowUpDown,
  Filter,
  FilterX,
  Grid3X3,
  List,
  Search,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { useIsMobileDevice } from "@/lib/device"
import { cn } from "@/lib/utils"
import type { PaymentMethod, SubscriptionRenewalMode } from "@/types"
import {
  renewalModeOptions,
  sortFieldOptions,
  type SortDirection,
  type SortField,
  statusOptions,
  type StatusFilter,
} from "./dashboard-filter-constants"

interface DashboardFiltersToolbarProps {
  batchMode: boolean
  onToggleBatchMode: () => void
  categoryOptions: string[]
  getSortFieldLabel: (field: SortField) => string
  hasActiveFilters: boolean
  includeNoCategory: boolean
  includeNoPaymentMethod: boolean
  onResetFiltersAndSorting: () => void
  onSearchTermChange: (value: string) => void
  onSortFieldSelect: (field: SortField) => void
  onToggleCategory: (category: string, checked: boolean) => void
  onToggleRenewalMode: (mode: SubscriptionRenewalMode, checked: boolean) => void
  onToggleStatus: (status: StatusFilter, checked: boolean) => void
  onToggleNoCategory: (checked: boolean) => void
  onToggleNoPaymentMethod: (checked: boolean) => void
  onTogglePaymentMethod: (paymentMethodID: number, checked: boolean) => void
  paymentMethodLabelMap: Map<number, string>
  paymentMethods: PaymentMethod[]
  searchTerm: string
  shownCount: number
  subscriptionView: "list" | "cards"
  selectedCategories: Set<string>
  selectedPaymentMethodIDs: Set<number>
  selectedRenewalModes: Set<SubscriptionRenewalMode>
  selectedStatuses: Set<StatusFilter>
  totalCount: number
  onToggleSubscriptionView: () => void
  viewToggleDisabled?: boolean
  sortDirection: SortDirection
  sortField: SortField
}

function useIsCompactScreen(): boolean {
  const isMobile = useIsMobileDevice()
  const isSmallScreen = useSyncExternalStore(
    (onStoreChange) => {
      if (typeof window === "undefined" || !window.matchMedia) return () => {}
      try {
        const mql = window.matchMedia("(max-width: 767px)")
        if (mql.addEventListener) {
          mql.addEventListener("change", onStoreChange)
          return () => mql.removeEventListener("change", onStoreChange)
        } else if ("addListener" in mql) {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const legacy = mql as any
          legacy.addListener(onStoreChange)
          return () => legacy.removeListener(onStoreChange)
        }
      } catch {
        // Ignore evaluation errors
      }
      return () => {}
    },
    () => {
      if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
        return false
      }
      try {
        return window.matchMedia("(max-width: 767px)").matches
      } catch {
        return false
      }
    },
    () => false
  )
  return isMobile || isSmallScreen
}

export default function DashboardFiltersToolbar({
  batchMode,
  onToggleBatchMode,
  categoryOptions,
  getSortFieldLabel,
  hasActiveFilters,
  includeNoCategory,
  includeNoPaymentMethod,
  onResetFiltersAndSorting,
  onSearchTermChange,
  onSortFieldSelect,
  onToggleCategory,
  onToggleRenewalMode,
  onToggleStatus,
  onToggleNoCategory,
  onToggleNoPaymentMethod,
  onTogglePaymentMethod,
  paymentMethodLabelMap,
  paymentMethods,
  searchTerm,
  shownCount,
  subscriptionView,
  selectedCategories,
  selectedPaymentMethodIDs,
  selectedRenewalModes,
  selectedStatuses,
  totalCount,
  onToggleSubscriptionView,
  viewToggleDisabled = false,
  sortDirection,
  sortField,
}: DashboardFiltersToolbarProps) {
  const { t } = useTranslation()
  const isMobile = useIsMobileDevice()
  const isCompact = useIsCompactScreen()

  const searchInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    function handleGlobalKeyDown(event: KeyboardEvent) {
      if (event.key === "/" && !event.ctrlKey && !event.metaKey && !event.altKey) {
        const activeEl = document.activeElement
        const tagName = activeEl?.tagName.toLowerCase()
        if (
          tagName === "input" ||
          tagName === "textarea" ||
          tagName === "select" ||
          activeEl?.getAttribute("contenteditable") === "true"
        ) {
          return
        }
        event.preventDefault()
        searchInputRef.current?.focus()
      }
    }

    window.addEventListener("keydown", handleGlobalKeyDown)
    return () => window.removeEventListener("keydown", handleGlobalKeyDown)
  }, [])

  const [expandedSections, setExpandedSections] = useState<Set<string>>(() => new Set())

  const toggleSection = (section: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev)
      if (next.has(section)) {
        next.delete(section)
      } else {
        next.add(section)
      }
      return next
    })
  }

  const statusActiveCount =
    selectedStatuses.size === 1 && selectedStatuses.has("active")
      ? 0
      : selectedStatuses.size
  const renewalModeActiveCount = selectedRenewalModes.size
  const categoryActiveCount = selectedCategories.size + (includeNoCategory ? 1 : 0)
  const paymentMethodActiveCount = selectedPaymentMethodIDs.size + (includeNoPaymentMethod ? 1 : 0)

  const activeFilterCount =
    (selectedStatuses.size === 1 && selectedStatuses.has("active") ? 0 : 1) +
    selectedCategories.size +
    (includeNoCategory ? 1 : 0) +
    selectedPaymentMethodIDs.size +
    selectedRenewalModes.size +
    (includeNoPaymentMethod ? 1 : 0)

  const [filterMenuOpen, setFilterMenuOpen] = useState(false)
  const [sortMenuOpen, setSortMenuOpen] = useState(false)

  const filterTooltipText =
    activeFilterCount > 0
      ? `${t("dashboard.filters.filterButton")} (${activeFilterCount})`
      : t("dashboard.filters.filterButton")

  const sortTooltipText = `${t("dashboard.filters.sortBy")}: ${getSortFieldLabel(sortField)} (${t(`dashboard.filters.orders.${sortDirection}`)})`

  return (
    <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <div className="relative w-full min-w-0 flex-1 max-w-[260px]">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            ref={searchInputRef}
            value={searchTerm}
            onChange={(event) => onSearchTermChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Escape") {
                event.preventDefault()
                if (searchTerm) {
                  onSearchTermChange("")
                } else {
                  searchInputRef.current?.blur()
                }
              }
            }}
            placeholder={t("dashboard.filters.searchPlaceholder")}
            className={cn("pl-9", searchTerm ? "pr-8" : isMobile ? "pr-3" : "pr-9")}
          />
          {searchTerm ? (
            <Tooltip content={t("common.clear")}>
              <button
                type="button"
                onClick={() => onSearchTermChange("")}
                className="absolute top-1/2 right-2.5 -translate-y-1/2 rounded-xs p-0.5 text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-ring"
                aria-label={t("common.clear")}
              >
                <X className="size-3.5" />
              </button>
            </Tooltip>
          ) : !isMobile ? (
            <kbd className="pointer-events-none absolute top-1/2 right-2.5 inline-flex -translate-y-1/2 select-none items-center rounded border border-border/80 bg-muted/60 px-1.5 font-mono text-[10px] font-medium text-muted-foreground">
              /
            </kbd>
          ) : null}
        </div>

        {totalCount > 0 ? (
          <Tooltip content={t("dashboard.filters.resultCountTooltip")}>
            <p className="hidden cursor-default select-none text-sm tabular-nums text-muted-foreground shrink-0 min-[480px]:block">
              {shownCount} / {totalCount}
            </p>
          </Tooltip>
        ) : null}
      </div>

      <div className="flex items-center gap-2 shrink-0">
        <DropdownMenu
          modal={false}
          open={filterMenuOpen}
          onOpenChange={(open) => {
            setFilterMenuOpen(open)
            if (!open) {
              setExpandedSections(new Set())
            }
          }}
        >
          <Tooltip content={filterMenuOpen ? null : filterTooltipText} contentClassName="md:hidden">
            <DropdownMenuTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                className="relative w-8 px-0 md:w-auto md:px-3 shrink-0"
                aria-label={filterTooltipText}
              >
                <Filter className="size-4" />
                <span className="hidden md:inline">
                  {t("dashboard.filters.filterButton")}
                  {activeFilterCount > 0 ? ` (${activeFilterCount})` : ""}
                </span>
                {activeFilterCount > 0 && (
                  <span className="absolute -top-1 -right-1 flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-primary px-0.5 text-[9px] font-semibold text-primary-foreground md:hidden">
                    {activeFilterCount}
                  </span>
                )}
              </Button>
            </DropdownMenuTrigger>
          </Tooltip>
          <DropdownMenuContent
            align="end"
            className="max-w-[calc(100vw-2rem)]"
          >
            {isCompact ? (
              <div className="space-y-0.5">
                {/* Status Section */}
                <button
                  type="button"
                  onClick={(e) => {
                    e.preventDefault()
                    toggleSection("status")
                  }}
                  className="flex w-full cursor-pointer items-center justify-between rounded-sm px-2 py-1.5 text-sm font-medium outline-hidden select-none hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground"
                >
                  <span className="flex items-center gap-1.5">
                    {t("dashboard.filters.status")}
                    {statusActiveCount > 0 && (
                      <span className="flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
                        {statusActiveCount}
                      </span>
                    )}
                  </span>
                  <ChevronDown
                    className={cn(
                      "size-4 text-muted-foreground transition-transform duration-200",
                      expandedSections.has("status") && "rotate-180 text-foreground"
                    )}
                  />
                </button>
                {expandedSections.has("status") && (
                  <div className="mt-0.5 mb-1 space-y-0.5 border-l border-border/80 ml-2.5 pl-1.5">
                    {statusOptions.map((status) => (
                      <DropdownMenuCheckboxItem
                        key={status}
                        checked={selectedStatuses.has(status)}
                        onSelect={(event) => event.preventDefault()}
                        onCheckedChange={(checked) => {
                          onToggleStatus(status, checked === true)
                        }}
                      >
                        <span className="truncate">{t(`subscription.card.status.${status}`)}</span>
                      </DropdownMenuCheckboxItem>
                    ))}
                  </div>
                )}

                {/* Renewal Mode Section */}
                <button
                  type="button"
                  onClick={(e) => {
                    e.preventDefault()
                    toggleSection("renewalMode")
                  }}
                  className="flex w-full cursor-pointer items-center justify-between rounded-sm px-2 py-1.5 text-sm font-medium outline-hidden select-none hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground"
                >
                  <span className="flex items-center gap-1.5">
                    {t("dashboard.filters.renewalMode")}
                    {renewalModeActiveCount > 0 && (
                      <span className="flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
                        {renewalModeActiveCount}
                      </span>
                    )}
                  </span>
                  <ChevronDown
                    className={cn(
                      "size-4 text-muted-foreground transition-transform duration-200",
                      expandedSections.has("renewalMode") && "rotate-180 text-foreground"
                    )}
                  />
                </button>
                {expandedSections.has("renewalMode") && (
                  <div className="mt-0.5 mb-1 space-y-0.5 border-l border-border/80 ml-2.5 pl-1.5">
                    {renewalModeOptions.map((mode) => (
                      <DropdownMenuCheckboxItem
                        key={mode}
                        checked={selectedRenewalModes.has(mode)}
                        onSelect={(event) => event.preventDefault()}
                        onCheckedChange={(checked) => {
                          onToggleRenewalMode(mode, checked === true)
                        }}
                      >
                        <span className="truncate">{t(`subscription.card.renewalMode.${mode}`)}</span>
                      </DropdownMenuCheckboxItem>
                    ))}
                  </div>
                )}

                {/* Category Section */}
                <button
                  type="button"
                  onClick={(e) => {
                    e.preventDefault()
                    toggleSection("category")
                  }}
                  className="flex w-full cursor-pointer items-center justify-between rounded-sm px-2 py-1.5 text-sm font-medium outline-hidden select-none hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground"
                >
                  <span className="flex items-center gap-1.5">
                    {t("dashboard.filters.category")}
                    {categoryActiveCount > 0 && (
                      <span className="flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
                        {categoryActiveCount}
                      </span>
                    )}
                  </span>
                  <ChevronDown
                    className={cn(
                      "size-4 text-muted-foreground transition-transform duration-200",
                      expandedSections.has("category") && "rotate-180 text-foreground"
                    )}
                  />
                </button>
                {expandedSections.has("category") && (
                  <div className="mt-0.5 mb-1 space-y-0.5 border-l border-border/80 ml-2.5 pl-1.5">
                    <DropdownMenuCheckboxItem
                      checked={includeNoCategory}
                      onSelect={(event) => event.preventDefault()}
                      onCheckedChange={(checked) => {
                        onToggleNoCategory(checked === true)
                      }}
                    >
                      <span className="truncate">{t("dashboard.filters.noCategory")}</span>
                    </DropdownMenuCheckboxItem>
                    {categoryOptions.length > 0 ? (
                      categoryOptions.map((category) => (
                        <DropdownMenuCheckboxItem
                          key={category}
                          checked={selectedCategories.has(category)}
                          onSelect={(event) => event.preventDefault()}
                          onCheckedChange={(checked) => {
                            onToggleCategory(category, checked === true)
                          }}
                        >
                          <span className="truncate">{category}</span>
                        </DropdownMenuCheckboxItem>
                      ))
                    ) : (
                      <div className="px-2 py-1.5 text-sm text-muted-foreground whitespace-nowrap">
                        {t("dashboard.filters.noCategories")}
                      </div>
                    )}
                  </div>
                )}

                {/* Payment Method Section */}
                <button
                  type="button"
                  onClick={(e) => {
                    e.preventDefault()
                    toggleSection("paymentMethod")
                  }}
                  className="flex w-full cursor-pointer items-center justify-between rounded-sm px-2 py-1.5 text-sm font-medium outline-hidden select-none hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground"
                >
                  <span className="flex items-center gap-1.5">
                    {t("dashboard.filters.paymentMethod")}
                    {paymentMethodActiveCount > 0 && (
                      <span className="flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
                        {paymentMethodActiveCount}
                      </span>
                    )}
                  </span>
                  <ChevronDown
                    className={cn(
                      "size-4 text-muted-foreground transition-transform duration-200",
                      expandedSections.has("paymentMethod") && "rotate-180 text-foreground"
                    )}
                  />
                </button>
                {expandedSections.has("paymentMethod") && (
                  <div className="mt-0.5 mb-1 space-y-0.5 border-l border-border/80 ml-2.5 pl-1.5">
                    <DropdownMenuCheckboxItem
                      checked={includeNoPaymentMethod}
                      onSelect={(event) => event.preventDefault()}
                      onCheckedChange={(checked) => {
                        onToggleNoPaymentMethod(checked === true)
                      }}
                    >
                      <span className="truncate">{t("dashboard.filters.noPaymentMethod")}</span>
                    </DropdownMenuCheckboxItem>
                    {paymentMethods.length > 0 ? (
                      paymentMethods.map((method) => (
                        <DropdownMenuCheckboxItem
                          key={method.id}
                          checked={selectedPaymentMethodIDs.has(method.id)}
                          onSelect={(event) => event.preventDefault()}
                          onCheckedChange={(checked) => {
                            onTogglePaymentMethod(method.id, checked === true)
                          }}
                        >
                          <span className="truncate">{paymentMethodLabelMap.get(method.id) ?? method.name}</span>
                        </DropdownMenuCheckboxItem>
                      ))
                    ) : (
                      <div className="px-2 py-1.5 text-sm text-muted-foreground whitespace-nowrap">
                        {t("dashboard.filters.noPaymentMethods")}
                      </div>
                    )}
                  </div>
                )}
              </div>
            ) : (
              <>
                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>
                    <span className="flex-1 text-left">{t("dashboard.filters.status")}</span>
                    {statusActiveCount > 0 && (
                      <span className="mr-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
                        {statusActiveCount}
                      </span>
                    )}
                  </DropdownMenuSubTrigger>
                  <DropdownMenuSubContent>
                    {statusOptions.map((status) => (
                      <DropdownMenuCheckboxItem
                        key={status}
                        checked={selectedStatuses.has(status)}
                        onSelect={(event) => event.preventDefault()}
                        onCheckedChange={(checked) => {
                          onToggleStatus(status, checked === true)
                        }}
                      >
                        <span className="truncate">{t(`subscription.card.status.${status}`)}</span>
                      </DropdownMenuCheckboxItem>
                    ))}
                  </DropdownMenuSubContent>
                </DropdownMenuSub>

                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>
                    <span className="flex-1 text-left">{t("dashboard.filters.renewalMode")}</span>
                    {renewalModeActiveCount > 0 && (
                      <span className="mr-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
                        {renewalModeActiveCount}
                      </span>
                    )}
                  </DropdownMenuSubTrigger>
                  <DropdownMenuSubContent>
                    {renewalModeOptions.map((mode) => (
                      <DropdownMenuCheckboxItem
                        key={mode}
                        checked={selectedRenewalModes.has(mode)}
                        onSelect={(event) => event.preventDefault()}
                        onCheckedChange={(checked) => {
                          onToggleRenewalMode(mode, checked === true)
                        }}
                      >
                        <span className="truncate">{t(`subscription.card.renewalMode.${mode}`)}</span>
                      </DropdownMenuCheckboxItem>
                    ))}
                  </DropdownMenuSubContent>
                </DropdownMenuSub>

                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>
                    <span className="flex-1 text-left">{t("dashboard.filters.category")}</span>
                    {categoryActiveCount > 0 && (
                      <span className="mr-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
                        {categoryActiveCount}
                      </span>
                    )}
                  </DropdownMenuSubTrigger>
                  <DropdownMenuSubContent>
                    <DropdownMenuCheckboxItem
                      checked={includeNoCategory}
                      onSelect={(event) => event.preventDefault()}
                      onCheckedChange={(checked) => {
                        onToggleNoCategory(checked === true)
                      }}
                    >
                      <span className="truncate">{t("dashboard.filters.noCategory")}</span>
                    </DropdownMenuCheckboxItem>
                    {categoryOptions.length > 0 ? (
                      categoryOptions.map((category) => (
                        <DropdownMenuCheckboxItem
                          key={category}
                          checked={selectedCategories.has(category)}
                          onSelect={(event) => event.preventDefault()}
                          onCheckedChange={(checked) => {
                            onToggleCategory(category, checked === true)
                          }}
                        >
                          <span className="truncate">{category}</span>
                        </DropdownMenuCheckboxItem>
                      ))
                    ) : (
                      <div className="px-2 py-1.5 text-sm text-muted-foreground whitespace-nowrap">
                        {t("dashboard.filters.noCategories")}
                      </div>
                    )}
                  </DropdownMenuSubContent>
                </DropdownMenuSub>

                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>
                    <span className="flex-1 text-left">{t("dashboard.filters.paymentMethod")}</span>
                    {paymentMethodActiveCount > 0 && (
                      <span className="mr-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
                        {paymentMethodActiveCount}
                      </span>
                    )}
                  </DropdownMenuSubTrigger>
                  <DropdownMenuSubContent>
                    <DropdownMenuCheckboxItem
                      checked={includeNoPaymentMethod}
                      onSelect={(event) => event.preventDefault()}
                      onCheckedChange={(checked) => {
                        onToggleNoPaymentMethod(checked === true)
                      }}
                    >
                      <span className="truncate">{t("dashboard.filters.noPaymentMethod")}</span>
                    </DropdownMenuCheckboxItem>
                    {paymentMethods.length > 0 ? (
                      paymentMethods.map((method) => (
                        <DropdownMenuCheckboxItem
                          key={method.id}
                          checked={selectedPaymentMethodIDs.has(method.id)}
                          onSelect={(event) => event.preventDefault()}
                          onCheckedChange={(checked) => {
                            onTogglePaymentMethod(method.id, checked === true)
                          }}
                        >
                          <span className="truncate">{paymentMethodLabelMap.get(method.id) ?? method.name}</span>
                        </DropdownMenuCheckboxItem>
                      ))
                    ) : (
                      <div className="px-2 py-1.5 text-sm text-muted-foreground whitespace-nowrap">
                        {t("dashboard.filters.noPaymentMethods")}
                      </div>
                    )}
                  </DropdownMenuSubContent>
                </DropdownMenuSub>
              </>
            )}

            <DropdownMenuSeparator />
            <DropdownMenuItem
              onSelect={(event) => {
                event.preventDefault()
                onResetFiltersAndSorting()
              }}
              disabled={!hasActiveFilters}
            >
              <FilterX className="size-4" />
              {t("dashboard.filters.clearFilters")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <DropdownMenu modal={false} open={sortMenuOpen} onOpenChange={setSortMenuOpen}>
          <Tooltip content={sortMenuOpen ? null : sortTooltipText} contentClassName="md:hidden">
            <DropdownMenuTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                className="w-8 px-0 md:w-auto md:px-3 shrink-0"
                aria-label={sortTooltipText}
              >
                <ArrowUpDown className="size-4" />
                <span className="hidden md:inline">
                  {getSortFieldLabel(sortField)}
                </span>
                <span className="hidden md:inline-flex">
                  {sortDirection === "asc" ? <ArrowUp className="size-3.5" /> : <ArrowDown className="size-3.5" />}
                </span>
              </Button>
            </DropdownMenuTrigger>
          </Tooltip>
          <DropdownMenuContent align="end">
            {sortFieldOptions.map((field) => (
              <DropdownMenuItem
                key={field}
                onSelect={(event) => {
                  event.preventDefault()
                  onSortFieldSelect(field)
                }}
              >
                {getSortFieldLabel(field)}
                {sortField === field ? (
                  sortDirection === "asc" ? (
                    <ArrowUp className="ml-auto size-3.5" />
                  ) : (
                    <ArrowDown className="ml-auto size-3.5" />
                  )
                ) : null}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant={batchMode ? "secondary" : "outline"}
              size="icon-sm"
              className="shrink-0"
              onClick={onToggleBatchMode}
              disabled={!batchMode && totalCount === 0}
              aria-pressed={batchMode}
              aria-label={t(batchMode ? "subscription.batch.exit" : "subscription.batch.actions")}
            >
              {batchMode ? <X className="size-4" /> : <CheckSquare className="size-4" />}
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            {t(batchMode ? "subscription.batch.exit" : "subscription.batch.actions")}
          </TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="outline"
              size="icon-sm"
              className="shrink-0"
              onClick={onToggleSubscriptionView}
              disabled={viewToggleDisabled}
              aria-label={
                subscriptionView === "list"
                  ? t("dashboard.views.toggleToCards")
                  : t("dashboard.views.toggleToList")
              }
            >
              {subscriptionView === "list" ? <Grid3X3 className="size-4" /> : <List className="size-4" />}
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            {subscriptionView === "list"
              ? t("dashboard.views.toggleToCards")
              : t("dashboard.views.toggleToList")}
          </TooltipContent>
        </Tooltip>
      </div>
    </div>
  )
}
