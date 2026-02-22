import { useState, type FormEvent } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { api } from "@/lib/api"
import type {
  CreateTemplateInput,
  NotificationTemplate,
  PreviewTemplateInput,
  UpdateTemplateInput,
} from "@/types"

import { NotificationTemplateFormDialog, type TemplateFormat } from "./notification-template-section/template-form-dialog"

interface Props {
  templates: NotificationTemplate[]
  onTemplatesChange: (templates: NotificationTemplate[]) => void
}

export function NotificationTemplateSection({ templates, onTemplatesChange }: Props) {
  const { t } = useTranslation()

  const [formOpen, setFormOpen] = useState(false)
  const [editingTemplate, setEditingTemplate] = useState<NotificationTemplate | null>(null)
  const [saving, setSaving] = useState(false)

  const [channelType, setChannelType] = useState<string>("default")
  const [format, setFormat] = useState<TemplateFormat>("plaintext")
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
    setFormat(template.format as TemplateFormat)
    setTemplateContent(template.template)
    setPreviewResult(null)
    setFormOpen(true)
  }

  async function handleDelete(template: NotificationTemplate) {
    if (!window.confirm(t("settings.notifications.templates.deleteConfirm"))) return
    try {
      await api.delete(`/notifications/templates/${template.id}`)
      onTemplatesChange(templates.filter((item) => item.id !== template.id))
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
        onTemplatesChange(templates.map((item) => (item.id === updated.id ? updated : item)))
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
    <div className="space-y-3">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold tracking-tight">{t("settings.notifications.templates.title")}</h2>
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
        <NotificationTemplateFormDialog
          channelType={channelType}
          editingTemplate={editingTemplate}
          formOpen={formOpen}
          format={format}
          onChannelTypeChange={setChannelType}
          onClose={() => setFormOpen(false)}
          onFormatChange={setFormat}
          onPreview={() => void handlePreview()}
          onSubmit={(e) => void handleSubmit(e)}
          onTemplateContentChange={setTemplateContent}
          previewResult={previewResult}
          previewing={previewing}
          saving={saving}
          t={t}
          templateContent={templateContent}
        />
      )}
    </div>
  )
}
