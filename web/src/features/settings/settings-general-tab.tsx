import { useTranslation } from "react-i18next"
import { Monitor, Moon, Sun } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { TabsContent } from "@/components/ui/tabs"
import type { Theme } from "@/lib/theme"

interface SettingsGeneralTabProps {
  language: string
  onLanguageChange: (language: string) => void
  onThemeChange: (next: Theme) => void
  theme: Theme
}

const languages = [
  { value: "en", label: "English" },
  { value: "zh-CN", label: "中文" },
  { value: "ja", label: "日本語" },
]

export default function SettingsGeneralTab({
  language,
  onLanguageChange,
  onThemeChange,
  theme,
}: SettingsGeneralTabProps) {
  const { t } = useTranslation()

  return (
    <TabsContent value="general" className="space-y-6">
      <div>
        <h2 className="text-sm font-medium">{t("settings.appearance.title")}</h2>
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
        <h2 className="text-sm font-medium">{t("settings.language.title")}</h2>
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
    </TabsContent>
  )
}
