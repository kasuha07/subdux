import { useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Separator } from "@/components/ui/separator"
import { TabsContent } from "@/components/ui/tabs"
import { api } from "@/lib/api"
import type {
  NotificationChannel,
  NotificationLog,
  NotificationPolicy,
  UpdateNotificationPolicyInput,
  NotificationTemplate,
} from "@/types"

import { NotificationChannelForm } from "./notification-channel-form"
import { NotificationChannelList } from "./notification-channel-list"
import { NotificationLogList } from "./notification-log-list"
import { NotificationPolicySection } from "./notification-policy-section"
import { NotificationTemplateSection } from "./notification-template-section"

export default function SettingsNotificationTab() {
  const { t } = useTranslation()

  const [channels, setChannels] = useState<NotificationChannel[]>([])
  const [policy, setPolicy] = useState<NotificationPolicy>({ days_before: 3, notify_on_due_day: true })
  const [logs, setLogs] = useState<NotificationLog[]>([])
  const [templates, setTemplates] = useState<NotificationTemplate[]>([])

  const [formOpen, setFormOpen] = useState(false)
  const [editingChannel, setEditingChannel] = useState<NotificationChannel | null>(null)
  const [formSaving, setFormSaving] = useState(false)
  const [policySaving, setPolicySaving] = useState(false)
  const [testingChannelId, setTestingChannelId] = useState<number | null>(null)
  const [updatingChannelId, setUpdatingChannelId] = useState<number | null>(null)

  const loaded = useRef(false)

  useEffect(() => {
    if (loaded.current) return
    loaded.current = true

    api.get<NotificationChannel[]>("/notifications/channels")
      .then((data) => setChannels(data ?? []))
      .catch(() => void 0)

    api.get<NotificationPolicy>("/notifications/policy")
      .then((data) => { if (data) setPolicy(data) })
      .catch(() => void 0)

    api.get<NotificationLog[]>("/notifications/logs")
      .then((data) => setLogs(data ?? []))
      .catch(() => void 0)

    api.get<NotificationTemplate[]>("/notifications/templates")
      .then((data) => setTemplates(data ?? []))
      .catch(() => void 0)
  }, [])

  function handleAddChannel() {
    setEditingChannel(null)
    setFormOpen(true)
  }

  function handleEditChannel(channel: NotificationChannel) {
    setEditingChannel(channel)
    setFormOpen(true)
  }

  async function handleSaveChannel(type: string, config: string) {
    setFormSaving(true)
    try {
      if (editingChannel) {
        const updated = await api.put<NotificationChannel>(
          `/notifications/channels/${editingChannel.id}`,
          { config }
        )
        setChannels((prev) => prev.map((ch) => (ch.id === updated.id ? updated : ch)))
        toast.success(t("settings.notifications.channels.updateSuccess"))
      } else {
        const created = await api.post<NotificationChannel>("/notifications/channels", {
          type,
          enabled: true,
          config,
        })
        setChannels((prev) => [...prev, created])
        toast.success(t("settings.notifications.channels.addSuccess"))
      }
      setFormOpen(false)
    } catch {
      void 0
    } finally {
      setFormSaving(false)
    }
  }

  async function handleToggleChannel(channel: NotificationChannel, enabled: boolean) {
    setUpdatingChannelId(channel.id)
    try {
      const updated = await api.put<NotificationChannel>(
        `/notifications/channels/${channel.id}`,
        { enabled }
      )
      setChannels((prev) => prev.map((ch) => (ch.id === updated.id ? updated : ch)))
    } catch {
      void 0
    } finally {
      setUpdatingChannelId(null)
    }
  }

  async function handleDeleteChannel(channel: NotificationChannel) {
    if (!window.confirm(t("settings.notifications.channels.deleteConfirm"))) return
    try {
      await api.delete(`/notifications/channels/${channel.id}`)
      setChannels((prev) => prev.filter((ch) => ch.id !== channel.id))
      toast.success(t("settings.notifications.channels.deleteSuccess"))
    } catch {
      void 0
    }
  }

  async function handleTestChannel(channel: NotificationChannel) {
    setTestingChannelId(channel.id)
    try {
      await api.post(`/notifications/channels/${channel.id}/test`, {})
      toast.success(t("settings.notifications.channels.testSuccess"))
    } catch {
      void 0
    } finally {
      setTestingChannelId(null)
    }
  }

  async function handleSavePolicy(input: UpdateNotificationPolicyInput) {
    setPolicySaving(true)
    try {
      const updated = await api.put<NotificationPolicy>("/notifications/policy", input)
      if (updated) setPolicy(updated)
      toast.success(t("settings.notifications.policy.saveSuccess"))
    } catch {
      void 0
    } finally {
      setPolicySaving(false)
    }
  }

  return (
    <TabsContent value="notification" className="space-y-6">
      <NotificationPolicySection
        policy={policy}
        onSave={handleSavePolicy}
        saving={policySaving}
      />

      <Separator />

      <NotificationTemplateSection
        templates={templates}
        onTemplatesChange={setTemplates}
      />

      <Separator />

      <NotificationChannelList
        channels={channels}
        onAddChannel={handleAddChannel}
        onEditChannel={handleEditChannel}
        onToggleChannel={handleToggleChannel}
        onDeleteChannel={handleDeleteChannel}
        onTestChannel={handleTestChannel}
        testingChannelId={testingChannelId}
        updatingChannelId={updatingChannelId}
      />

      {formOpen && (
        <NotificationChannelForm
          open={formOpen}
          channel={editingChannel}
          onClose={() => setFormOpen(false)}
          onSave={handleSaveChannel}
          saving={formSaving}
        />
      )}

      <Separator />

      <NotificationLogList logs={logs} />
    </TabsContent>
  )
}
