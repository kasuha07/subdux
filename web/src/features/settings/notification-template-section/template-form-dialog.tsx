import type { FormEvent } from "react"
import type { TFunction } from "i18next"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import type { NotificationTemplate } from "@/types"

import { CHANNEL_TYPES, TEMPLATE_FORMATS, TEMPLATE_VARIABLES } from "./constants"

export type TemplateFormat = (typeof TEMPLATE_FORMATS)[number]

interface Props {
  channelType: string
  editingTemplate: NotificationTemplate | null
  formOpen: boolean
  format: TemplateFormat
  previewResult: string | null
  previewing: boolean
  saving: boolean
  t: TFunction
  templateContent: string
  onChannelTypeChange: (value: string) => void
  onClose: () => void
  onFormatChange: (value: TemplateFormat) => void
  onPreview: () => void
  onSubmit: (e: FormEvent) => void
  onTemplateContentChange: (value: string) => void
}

export function NotificationTemplateFormDialog({
  channelType,
  editingTemplate,
  formOpen,
  format,
  onChannelTypeChange,
  onClose,
  onFormatChange,
  onPreview,
  onSubmit,
  onTemplateContentChange,
  previewResult,
  previewing,
  saving,
  t,
  templateContent,
}: Props) {
  return (
    <Dialog open={formOpen} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent className="flex max-h-[calc(100vh-1.5rem)] max-w-2xl flex-col gap-0 overflow-hidden p-0 sm:max-h-[85vh]">
        <DialogHeader className="border-b px-5 pt-5 pb-4 sm:px-6">
          <DialogTitle>
            {editingTemplate
              ? t("settings.notifications.templates.editButton")
              : t("settings.notifications.templates.addButton")}
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="flex min-h-0 flex-1 flex-col">
          <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4 sm:px-6">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>{t("settings.notifications.templates.channelType")}</Label>
                <Select value={channelType} onValueChange={onChannelTypeChange} disabled={!!editingTemplate}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="default">{t("settings.notifications.templates.defaultTemplate")}</SelectItem>
                    {CHANNEL_TYPES.map((type) => (
                      <SelectItem key={type} value={type}>
                        {t(`settings.notifications.channels.type.${type}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>{t("settings.notifications.templates.format")}</Label>
                <Select value={format} onValueChange={(value) => onFormatChange(value as TemplateFormat)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="plaintext">Plaintext</SelectItem>
                    <SelectItem value="markdown">Markdown</SelectItem>
                    <SelectItem value="html">HTML</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>{t("settings.notifications.templates.template")}</Label>
                <span className="text-xs text-muted-foreground">
                  {templateContent.length} / 2000
                </span>
              </div>
              <Textarea
                className="min-h-[200px] font-mono text-sm"
                value={templateContent}
                onChange={(e) => onTemplateContentChange(e.target.value)}
                maxLength={2000}
                required
              />
              <div className="space-y-2 rounded-md border bg-muted/50 p-3 text-xs">
                <p className="font-medium">{t("settings.notifications.templates.availableVariables")}:</p>
                <div className="grid grid-cols-1 gap-1.5 font-mono">
                  {TEMPLATE_VARIABLES.map((item) => (
                    <div key={item.name}>
                      <span className="text-primary">{item.name}</span>
                      {" - "}
                      {t(`settings.notifications.templates.${item.key}`)}
                    </div>
                  ))}
                </div>
                <div className="border-t pt-2">
                  <p className="mb-1 font-medium">{t("settings.notifications.templates.exampleTitle")}:</p>
                  <p className="font-mono text-muted-foreground">
                    {t("settings.notifications.templates.exampleTemplate")}
                  </p>
                </div>
              </div>
            </div>

            {previewResult && (
              <div className="space-y-2 rounded-md border bg-muted/50 p-3">
                <Label>{t("settings.notifications.templates.previewResult")}</Label>
                <div className="whitespace-pre-wrap font-mono text-sm">
                  {previewResult}
                </div>
              </div>
            )}
          </div>

          <div className="sticky bottom-0 z-10 border-t bg-background/95 px-5 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6">
            <div className="flex gap-2">
              <Button type="button" variant="outline" onClick={onPreview} disabled={previewing || !templateContent.trim()}>
                {t("settings.notifications.templates.previewButton")}
              </Button>
              <div className="flex-1" />
              <Button type="button" variant="outline" onClick={onClose}>
                {t("settings.notifications.channels.cancel")}
              </Button>
              <Button type="submit" disabled={saving || !templateContent.trim()}>
                {saving ? t("settings.notifications.channels.adding") : t("settings.notifications.channels.save")}
              </Button>
            </div>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
