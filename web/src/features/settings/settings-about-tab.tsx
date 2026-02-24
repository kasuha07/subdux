import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { ExternalLink, GitCommitHorizontal, Info, Scale } from "lucide-react"

import { TabsContent } from "@/components/ui/tabs"
import { Separator } from "@/components/ui/separator"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { api } from "@/lib/api"
import type { VersionInfo } from "@/types"

const githubRepoURL = "https://github.com/kasuha07/subdux"

interface SettingsAboutTabProps {
  versionInfo: VersionInfo | null
}

export default function SettingsAboutTab({ versionInfo }: SettingsAboutTabProps) {
  const { t } = useTranslation()
  const [latestVersion, setLatestVersion] = useState<string | null>(null)
  const [latestCheckStatus, setLatestCheckStatus] = useState<"idle" | "checking" | "done" | "error">("idle")

  useEffect(() => {
    if (!versionInfo) return
    api
      .get<{ tag_name: string }>(`/version/latest`)
      .then((data) => {
        setLatestVersion(data.tag_name.replace(/^v/, ""))
        setLatestCheckStatus("done")
      })
      .catch(() => {
        setLatestCheckStatus("error")
      })
  }, [versionInfo])

  const isNewVersionAvailable =
    latestCheckStatus === "done" &&
    latestVersion &&
    versionInfo &&
    latestVersion !== versionInfo.version.replace(/^v/, "")

  return (
    <TabsContent value="about" className="space-y-6">
      {/* Version Info */}
      <div className="space-y-3">
        <div>
          <h2 className="flex items-center gap-2 text-base font-semibold tracking-tight">
            <Info className="size-4" />
            {t("settings.about.currentVersion")}
          </h2>
          <p className="text-sm text-muted-foreground">{t("settings.about.description")}</p>
        </div>
        {versionInfo ? (
          <div className="grid gap-3 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">{t("settings.about.currentVersion")}</span>
              <Badge variant="secondary">{versionInfo.version}</Badge>
            </div>
            {versionInfo.commit !== "unknown" && (
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">{t("settings.about.commit")}</span>
                <code className="text-xs">{versionInfo.commit}</code>
              </div>
            )}
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">{t("settings.about.buildDate")}</span>
              <span>{versionInfo.build_date}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">{t("settings.about.goVersion")}</span>
              <span>{versionInfo.go_version}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">{t("settings.about.latestVersion")}</span>
              <span>
                {latestCheckStatus === "checking" && (
                  <span className="text-muted-foreground">{t("settings.about.checking")}</span>
                )}
                {latestCheckStatus === "idle" && (
                  <span className="text-muted-foreground">{t("settings.about.checking")}</span>
                )}
                {latestCheckStatus === "error" && (
                  <span className="text-muted-foreground">{t("settings.about.checkFailed")}</span>
                )}
                {latestCheckStatus === "done" && latestVersion && (
                  <span className="flex items-center gap-2">
                    <Badge variant={isNewVersionAvailable ? "default" : "secondary"}>
                      {latestVersion}
                    </Badge>
                    {isNewVersionAvailable ? (
                      <a
                        href={`${githubRepoURL}/releases/latest`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-xs text-primary underline underline-offset-4"
                      >
                        {t("settings.about.newVersionAvailable")}
                      </a>
                    ) : (
                      <span className="text-xs text-muted-foreground">
                        {t("settings.about.upToDate")}
                      </span>
                    )}
                  </span>
                )}
              </span>
            </div>
          </div>
        ) : (
          <div className="text-sm text-muted-foreground">{t("settings.about.checking")}</div>
        )}
      </div>

      <Separator />

      {/* Feedback & Source */}
      <div className="space-y-3">
        <div>
          <h2 className="flex items-center gap-2 text-base font-semibold tracking-tight">
            <GitCommitHorizontal className="size-4" />
            {t("settings.about.feedback")}
          </h2>
          <p className="text-sm text-muted-foreground">{t("settings.about.feedbackDescription")}</p>
        </div>
        <div className="flex flex-wrap gap-3">
          <Button variant="outline" size="sm" asChild>
            <a href={`${githubRepoURL}/issues`} target="_blank" rel="noopener noreferrer">
              <ExternalLink className="mr-2 size-4" />
              {t("settings.about.openIssue")}
            </a>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <a href={githubRepoURL} target="_blank" rel="noopener noreferrer">
              <ExternalLink className="mr-2 size-4" />
              {t("settings.about.sourceCode")}
            </a>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <a href={`${githubRepoURL}/blob/main/LICENSE`} target="_blank" rel="noopener noreferrer">
              <Scale className="mr-2 size-4" />
              {t("settings.about.license")}
            </a>
          </Button>
        </div>
      </div>

      <Separator />

      {/* Acknowledgments */}
      <div className="space-y-4">
        <div>
          <h2 className="text-base font-semibold tracking-tight">{t("settings.about.acknowledgments")}</h2>
          <p className="text-sm text-muted-foreground">{t("settings.about.acknowledgmentsDescription")}</p>
        </div>
        <div>
          <h4 className="mb-2 text-sm font-medium">{t("settings.about.backendTech")}</h4>
          <div className="flex flex-wrap gap-2">
            {["Go", "Echo", "GORM", "SQLite"].map((tech) => (
              <Badge key={tech} variant="outline">{tech}</Badge>
            ))}
          </div>
        </div>
        <div>
          <h4 className="mb-2 text-sm font-medium">{t("settings.about.frontendTech")}</h4>
          <div className="flex flex-wrap gap-2">
            {["React", "TypeScript", "Vite", "Tailwind CSS", "shadcn/ui", "Lucide Icons", "IconGo", "i18next"].map((tech) => (
              <Badge key={tech} variant="outline">{tech}</Badge>
            ))}
          </div>
        </div>
      </div>
    </TabsContent>
  )
}
