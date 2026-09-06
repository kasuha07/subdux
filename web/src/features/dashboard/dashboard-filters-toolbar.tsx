import { useEffect, useRef } from "react"
import { useTranslation } from "react-i18next"
import {
  CheckSquare,
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

  const activeFilterCount =
    (selectedStatuses.size === 1 && selectedStatuses.has("active") ? 0 : 1) +
    selectedCategories.size +
    (includeNoCategory ? 1 : 0) +
    selectedPaymentMethodIDs.size +
    selectedRenewalModes.size +
    (includeNoPaymentMethod ? 1 : 0)

  return (
    <div className="mb-6 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <div className="flex min-w-0 flex-1 flex-col gap-2 sm:flex-row sm:items-center">
        <div className="relative w-full max-w-md lg:max-w-sm">
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
            className={cn("pl-9", searchTerm ? "pr-8" : "pr-8 sm:pr-9")}
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
          ) : (
            <kbd className="pointer-events-none absolute top-1/2 right-2.5 hidden -translate-y-1/2 select-none items-center rounded border border-border/80 bg-muted/60 px-1.5 font-mono text-[10px] font-medium text-muted-foreground sm:inline-flex">
              /
            </kbd>
          )}
        </div>

        {totalCount > 0 ? (
          <p className="hidden text-sm text-muted-foreground sm:block">
            {t("dashboard.filters.resultCount", { shown: shownCount, total: totalCount })}
          </p>
        ) : null}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <DropdownMenu modal={false}>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="shrink-0">
              <Filter className="size-4" />
              {t("dashboard.filters.filterButton")}
              {activeFilterCount > 0 ? ` (${activeFilterCount})` : ""}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>{t("dashboard.filters.status")}</DropdownMenuSubTrigger>
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
                    {t(`subscription.card.status.${status}`)}
                  </DropdownMenuCheckboxItem>
                ))}
              </DropdownMenuSubContent>
            </DropdownMenuSub>

            <DropdownMenuSub>
              <DropdownMenuSubTrigger>{t("dashboard.filters.renewalMode")}</DropdownMenuSubTrigger>
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
                    {t(`subscription.card.renewalMode.${mode}`)}
                  </DropdownMenuCheckboxItem>
                ))}
              </DropdownMenuSubContent>
            </DropdownMenuSub>

            <DropdownMenuSub>
              <DropdownMenuSubTrigger>{t("dashboard.filters.category")}</DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                <DropdownMenuCheckboxItem
                  checked={includeNoCategory}
                  onSelect={(event) => event.preventDefault()}
                  onCheckedChange={(checked) => {
                    onToggleNoCategory(checked === true)
                  }}
                >
                  {t("dashboard.filters.noCategory")}
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
                      {category}
                    </DropdownMenuCheckboxItem>
                  ))
                ) : (
                  <div className="px-2 py-1.5 text-sm text-muted-foreground">
                    {t("dashboard.filters.noCategories")}
                  </div>
                )}
              </DropdownMenuSubContent>
            </DropdownMenuSub>

            <DropdownMenuSub>
              <DropdownMenuSubTrigger>{t("dashboard.filters.paymentMethod")}</DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                <DropdownMenuCheckboxItem
                  checked={includeNoPaymentMethod}
                  onSelect={(event) => event.preventDefault()}
                  onCheckedChange={(checked) => {
                    onToggleNoPaymentMethod(checked === true)
                  }}
                >
                  {t("dashboard.filters.noPaymentMethod")}
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
                      {paymentMethodLabelMap.get(method.id) ?? method.name}
                    </DropdownMenuCheckboxItem>
                  ))
                ) : (
                  <div className="px-2 py-1.5 text-sm text-muted-foreground">
                    {t("dashboard.filters.noPaymentMethods")}
                  </div>
                )}
              </DropdownMenuSubContent>
            </DropdownMenuSub>

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

        <DropdownMenu modal={false}>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="shrink-0">
              <ArrowUpDown className="size-4" />
              {getSortFieldLabel(sortField)}
              {sortDirection === "asc" ? <ArrowUp className="size-3.5" /> : <ArrowDown className="size-3.5" />}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
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

        {totalCount > 0 ? (
          <p className="ml-auto text-sm text-muted-foreground sm:hidden">
            {t("dashboard.filters.resultCount", { shown: shownCount, total: totalCount })}
          </p>
        ) : null}
      </div>
    </div>
  )
}
