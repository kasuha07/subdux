import { type ChangeEvent, type RefObject } from "react"
import { useTranslation } from "react-i18next"
import { AlertTriangle, Download } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import type {
  ImportPreview,
  SubduxImportPreview,
} from "@/features/settings/settings-account-import-types"

interface SettingsAccountTransferSectionProps {
  exportLoading: boolean
  exportSecretsConfirmOpen: boolean
  importFileRef: RefObject<HTMLInputElement | null>
  importLoading: boolean
  onExport: (includeSecrets?: boolean) => void | Promise<void>
  onExportSecretsConfirmOpenChange: (open: boolean) => void
  onImportSubdux: (event: ChangeEvent<HTMLInputElement>) => void | Promise<void>
  onImportWallos: (event: ChangeEvent<HTMLInputElement>) => void | Promise<void>
  subduxImportFileRef: RefObject<HTMLInputElement | null>
  subduxImportLoading: boolean
}

interface ImportPreviewDialogProps {
  loading: boolean
  onConfirm: () => void | Promise<void>
  onOpenChange: (open: boolean) => void
  onReset: () => void
  open: boolean
  preview: ImportPreview | null
}

interface SubduxImportPreviewDialogProps {
  loading: boolean
  onConfirm: () => void | Promise<void>
  onOpenChange: (open: boolean) => void
  onReset: () => void
  open: boolean
  preview: SubduxImportPreview | null
}

export function SettingsAccountTransferSection({
  exportLoading,
  exportSecretsConfirmOpen,
  importFileRef,
  importLoading,
  onExport,
  onExportSecretsConfirmOpenChange,
  onImportSubdux,
  onImportWallos,
  subduxImportFileRef,
  subduxImportLoading,
}: SettingsAccountTransferSectionProps) {
  const { t } = useTranslation()

  return (
    <>
      <div>
        <h3 className="text-base font-semibold tracking-tight">{t("settings.account.exportTitle")}</h3>
        <p className="mt-0.5 text-sm text-muted-foreground">
          {t("settings.account.exportDescription")}
        </p>
        <div className="mt-2 flex flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={exportLoading}
            onClick={() => void onExport(false)}
          >
            <Download className="size-4" />
            {exportLoading
              ? t("settings.account.exporting")
              : t("settings.account.exportButton")}
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={exportLoading}
            onClick={() => onExportSecretsConfirmOpenChange(true)}
          >
            <Download className="size-4" />
            {t("settings.account.exportWithSecretsButton")}
          </Button>
        </div>
        {exportSecretsConfirmOpen && (
          <div className="mt-3 rounded-md border border-destructive bg-destructive/10 p-3 text-sm">
            <div className="flex items-start gap-2 font-medium text-destructive">
              <AlertTriangle className="mt-0.5 size-4 shrink-0" />
              <span>{t("settings.account.exportWithSecretsConfirmTitle")}</span>
            </div>
            <p className="mt-2 text-muted-foreground">
              {t("settings.account.exportWithSecretsConfirmDescription")}
            </p>
            <div className="mt-3 flex flex-wrap gap-2">
              <Button
                size="sm"
                variant="destructive"
                disabled={exportLoading}
                onClick={() => void onExport(true)}
              >
                {exportLoading
                  ? t("settings.account.exporting")
                  : t("settings.account.exportWithSecretsConfirmButton")}
              </Button>
              <Button
                size="sm"
                variant="outline"
                disabled={exportLoading}
                onClick={() => onExportSecretsConfirmOpenChange(false)}
              >
                {t("settings.account.exportWithSecretsCancelButton")}
              </Button>
            </div>
          </div>
        )}
      </div>

      <div className="mt-3">
        <h3 className="text-base font-semibold tracking-tight">{t("settings.account.subduxImportTitle")}</h3>
        <p className="mt-0.5 text-sm text-muted-foreground">
          {t("settings.account.subduxImportDescription")}
        </p>
        <input
          ref={subduxImportFileRef}
          type="file"
          accept=".json"
          className="hidden"
          onChange={(event) => void onImportSubdux(event)}
        />
        <Button
          variant="outline"
          size="sm"
          className="mt-2"
          disabled={subduxImportLoading}
          onClick={() => subduxImportFileRef.current?.click()}
        >
          {subduxImportLoading
            ? t("settings.account.subduxImportAnalyzing")
            : t("settings.account.subduxImportButton")}
        </Button>
      </div>

      <div className="mt-3">
        <h3 className="text-base font-semibold tracking-tight">{t("settings.account.importTitle")}</h3>
        <p className="mt-0.5 text-sm text-muted-foreground">
          {t("settings.account.importDescription")}
        </p>
        <input
          ref={importFileRef}
          type="file"
          accept=".json"
          className="hidden"
          onChange={(event) => void onImportWallos(event)}
        />
        <Button
          variant="outline"
          size="sm"
          className="mt-2"
          disabled={importLoading}
          onClick={() => importFileRef.current?.click()}
        >
          {importLoading
            ? t("settings.account.importAnalyzing")
            : t("settings.account.importButton")}
        </Button>
      </div>
    </>
  )
}

export function SubduxImportPreviewDialog({
  loading,
  onConfirm,
  onOpenChange,
  onReset,
  open,
  preview,
}: SubduxImportPreviewDialogProps) {
  const { t } = useTranslation()

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => {
      if (nextOpen) {
        onOpenChange(true)
        return
      }
      if (!loading) {
        onReset()
      }
    }}>
      <DialogContent
        className="flex max-h-[calc(100vh-1.5rem)] flex-col gap-0 overflow-hidden p-0 sm:max-h-[85vh] sm:max-w-2xl"
        onInteractOutside={(event) => event.preventDefault()}
        onEscapeKeyDown={(event) => { if (loading) event.preventDefault() }}
        showCloseButton={false}
      >
        <DialogHeader className="border-b px-5 pt-5 pb-4 sm:px-6">
          <DialogTitle>{t("settings.account.subduxImportPreviewTitle")}</DialogTitle>
          <DialogDescription>{t("settings.account.subduxImportPreviewDescription")}</DialogDescription>
        </DialogHeader>

        <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4 sm:px-6">
          {preview && (
            <div className="space-y-5">
              <div className="grid grid-cols-2 gap-2 text-sm sm:grid-cols-3">
                <PreviewCount label={t("settings.account.importPreviewCurrencies")} value={preview.currencies.length} />
                <PreviewCount label={t("settings.account.importPreviewCategories")} value={preview.categories.length} />
                <PreviewCount label={t("settings.account.importPreviewPaymentMethods")} value={preview.payment_methods.length} />
                <PreviewCount label={t("settings.account.importPreviewSubscriptions")} value={preview.subscriptions.length} />
                <PreviewCount label={t("settings.account.subduxImportChannels")} value={preview.channels.length} />
                <PreviewCount label={t("settings.account.subduxImportTemplates")} value={preview.templates.length} />
              </div>

              {(preview.preference || preview.policy) && (
                <div className="space-y-2">
                  {preview.preference && (
                    <PreviewUpdateRow
                      label={t("settings.account.subduxImportPreference")}
                      changed={preview.preference.will_create || preview.preference.will_update}
                    />
                  )}
                  {preview.policy && (
                    <PreviewUpdateRow
                      label={t("settings.account.subduxImportPolicy")}
                      changed={preview.policy.will_create || preview.policy.will_update}
                    />
                  )}
                </div>
              )}

              {preview.subscriptions.length > 0 && (
                <SubscriptionPreviewList subscriptions={preview.subscriptions} />
              )}

              {preview.channels.length > 0 && (
                <div>
                  <h4 className="mb-2 text-sm font-medium">{t("settings.account.subduxImportChannels")}</h4>
                  <div className="space-y-1.5">
                    {preview.channels.map((channel) => (
                      <div key={`${channel.type}-${channel.config}`} className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
                        <span>{channel.type}</span>
                        <Badge variant={channel.is_new ? "default" : "secondary"} className="text-xs">
                          {channel.is_new ? t("settings.account.importPreviewNew") : t("settings.account.importPreviewExists")}
                        </Badge>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {preview.templates.length > 0 && (
                <div>
                  <h4 className="mb-2 text-sm font-medium">{t("settings.account.subduxImportTemplates")}</h4>
                  <div className="space-y-1.5">
                    {preview.templates.map((template, index) => (
                      <div key={`${template.channel_type}-${template.format}-${index}`} className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
                        <span>
                          {template.channel_type ? `${template.channel_type} / ` : ""}
                          {template.format}
                        </span>
                        <Badge variant={template.is_new ? "default" : "secondary"} className="text-xs">
                          {template.is_new ? t("settings.account.importPreviewNew") : t("settings.account.importPreviewExists")}
                        </Badge>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        <PreviewDialogFooter
          loading={loading}
          loadingLabel={t("settings.account.subduxImporting")}
          onConfirm={onConfirm}
          onReset={onReset}
          confirmLabel={t("settings.account.subduxImportPreviewConfirm")}
        />
      </DialogContent>
    </Dialog>
  )
}

export function ImportPreviewDialog({
  loading,
  onConfirm,
  onOpenChange,
  onReset,
  open,
  preview,
}: ImportPreviewDialogProps) {
  const { t } = useTranslation()

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => {
      if (nextOpen) {
        onOpenChange(true)
        return
      }
      if (!loading) {
        onReset()
      }
    }}>
      <DialogContent
        className="flex max-h-[calc(100vh-1.5rem)] flex-col gap-0 overflow-hidden p-0 sm:max-h-[85vh] sm:max-w-2xl"
        onInteractOutside={(event) => event.preventDefault()}
        onEscapeKeyDown={(event) => { if (loading) event.preventDefault() }}
        showCloseButton={false}
      >
        <DialogHeader className="border-b px-5 pt-5 pb-4 sm:px-6">
          <DialogTitle>{t("settings.account.importPreviewTitle")}</DialogTitle>
          <DialogDescription>{t("settings.account.importPreviewDescription")}</DialogDescription>
        </DialogHeader>

        <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4 sm:px-6">
          {preview && (
            <div className="space-y-5">
              {preview.currencies.length > 0 && (
                <div>
                  <h4 className="mb-2 text-sm font-medium">{t("settings.account.importPreviewCurrencies")}</h4>
                  <div className="space-y-1.5">
                    {preview.currencies.map((currency) => (
                      <div key={currency.code} className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
                        <span>{currency.code}{currency.symbol ? ` (${currency.symbol})` : ""}</span>
                        <Badge variant={currency.is_new ? "default" : "secondary"} className="text-xs">
                          {currency.is_new ? t("settings.account.importPreviewNew") : t("settings.account.importPreviewExists")}
                        </Badge>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {preview.payment_methods.length > 0 && (
                <div>
                  <h4 className="mb-2 text-sm font-medium">{t("settings.account.importPreviewPaymentMethods")}</h4>
                  <div className="space-y-1.5">
                    {preview.payment_methods.map((paymentMethod) => (
                      <div key={paymentMethod.name} className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
                        <div>
                          <span>{paymentMethod.name}</span>
                          {paymentMethod.matched && (
                            <span className="ml-2 text-xs text-muted-foreground">
                              {t("settings.account.importPreviewMatchedAs", { name: paymentMethod.matched })}
                            </span>
                          )}
                        </div>
                        <Badge variant={paymentMethod.is_new ? "default" : "secondary"} className="text-xs">
                          {paymentMethod.is_new ? t("settings.account.importPreviewNew") : t("settings.account.importPreviewExists")}
                        </Badge>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {preview.categories.length > 0 && (
                <div>
                  <h4 className="mb-2 text-sm font-medium">{t("settings.account.importPreviewCategories")}</h4>
                  <div className="space-y-1.5">
                    {preview.categories.map((category) => (
                      <div key={category.name} className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
                        <span>{category.name}</span>
                        <Badge variant={category.is_new ? "default" : "secondary"} className="text-xs">
                          {category.is_new ? t("settings.account.importPreviewNew") : t("settings.account.importPreviewExists")}
                        </Badge>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {preview.subscriptions.length > 0 && (
                <SubscriptionPreviewList subscriptions={preview.subscriptions} showCategory />
              )}

              {preview.currencies.length === 0 &&
                preview.payment_methods.length === 0 &&
                preview.categories.length === 0 &&
                preview.subscriptions.length === 0 && (
                  <p className="py-8 text-center text-sm text-muted-foreground">
                    {t("settings.account.importPreviewEmpty")}
                  </p>
                )}
            </div>
          )}
        </div>

        <PreviewDialogFooter
          confirmDisabled={preview?.subscriptions.every((subscription) => subscription.skipped) ?? false}
          loading={loading}
          loadingLabel={t("settings.account.importing")}
          onConfirm={onConfirm}
          onReset={onReset}
          confirmLabel={t("settings.account.importPreviewConfirm")}
        />
      </DialogContent>
    </Dialog>
  )
}

function PreviewCount({ label, value }: { label: string, value: number }) {
  return (
    <div className="rounded-md border px-3 py-2">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="font-medium">{value}</div>
    </div>
  )
}

function PreviewUpdateRow({ changed, label }: { changed: boolean, label: string }) {
  const { t } = useTranslation()

  return (
    <div className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
      <span>{label}</span>
      <Badge variant={changed ? "default" : "secondary"} className="text-xs">
        {changed
          ? t("settings.account.subduxImportWillUpdate")
          : t("settings.account.subduxImportUnchanged")}
      </Badge>
    </div>
  )
}

function SubscriptionPreviewList({
  showCategory = false,
  subscriptions,
}: {
  showCategory?: boolean
  subscriptions: ImportPreview["subscriptions"]
}) {
  const { t } = useTranslation()

  return (
    <div>
      <h4 className="mb-2 text-sm font-medium">{t("settings.account.importPreviewSubscriptions")}</h4>
      <div className="space-y-1.5">
        {subscriptions.map((subscription, index) => (
          <div key={index} className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <span className="truncate font-medium">{subscription.name}</span>
                <span className="shrink-0 text-xs text-muted-foreground">
                  {subscription.amount} {subscription.currency}
                </span>
              </div>
              {showCategory && subscription.category && (
                <span className="text-xs text-muted-foreground">{subscription.category}</span>
              )}
            </div>
            <Badge
              variant={subscription.skipped ? "outline" : "default"}
              className="ml-2 shrink-0 text-xs"
            >
              {subscription.skipped ? t("settings.account.importPreviewSkipped") : t("settings.account.importPreviewNew")}
            </Badge>
          </div>
        ))}
      </div>
    </div>
  )
}

function PreviewDialogFooter({
  confirmDisabled = false,
  confirmLabel,
  loading,
  loadingLabel,
  onConfirm,
  onReset,
}: {
  confirmDisabled?: boolean
  confirmLabel: string
  loading: boolean
  loadingLabel: string
  onConfirm: () => void | Promise<void>
  onReset: () => void
}) {
  const { t } = useTranslation()

  return (
    <div className="sticky bottom-0 z-10 flex justify-end gap-2 border-t bg-background/95 px-5 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6">
      <Button
        variant="outline"
        size="sm"
        disabled={loading}
        onClick={onReset}
      >
        {t("settings.account.importPreviewCancel")}
      </Button>
      <Button
        size="sm"
        disabled={loading || confirmDisabled}
        onClick={() => void onConfirm()}
      >
        {loading ? loadingLabel : confirmLabel}
      </Button>
    </div>
  )
}
