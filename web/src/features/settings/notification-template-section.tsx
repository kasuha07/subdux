import { useState, type FormEvent } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
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
import { api } from "@/lib/api"
import type { NotificationTemplate, CreateTemplateInput, UpdateTemplateInput, PreviewTemplateInput } from "@/types"

interface Props {
  templates: NotificationTemplate[]
  onTemplatesChange: (templates: NotificationTemplate[]) => void
}

const CHANNEL_TYPES = [
  "smtp", "resend", "telegram", "webhook", "gotify", "ntfy", "bark",
  "serverchan", "feishu", "wecom", "dingtalk", "pushdeer", "pushplus",
  "pushover", "napcat"
]

export function NotificationTemplateSection({ templates, onTemplatesChange }: Props) {
  const { t } = useTranslation()

  const [formOpen, setFormOpen] = useState(false)
  const [editingTemplate, setEditingTemplate] = useState<NotificationTemplate | null>(null)
  const [saving, setSaving] = useState(false)

  const [channelType, setChannelType] = useState<string>("default")
  const [format, setFormat] = useState<"plaintext" | "markdown" | "html">("plaintext")
  const [templateContent, setTemplateContent] = useState("")

  const [previewing, setPreviewing] = useState(false)
  const [previewResult, setPreviewResult] = useState<string | null>(null)

  function handleAdd() {
    setEditingTemplate(null)
    setChannelType("default")
    setFormat("plaintext")
    setTemplateContent("")
    setPreviewResult(null)
    setFormOpen(true)
  }

  function handleEdit(template: NotificationTemplate) {
    setEditingTemplate(template)
    setChannelType(template.channel_type ?? "default")
    setFormat(template.format as "plaintext" | "markdown" | "html")
    setTemplateContent(template.template)
    setPreviewResult(null)
    setFormOpen(true)
  }

  async function handleDelete(template: NotificationTemplate) {
    if (!window.confirm(t("settings.notifications.templates.deleteConfirm"))) return
    try {
      await api.delete(`/notifications/templates/${template.id}`)
      onTemplatesChange(templates.filter((t) => t.id !== template.id))
      toast.success(t("settings.notifications.templates.deleteSuccess"))
    } catch {
      void 0
    }
  }

  async function handlePreview() {
    if (!templateContent.trim()) return
    setPreviewing(true)
    try {
      const input: PreviewTemplateInput = {
        format,
        template: templateContent,
      }
      const result = await api.post<{ preview: string }>("/notifications/templates/preview", input)
      setPreviewResult(result.preview)
    } catch {
      void 0
    } finally {
      setPreviewing(false)
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!templateContent.trim()) return

    setSaving(true)
    try {
      if (editingTemplate) {
        const input: UpdateTemplateInput = {
          format,
          template: templateContent,
        }
        const updated = await api.put<NotificationTemplate>(`/notifications/templates/${editingTemplate.id}`, input)
        onTemplatesChange(templates.map((t) => (t.id === updated.id ? updated : t)))
        toast.success(t("settings.notifications.templates.updateSuccess"))
      } else {
        const input: CreateTemplateInput = {
          channel_type: channelType === "default" ? null : channelType,
          format,
          template: templateContent,
        }
        const created = await api.post<NotificationTemplate>("/notifications/templates", input)
        onTemplatesChange([...templates, created])
        toast.success(t("settings.notifications.templates.addSuccess"))
      }
      setFormOpen(false)
    } catch {
      void 0
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card>
      <CardContent className="space-y-3 p-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 className="text-sm font-medium">{t("settings.notifications.templates.title")}</h2>
            <p className="mt-0.5 text-sm text-muted-foreground">
              {t("settings.notifications.templates.description")}
            </p>
          </div>
          <Button size="sm" onClick={handleAdd}>
            {t("settings.notifications.templates.addButton")}
          </Button>
        </div>

        <div className="space-y-2">
          {templates.map((template) => (
            <div
              key={template.id}
              className="flex flex-col gap-3 rounded-md border bg-card/70 p-3 sm:flex-row sm:items-center sm:justify-between"
            >
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium">
                    {template.channel_type
                      ? t(`settings.notifications.channels.type.${template.channel_type}`)
                      : t("settings.notifications.templates.defaultTemplate")}
                  </p>
                  <Badge variant="secondary">{template.format}</Badge>
                </div>
                <p className="text-xs text-muted-foreground line-clamp-1">
                  {template.template}
                </p>
              </div>

              <div className="flex flex-wrap items-center gap-2">
                <Button size="sm" variant="outline" onClick={() => handleEdit(template)}>
                  {t("settings.notifications.templates.editButton")}
                </Button>
                <Button size="sm" variant="outline" onClick={() => void handleDelete(template)}>
                  {t("settings.notifications.templates.deleteButton")}
                </Button>
              </div>
            </div>
          ))}
        </div>

        {formOpen && (
          <Dialog open={formOpen} onOpenChange={(v) => { if (!v) setFormOpen(false) }}>
            <DialogContent className="max-w-2xl">
              <DialogHeader>
                <DialogTitle>
                  {editingTemplate
                    ? t("settings.notifications.templates.editButton")
                    : t("settings.notifications.templates.addButton")}
                </DialogTitle>
              </DialogHeader>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>{t("settings.notifications.templates.channelType")}</Label>
                    <Select value={channelType} onValueChange={setChannelType} disabled={!!editingTemplate}>
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
                    <Select value={format} onValueChange={(v) => setFormat(v as any)}>
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
                    onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setTemplateContent(e.target.value)}
                    maxLength={2000}
                    required
                  />
                  <div className="space-y-2 rounded-md border p-3 bg-muted/50 text-xs">
                    <p className="font-medium">{t("settings.notifications.templates.availableVariables")}:</p>
                    <div className="grid grid-cols-1 gap-1.5 font-mono">
                      <div><span className="text-primary">{"{{.SubscriptionName}}"}</span> - {t("settings.notifications.templates.varSubscriptionName")}</div>
                      <div><span className="text-primary">{"{{.BillingDate}}"}</span> - {t("settings.notifications.templates.varBillingDate")}</div>
                      <div><span className="text-primary">{"{{.Amount}}"}</span> - {t("settings.notifications.templates.varAmount")}</div>
                      <div><span className="text-primary">{"{{.Currency}}"}</span> - {t("settings.notifications.templates.varCurrency")}</div>
                      <div><span className="text-primary">{"{{.DaysUntil}}"}</span> - {t("settings.notifications.templates.varDaysUntil")}</div>
                      <div><span className="text-primary">{"{{.Category}}"}</span> - {t("settings.notifications.templates.varCategory")}</div>
                      <div><span className="text-primary">{"{{.UserEmail}}"}</span> - {t("settings.notifications.templates.varUserEmail")}</div>
                    </div>
                    <div className="pt-2 border-t">
                      <p className="font-medium mb-1">{t("settings.notifications.templates.exampleTitle")}:</p>
                      <p className="font-mono text-muted-foreground">
                        {t("settings.notifications.templates.exampleTemplate")}
                      </p>
                    </div>
                </div>
                </div>

                {previewResult && (
                  <div className="space-y-2 rounded-md border p-3 bg-muted/50">
                    <Label>{t("settings.notifications.templates.previewResult")}</Label>
                    <div className="text-sm whitespace-pre-wrap font-mono">
                      {previewResult}
                    </div>
                  </div>
                )}

                <div className="flex gap-2 pt-2">
                  <Button type="button" variant="outline" onClick={handlePreview} disabled={previewing || !templateContent.trim()}>
                    {t("settings.notifications.templates.previewButton")}
                  </Button>
                  <div className="flex-1" />
                  <Button type="button" variant="outline" onClick={() => setFormOpen(false)}>
                    {t("settings.notifications.channels.cancel")}
                  </Button>
                  <Button type="submit" disabled={saving || !templateContent.trim()}>
                    {saving ? t("settings.notifications.channels.adding") : t("settings.notifications.channels.save")}
                  </Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>
        )}
      </CardContent>
    </Card>
  )
}
