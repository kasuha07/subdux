import { useTranslation } from "react-i18next"
import { Loader2, ShieldCheck, ShieldX } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"

import type { AdminSettingsSSRFSectionProps } from "./admin-settings-types"

export default function AdminSettingsSSRFSection({
  onSSRFAllowPrivateIPChange,
  onSSRFDomainFilterListChange,
  onSSRFDomainFilterModeChange,
  onSSRFFilterResolvedIPsChange,
  onSSRFIPFilterListChange,
  onSSRFIPFilterModeChange,
  onSSRFProtectionEnabledChange,
  onSSRFTest,
  onSSRFTestTargetChange,
  ssrfAllowPrivateIP,
  ssrfDomainFilterList,
  ssrfDomainFilterMode,
  ssrfFilterResolvedIPs,
  ssrfIPFilterList,
  ssrfIPFilterMode,
  ssrfProtectionEnabled,
  ssrfTestResult,
  ssrfTestTarget,
  ssrfTesting,
}: AdminSettingsSSRFSectionProps) {
  const { t } = useTranslation()

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4">
        <div className="space-y-0.5">
          <Label htmlFor="ssrf-protection-enabled">{t("admin.settings.ssrfProtectionEnabled")}</Label>
          <p className="text-sm text-muted-foreground">
            {t("admin.settings.ssrfProtectionEnabledDescription")}
          </p>
        </div>
        <Switch
          id="ssrf-protection-enabled"
          checked={ssrfProtectionEnabled}
          onCheckedChange={onSSRFProtectionEnabledChange}
        />
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="space-y-0.5">
          <Label htmlFor="ssrf-allow-private-ip">{t("admin.settings.ssrfAllowPrivateIP")}</Label>
          <p className="text-sm text-muted-foreground">
            {t("admin.settings.ssrfAllowPrivateIPDescription")}
          </p>
        </div>
        <Switch
          id="ssrf-allow-private-ip"
          checked={ssrfAllowPrivateIP}
          onCheckedChange={onSSRFAllowPrivateIPChange}
        />
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="ssrf-domain-filter-mode">{t("admin.settings.ssrfDomainFilterMode")}</Label>
          <Select value={ssrfDomainFilterMode} onValueChange={onSSRFDomainFilterModeChange}>
            <SelectTrigger id="ssrf-domain-filter-mode" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="blacklist">{t("admin.settings.ssrfFilterModeBlacklist")}</SelectItem>
              <SelectItem value="whitelist">{t("admin.settings.ssrfFilterModeWhitelist")}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="ssrf-ip-filter-mode">{t("admin.settings.ssrfIPFilterMode")}</Label>
          <Select value={ssrfIPFilterMode} onValueChange={onSSRFIPFilterModeChange}>
            <SelectTrigger id="ssrf-ip-filter-mode" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="blacklist">{t("admin.settings.ssrfFilterModeBlacklist")}</SelectItem>
              <SelectItem value="whitelist">{t("admin.settings.ssrfFilterModeWhitelist")}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="ssrf-domain-filter-list">{t("admin.settings.ssrfDomainFilterList")}</Label>
          <Textarea
            id="ssrf-domain-filter-list"
            value={ssrfDomainFilterList}
            onChange={(event) => onSSRFDomainFilterListChange(event.target.value)}
            placeholder={t("admin.settings.ssrfDomainFilterListPlaceholder")}
            rows={4}
          />
          <p className="text-xs text-muted-foreground">
            {t("admin.settings.ssrfDomainFilterListDescription")}
          </p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="ssrf-ip-filter-list">{t("admin.settings.ssrfIPFilterList")}</Label>
          <Textarea
            id="ssrf-ip-filter-list"
            value={ssrfIPFilterList}
            onChange={(event) => onSSRFIPFilterListChange(event.target.value)}
            placeholder={t("admin.settings.ssrfIPFilterListPlaceholder")}
            rows={4}
          />
          <p className="text-xs text-muted-foreground">
            {t("admin.settings.ssrfIPFilterListDescription")}
          </p>
        </div>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="space-y-0.5">
          <Label htmlFor="ssrf-filter-resolved-ips">
            {t("admin.settings.ssrfFilterResolvedIPs")}
          </Label>
          <p className="text-sm text-muted-foreground">
            {t("admin.settings.ssrfFilterResolvedIPsDescription")}
          </p>
        </div>
        <Switch
          id="ssrf-filter-resolved-ips"
          checked={ssrfFilterResolvedIPs}
          onCheckedChange={onSSRFFilterResolvedIPsChange}
        />
      </div>

      <div className="space-y-3 rounded-md border p-4">
        <div className="space-y-1">
          <Label htmlFor="ssrf-test-target">{t("admin.settings.ssrfTestTarget")}</Label>
          <p className="text-sm text-muted-foreground">
            {t("admin.settings.ssrfTestTargetDescription")}
          </p>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input
            id="ssrf-test-target"
            value={ssrfTestTarget}
            onChange={(event) => onSSRFTestTargetChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault()
                void onSSRFTest()
              }
            }}
            placeholder={t("admin.settings.ssrfTestTargetPlaceholder")}
          />
          <Button
            type="button"
            variant="outline"
            onClick={() => void onSSRFTest()}
            disabled={ssrfTesting}
            className="sm:w-32"
          >
            {ssrfTesting ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <ShieldCheck className="size-4" />
            )}
            {ssrfTesting ? t("admin.settings.ssrfTesting") : t("admin.settings.ssrfTestButton")}
          </Button>
        </div>

        {ssrfTestResult && (
          <div className="space-y-3 rounded-md bg-muted/40 p-3 text-sm">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={ssrfTestResult.allowed ? "default" : "destructive"}>
                {ssrfTestResult.allowed ? (
                  <ShieldCheck className="size-3" />
                ) : (
                  <ShieldX className="size-3" />
                )}
                {ssrfTestResult.allowed
                  ? t("admin.settings.ssrfTestAllowed")
                  : t("admin.settings.ssrfTestBlocked")}
              </Badge>
              <span className="break-all text-muted-foreground">{ssrfTestResult.reason}</span>
            </div>

            <dl className="grid gap-2 sm:grid-cols-2">
              <div>
                <dt className="text-xs text-muted-foreground">{t("admin.settings.ssrfTestHost")}</dt>
                <dd className="break-all font-medium">{ssrfTestResult.host}</dd>
              </div>
              <div>
                <dt className="text-xs text-muted-foreground">{t("admin.settings.ssrfTestMode")}</dt>
                <dd className="font-medium">
                  {ssrfTestResult.proxy_mediated
                    ? t("admin.settings.ssrfTestModeProxy")
                    : t("admin.settings.ssrfTestModeDirect")}
                </dd>
              </div>
              <div>
                <dt className="text-xs text-muted-foreground">
                  {t("admin.settings.ssrfTestResolvedIPs")}
                </dt>
                <dd className="break-all font-medium">
                  {(ssrfTestResult.resolved_ips?.length ?? 0) > 0
                    ? ssrfTestResult.resolved_ips.join(", ")
                    : t("admin.settings.ssrfTestNoResolvedIPs")}
                </dd>
              </div>
              <div>
                <dt className="text-xs text-muted-foreground">
                  {t("admin.settings.ssrfTestResolvedIPFilter")}
                </dt>
                <dd className="font-medium">
                  {ssrfTestResult.resolved_ip_filter_applied
                    ? t("admin.settings.ssrfTestApplied")
                    : t("admin.settings.ssrfTestNotApplied")}
                </dd>
              </div>
            </dl>
          </div>
        )}
      </div>
    </div>
  )
}
