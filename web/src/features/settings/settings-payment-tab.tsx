import { type DragEvent, type FormEvent } from "react"
import { useTranslation } from "react-i18next"
import { GripVertical, Trash2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { TabsContent } from "@/components/ui/tabs"
import type { UpdateCurrencyInput, UserCurrency } from "@/types"

import CategoryManagement from "./category-management"
import PaymentMethodManagement from "./payment-method-management"

interface SettingsPaymentTabProps {
  addAlias: string
  addAliasPlaceholder: string
  addCode: string
  addLoading: boolean
  addSymbol: string
  addSymbolPlaceholder: string
  addableCurrencyCodes: string[]
  currency: string
  customCode: string
  customCodeOption: string
  getCurrencyAliasPlaceholder: (code: string) => string
  getCurrencySymbolPlaceholder: (code: string) => string
  onAddAliasChange: (value: string) => void
  onAddCodeChange: (value: string) => void
  onAddCurrency: (event: FormEvent<HTMLFormElement>) => void | Promise<void>
  onAddSymbolChange: (value: string) => void
  onCurrencyChange: (value: string) => void | Promise<void>
  onCustomCodeChange: (value: string) => void
  onDeleteCurrency: (id: number) => void | Promise<void>
  onDragOver: (event: DragEvent<HTMLDivElement>, index: number) => void
  onDragStart: (index: number) => void
  onDrop: () => void
  onSaveOrder: () => void | Promise<void>
  onUpdateCurrency: (id: number, input: UpdateCurrencyInput) => void | Promise<void>
  orderChanged: boolean
  orderSaving: boolean
  preferredCurrencyCodes: string[]
  userCurrencies: UserCurrency[]
}

export default function SettingsPaymentTab({
  addAlias,
  addAliasPlaceholder,
  addCode,
  addLoading,
  addSymbol,
  addSymbolPlaceholder,
  addableCurrencyCodes,
  currency,
  customCode,
  customCodeOption,
  getCurrencyAliasPlaceholder,
  getCurrencySymbolPlaceholder,
  onAddAliasChange,
  onAddCodeChange,
  onAddCurrency,
  onAddSymbolChange,
  onCurrencyChange,
  onCustomCodeChange,
  onDeleteCurrency,
  onDragOver,
  onDragStart,
  onDrop,
  onSaveOrder,
  onUpdateCurrency,
  orderChanged,
  orderSaving,
  preferredCurrencyCodes,
  userCurrencies,
}: SettingsPaymentTabProps) {
  const { t } = useTranslation()

  return (
    <TabsContent value="payment" className="space-y-6">
      <div>
        <h2 className="text-base font-semibold tracking-tight">{t("settings.currency.title")}</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          {t("settings.currency.description")}
        </p>
        <div className="mt-3">
          <Select value={currency} onValueChange={(value) => void onCurrencyChange(value)}>
            <SelectTrigger className="w-48">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {preferredCurrencyCodes.map((item) => (
                <SelectItem key={item} value={item}>
                  {item}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <Separator />

      <div>
        <h2 className="text-base font-semibold tracking-tight">{t("settings.currencyManagement.title")}</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          {t("settings.currencyManagement.description")}
        </p>

        <div className="mt-3 space-y-1">
          {userCurrencies.length === 0 && (
            <p className="py-2 text-sm text-muted-foreground">{t("settings.currencyManagement.empty")}</p>
          )}

          {userCurrencies.map((item, index) => (
            <div
              key={item.id}
              draggable
              onDragStart={() => onDragStart(index)}
              onDragOver={(event) => onDragOver(event, index)}
              onDrop={onDrop}
              className="grid grid-cols-[1rem_3rem_5rem_minmax(0,1fr)_1.75rem] items-center gap-2 rounded-md border bg-card px-2 py-1.5"
            >
              <GripVertical className="size-4 shrink-0 cursor-grab text-muted-foreground" />
              <span className="inline-flex h-7 items-center font-mono text-sm font-medium">{item.code}</span>

              <Input
                className="h-7 w-full px-2 text-sm"
                placeholder={getCurrencySymbolPlaceholder(item.code)}
                defaultValue={item.symbol}
                maxLength={10}
                onBlur={(event) => {
                  if (event.target.value !== item.symbol) {
                    void onUpdateCurrency(item.id, { symbol: event.target.value })
                  }
                }}
              />

              <Input
                className="h-7 w-full px-2 text-sm"
                placeholder={getCurrencyAliasPlaceholder(item.code)}
                defaultValue={item.alias}
                maxLength={100}
                onBlur={(event) => {
                  if (event.target.value !== item.alias) {
                    void onUpdateCurrency(item.id, { alias: event.target.value })
                  }
                }}
              />

              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7 text-muted-foreground hover:text-destructive"
                onClick={() => void onDeleteCurrency(item.id)}
              >
                <Trash2 className="size-3.5" />
              </Button>
            </div>
          ))}
        </div>

        {orderChanged && (
          <Button
            size="sm"
            variant="outline"
            className="mt-2"
            disabled={orderSaving}
            onClick={() => void onSaveOrder()}
          >
            {orderSaving
              ? t("settings.currencyManagement.savingOrder")
              : t("settings.currencyManagement.saveOrder")}
          </Button>
        )}

        <form onSubmit={(event) => void onAddCurrency(event)} className="mt-3 space-y-2">
          <div className="grid gap-1 sm:grid-cols-[6rem_5rem_minmax(0,1fr)_auto]">
            <Label className="text-xs text-muted-foreground">
              {t("settings.currencyManagement.codeLabel")}
            </Label>
            <Label className="text-xs text-muted-foreground">
              {t("settings.currencyManagement.symbolLabel")}
            </Label>
            <Label className="text-xs text-muted-foreground">
              {t("settings.currencyManagement.aliasLabel")}
            </Label>
            <Label className="text-xs text-transparent">
              {t("settings.currencyManagement.addButton")}
            </Label>
          </div>

          <div className="grid items-center gap-2 sm:grid-cols-[6rem_5rem_minmax(0,1fr)_auto]">
            <Select value={addCode} onValueChange={onAddCodeChange}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder={t("settings.currencyManagement.codePlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                {addableCurrencyCodes.map((code) => (
                  <SelectItem key={code} value={code}>
                    {code}
                  </SelectItem>
                ))}
                <SelectItem value={customCodeOption}>
                  {t("settings.currencyManagement.customCode")}
                </SelectItem>
              </SelectContent>
            </Select>

            <Input
              className="w-full text-sm"
              placeholder={addSymbolPlaceholder}
              value={addSymbol}
              onChange={(event) => onAddSymbolChange(event.target.value)}
              maxLength={10}
            />

            <Input
              className="w-full text-sm"
              placeholder={addAliasPlaceholder}
              value={addAlias}
              onChange={(event) => onAddAliasChange(event.target.value)}
              maxLength={100}
            />

            <Button
              type="submit"
              className="sm:min-w-20"
              disabled={
                addLoading ||
                (addCode === customCodeOption ? customCode.trim() === "" : addCode.trim() === "")
              }
            >
              {addLoading
                ? t("settings.currencyManagement.adding")
                : t("settings.currencyManagement.addButton")}
            </Button>
          </div>

          {addCode === customCodeOption && (
            <div className="grid gap-2 sm:max-w-72">
              <div className="space-y-1">
                <Label className="text-xs text-muted-foreground">
                  {t("settings.currencyManagement.customCode")}
                </Label>
                <Input
                  className="w-full text-sm uppercase"
                  placeholder={t("settings.currencyManagement.codePlaceholder")}
                  value={customCode}
                  onChange={(event) => onCustomCodeChange(event.target.value.toUpperCase())}
                  maxLength={10}
                />
              </div>
            </div>
          )}
        </form>
      </div>

      <Separator />

      <PaymentMethodManagement />

      <Separator />

      <CategoryManagement />
    </TabsContent>
  )
}
