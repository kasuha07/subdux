import { useState } from "react"
import { useTranslation } from "react-i18next"
import { RefreshCw } from "lucide-react"

import { formatDate } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
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
import type { ExchangeRateStatus } from "@/types"

interface AdminExchangeRatesTabProps {
  currencyApiKey: string
  currencyApiKeyConfigured: boolean
  exchangeRateSource: string
  onCurrencyApiKeyChange: (value: string) => void
  onExchangeRateSourceChange: (value: string) => void
  onRefresh: () => void | Promise<void>
  onSave: () => void | Promise<void>
  rateStatus: ExchangeRateStatus | null
  refreshing: boolean
}

export default function AdminExchangeRatesTab({
  currencyApiKey,
  currencyApiKeyConfigured,
  exchangeRateSource,
  onCurrencyApiKeyChange,
  onExchangeRateSourceChange,
  onRefresh,
  onSave,
  rateStatus,
  refreshing,
}: AdminExchangeRatesTabProps) {
  const { t, i18n } = useTranslation()
  const [editingCurrencyApiKey, setEditingCurrencyApiKey] = useState(false)
  const configuredMaskValue = "••••••••"
  const currencyApiKeyDisplayValue = editingCurrencyApiKey
    ? currencyApiKey
    : currencyApiKey || (currencyApiKeyConfigured ? configuredMaskValue : "")

  return (
    <TabsContent value="exchange-rates">
      <Card>
        <CardContent className="space-y-6 p-6">
          <div className="space-y-2">
            <Label htmlFor="currency-api-key">{t("admin.exchangeRates.apiKeyLabel")}</Label>
            <Input
              id="currency-api-key"
              value={currencyApiKeyDisplayValue}
              onFocus={() => setEditingCurrencyApiKey(true)}
              onBlur={() => setEditingCurrencyApiKey(false)}
              onChange={(event) => onCurrencyApiKeyChange(event.target.value)}
              placeholder={t("admin.exchangeRates.apiKeyPlaceholder")}
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.exchangeRates.apiKeyDescription")}
            </p>
          </div>

          <Separator />

          <div className="space-y-2">
            <Label htmlFor="exchange-rate-source">{t("admin.exchangeRates.sourceSelectLabel")}</Label>
            <Select value={exchangeRateSource} onValueChange={onExchangeRateSourceChange}>
              <SelectTrigger id="exchange-rate-source" className="w-64">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="auto">{t("admin.exchangeRates.auto")}</SelectItem>
                <SelectItem value="free">{t("admin.exchangeRates.free")}</SelectItem>
                <SelectItem value="premium">{t("admin.exchangeRates.premium")}</SelectItem>
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              {t("admin.exchangeRates.sourceSelectDescription")}
            </p>
          </div>

          <Separator />

          <div className="space-y-4">
            <div className="grid gap-4 md:grid-cols-3">
              <div className="space-y-1">
                <p className="text-sm font-medium leading-none">{t("admin.exchangeRates.source")}</p>
                <p className="text-sm text-muted-foreground">
                  {rateStatus?.source === "premium"
                    ? t("admin.exchangeRates.premium")
                    : t("admin.exchangeRates.free")}
                </p>
              </div>
              <div className="space-y-1">
                <p className="text-sm font-medium leading-none">{t("admin.exchangeRates.lastFetched")}</p>
                <p className="text-sm text-muted-foreground">
                  {rateStatus?.last_fetched_at
                    ? formatDate(rateStatus.last_fetched_at, i18n.language)
                    : t("admin.exchangeRates.never")}
                </p>
              </div>
              <div className="space-y-1">
                <p className="text-sm font-medium leading-none">{t("admin.exchangeRates.rateCount")}</p>
                <p className="text-sm text-muted-foreground">{rateStatus?.rate_count ?? 0}</p>
              </div>
            </div>

            <Button onClick={() => void onRefresh()} disabled={refreshing}>
              <RefreshCw className={`mr-2 size-4 ${refreshing ? "animate-spin" : ""}`} />
              {refreshing
                ? t("admin.exchangeRates.refreshing")
                : t("admin.exchangeRates.refresh")}
            </Button>
          </div>

          <Separator />

          <Button onClick={() => void onSave()}>{t("admin.settings.save")}</Button>
        </CardContent>
      </Card>
    </TabsContent>
  )
}
