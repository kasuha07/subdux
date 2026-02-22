import { useTranslation } from "react-i18next"
import { Monitor, Moon, Sun } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { TabsContent } from "@/components/ui/tabs"
import {
  THEME_COLOR_SCHEME_OPTIONS,
  getThemeColorSchemeSwatch,
  type CustomThemeColors,
  type Theme,
  type ThemeColorScheme,
} from "@/lib/theme"

interface SettingsGeneralTabProps {
  colorScheme: ThemeColorScheme
  customThemeColors: CustomThemeColors
  displayAllAmountsInPrimaryCurrency: boolean
  displayRecurringAmountsAsMonthlyCost: boolean
  language: string
  onColorSchemeChange: (next: ThemeColorScheme) => void
  onCustomThemeColorChange: (key: keyof CustomThemeColors, value: string) => void
  onDisplayAllAmountsInPrimaryCurrencyChange: (enabled: boolean) => void
  onDisplayRecurringAmountsAsMonthlyCostChange: (enabled: boolean) => void
  onLanguageChange: (language: string) => void
  onResetCustomThemeColors: () => void
  onThemeChange: (next: Theme) => void
  theme: Theme
}

const languages = [
  { value: "en", label: "English" },
  { value: "zh-CN", label: "中文" },
  { value: "ja", label: "日本語" },
]

const customColorFields: Array<{
  id: string
  key: keyof CustomThemeColors
  labelKey: string
}> = [
  { id: "theme-light-primary", key: "light_primary", labelKey: "settings.appearance.lightPrimary" },
  { id: "theme-light-accent", key: "light_accent", labelKey: "settings.appearance.lightAccent" },
  { id: "theme-dark-primary", key: "dark_primary", labelKey: "settings.appearance.darkPrimary" },
  { id: "theme-dark-accent", key: "dark_accent", labelKey: "settings.appearance.darkAccent" },
]

export default function SettingsGeneralTab({
  colorScheme,
  customThemeColors,
  displayAllAmountsInPrimaryCurrency,
  displayRecurringAmountsAsMonthlyCost,
  language,
  onColorSchemeChange,
  onCustomThemeColorChange,
  onDisplayAllAmountsInPrimaryCurrencyChange,
  onDisplayRecurringAmountsAsMonthlyCostChange,
  onLanguageChange,
  onResetCustomThemeColors,
  onThemeChange,
  theme,
}: SettingsGeneralTabProps) {
  const { t } = useTranslation()

  return (
    <TabsContent value="general" className="space-y-6">
      <div>
        <h2 className="text-base font-semibold tracking-tight">{t("settings.displayAmount.title")}</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          {t("settings.displayAmount.description")}
        </p>
        <div className="mt-3 space-y-3">
          <div className="flex items-center gap-3">
            <Switch
              id="display-all-amounts-in-primary-currency"
              checked={displayAllAmountsInPrimaryCurrency}
              onCheckedChange={onDisplayAllAmountsInPrimaryCurrencyChange}
            />
            <Label
              htmlFor="display-all-amounts-in-primary-currency"
              className="cursor-pointer"
            >
              {t("settings.displayAmount.toggle")}
            </Label>
          </div>
          <div className="flex items-center gap-3">
            <Switch
              id="display-recurring-amounts-as-monthly-cost"
              checked={displayRecurringAmountsAsMonthlyCost}
              onCheckedChange={onDisplayRecurringAmountsAsMonthlyCostChange}
            />
            <Label
              htmlFor="display-recurring-amounts-as-monthly-cost"
              className="cursor-pointer"
            >
              {t("settings.displayAmount.monthlyCostToggle")}
            </Label>
          </div>
        </div>
      </div>

      <Separator />

      <div>
        <h2 className="text-base font-semibold tracking-tight">{t("settings.appearance.title")}</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          {t("settings.appearance.description")}
        </p>
        <div className="mt-3 flex gap-2">
          <Button
            size="sm"
            variant={theme === "light" ? "default" : "outline"}
            onClick={() => onThemeChange("light")}
          >
            <Sun className="size-4" />
            {t("settings.appearance.light")}
          </Button>
          <Button
            size="sm"
            variant={theme === "dark" ? "default" : "outline"}
            onClick={() => onThemeChange("dark")}
          >
            <Moon className="size-4" />
            {t("settings.appearance.dark")}
          </Button>
          <Button
            size="sm"
            variant={theme === "system" ? "default" : "outline"}
            onClick={() => onThemeChange("system")}
          >
            <Monitor className="size-4" />
            {t("settings.appearance.system")}
          </Button>
        </div>
      </div>

      <Separator />

      <div>
        <h2 className="text-base font-semibold tracking-tight">{t("settings.language.title")}</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          {t("settings.language.description")}
        </p>
        <div className="mt-3">
          <Select value={language} onValueChange={onLanguageChange}>
            <SelectTrigger className="w-48">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {languages.map((lang) => (
                <SelectItem key={lang.value} value={lang.value}>
                  {lang.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <Separator />

      <div>
        <h2 className="text-base font-semibold tracking-tight">{t("settings.appearance.colorSchemeTitle")}</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          {t("settings.appearance.colorSchemeDescription")}
        </p>
        <div className="mt-3 grid grid-cols-2 gap-2 lg:grid-cols-5">
          {THEME_COLOR_SCHEME_OPTIONS.map((scheme) => {
            const swatch = getThemeColorSchemeSwatch(scheme, customThemeColors)
            return (
              <Button
                key={scheme}
                type="button"
                size="sm"
                variant={colorScheme === scheme ? "default" : "outline"}
                className="w-full justify-between"
                onClick={() => onColorSchemeChange(scheme)}
              >
                <span>{t(`settings.appearance.colorSchemes.${scheme}`)}</span>
                <span className="flex items-center gap-1.5">
                  <span
                    className="size-3 rounded-full border border-black/10"
                    style={{ backgroundColor: swatch[0] }}
                  />
                  <span
                    className="size-3 rounded-full border border-black/10"
                    style={{ backgroundColor: swatch[1] }}
                  />
                </span>
              </Button>
            )
          })}
        </div>

        {colorScheme === "custom" ? (
          <div className="mt-4 space-y-3 rounded-md border bg-card/60 p-3">
            <p className="text-xs text-muted-foreground">
              {t("settings.appearance.customDescription")}
            </p>
            <div className="grid grid-cols-5 items-end gap-2">
              {customColorFields.map((field) => (
                <div key={field.id} className="space-y-1.5">
                  <Label htmlFor={field.id}>{t(field.labelKey)}</Label>
                  <Input
                    id={field.id}
                    type="color"
                    value={customThemeColors[field.key]}
                    onChange={(event) => onCustomThemeColorChange(field.key, event.target.value)}
                    className="h-10 cursor-pointer p-1"
                  />
                </div>
              ))}
              <Button
                type="button"
                size="sm"
                variant="outline"
                className="h-10"
                onClick={onResetCustomThemeColors}
              >
                {t("settings.appearance.resetCustom")}
              </Button>
            </div>
          </div>
        ) : null}
      </div>
    </TabsContent>
  )
}
