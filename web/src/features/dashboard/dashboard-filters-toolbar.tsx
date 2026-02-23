import { useTranslation } from "react-i18next"
import {
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
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import type { PaymentMethod } from "@/types"
import {
  enabledOptions,
  sortFieldOptions,
  type EnabledFilter,
  type SortDirection,
  type SortField,
} from "./dashboard-filter-constants"

interface DashboardFiltersToolbarProps {
  categoryOptions: string[]
  getSortFieldLabel: (field: SortField) => string
  hasActiveFilters: boolean
  includeNoCategory: boolean
  includeNoPaymentMethod: boolean
  onResetFiltersAndSorting: () => void
  onSearchTermChange: (value: string) => void
  onSortFieldSelect: (field: SortField) => void
  onToggleCategory: (category: string, checked: boolean) => void
  onToggleEnabledState: (status: EnabledFilter, checked: boolean) => void
  onToggleNoCategory: (checked: boolean) => void
  onToggleNoPaymentMethod: (checked: boolean) => void
  onTogglePaymentMethod: (paymentMethodID: number, checked: boolean) => void
  paymentMethodLabelMap: Map<number, string>
  paymentMethods: PaymentMethod[]
  searchTerm: string
  subscriptionView: "list" | "cards"
  selectedCategories: Set<string>
  selectedEnabledStates: Set<EnabledFilter>
  selectedPaymentMethodIDs: Set<number>
  onToggleSubscriptionView: () => void
  viewToggleDisabled?: boolean
  sortDirection: SortDirection
  sortField: SortField
}

export default function DashboardFiltersToolbar({
  categoryOptions,
  getSortFieldLabel,
  hasActiveFilters,
  includeNoCategory,
  includeNoPaymentMethod,
  onResetFiltersAndSorting,
  onSearchTermChange,
  onSortFieldSelect,
  onToggleCategory,
  onToggleEnabledState,
  onToggleNoCategory,
  onToggleNoPaymentMethod,
  onTogglePaymentMethod,
  paymentMethodLabelMap,
  paymentMethods,
  searchTerm,
  subscriptionView,
  selectedCategories,
  selectedEnabledStates,
  selectedPaymentMethodIDs,
  onToggleSubscriptionView,
  viewToggleDisabled = false,
  sortDirection,
  sortField,
}: DashboardFiltersToolbarProps) {
  const { t } = useTranslation()

  const activeFilterCount =
    selectedEnabledStates.size +
    selectedCategories.size +
    (includeNoCategory ? 1 : 0) +
    selectedPaymentMethodIDs.size +
    (includeNoPaymentMethod ? 1 : 0)

  return (
    <div className="mb-4 flex flex-wrap items-center justify-end gap-2 lg:flex-nowrap">
      <div className="relative w-56 shrink-0 lg:w-72">
        <Search className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={searchTerm}
          onChange={(event) => onSearchTermChange(event.target.value)}
          placeholder={t("dashboard.filters.searchPlaceholder")}
          className="pl-9"
        />
      </div>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="shrink-0">
            <Filter className="size-4" />
            {t("dashboard.filters.filter")}
            {activeFilterCount > 0 ? ` (${activeFilterCount})` : ""}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-52">
          <DropdownMenuLabel>{t("dashboard.filters.filter")}</DropdownMenuLabel>
          <DropdownMenuSeparator />

          <DropdownMenuSub>
            <DropdownMenuSubTrigger>{t("dashboard.filters.status")}</DropdownMenuSubTrigger>
            <DropdownMenuSubContent className="w-48">
              {enabledOptions.map((status) => (
                <DropdownMenuCheckboxItem
                  key={status}
                  checked={selectedEnabledStates.has(status)}
                  onSelect={(event) => event.preventDefault()}
                  onCheckedChange={(checked) => {
                    onToggleEnabledState(status, checked === true)
                  }}
                >
                  {t(`subscription.card.status.${status}`)}
                </DropdownMenuCheckboxItem>
              ))}
            </DropdownMenuSubContent>
          </DropdownMenuSub>

          <DropdownMenuSub>
            <DropdownMenuSubTrigger>{t("dashboard.filters.category")}</DropdownMenuSubTrigger>
            <DropdownMenuSubContent className="w-56">
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
            <DropdownMenuSubContent className="w-56">
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

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="shrink-0">
            <ArrowUpDown className="size-4" />
            {getSortFieldLabel(sortField)}
            {sortDirection === "asc" ? <ArrowUp className="size-3.5" /> : <ArrowDown className="size-3.5" />}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-56">
          <DropdownMenuLabel>{t("dashboard.filters.sort")}</DropdownMenuLabel>
          <DropdownMenuSeparator />
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
        title={
          subscriptionView === "list"
            ? t("dashboard.views.toggleToCards")
            : t("dashboard.views.toggleToList")
        }
      >
        {subscriptionView === "list" ? <Grid3X3 className="size-4" /> : <List className="size-4" />}
      </Button>
    </div>
  )
}
