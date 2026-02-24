import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { Copy, KeyRound, Plus, Trash2 } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { TabsContent } from "@/components/ui/tabs"
import { api } from "@/lib/api"
import type { APIKey, CreateAPIKeyResponse } from "@/types"

interface SettingsAPIKeyTabProps {
  active: boolean
}

export default function SettingsAPIKeyTab({ active }: SettingsAPIKeyTabProps) {
  const { t } = useTranslation()
  const [keys, setKeys] = useState<APIKey[]>([])
  const [loading, setLoading] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [name, setName] = useState("")
  const [creating, setCreating] = useState(false)
  const [newKey, setNewKey] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<number | null>(null)

  const loadKeys = useCallback(() => {
    setLoading(true)
    api
      .get<APIKey[]>("/api-keys")
      .then(setKeys)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (active) {
      loadKeys()
    }
  }, [active, loadKeys])

  async function handleCreate() {
    if (!name.trim()) return
    setCreating(true)
    try {
      const resp = await api.post<CreateAPIKeyResponse>("/api-keys", {
        name: name.trim(),
      })
      setNewKey(resp.key)
      setKeys((prev) => [resp.api_key, ...prev])
      setName("")
    } catch {
      // error toast handled by api helper
    } finally {
      setCreating(false)
    }
  }

  async function handleDelete(id: number) {
    if (!confirm(t("settings.apiKeys.deleteConfirm"))) return
    setDeletingId(id)
    try {
      await api.delete(`/api-keys/${id}`)
      setKeys((prev) => prev.filter((k) => k.id !== id))
    } catch {
      // error toast handled by api helper
    } finally {
      setDeletingId(null)
    }
  }

  function handleCopy(text: string) {
    navigator.clipboard.writeText(text).then(() => {
      toast.success(t("settings.apiKeys.copied"))
    })
  }

  function handleDialogClose(open: boolean) {
    setCreateOpen(open)
    if (!open) {
      setNewKey(null)
      setName("")
    }
  }

  function formatDate(dateStr: string | null) {
    if (!dateStr) return t("settings.apiKeys.never")
    return new Date(dateStr).toLocaleDateString()
  }

  return (
    <TabsContent value="apikey">
      <div className="space-y-4">
        <div className="flex items-start justify-between">
          <div>
            <h2 className="text-base font-semibold tracking-tight">
              {t("settings.apiKeys.title")}
            </h2>
            <p className="mt-0.5 text-sm text-muted-foreground">
              {t("settings.apiKeys.description")}
            </p>
          </div>
          <Dialog open={createOpen} onOpenChange={handleDialogClose}>
            <DialogTrigger asChild>
              <Button size="sm" className="gap-1.5">
                <Plus className="size-4" />
                {t("settings.apiKeys.create")}
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-md">
              <DialogHeader>
                <DialogTitle>{t("settings.apiKeys.create")}</DialogTitle>
              </DialogHeader>
              {newKey ? (
                <div className="space-y-3">
                  <div className="rounded-md bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-400">
                    {t("settings.apiKeys.copyWarning")}
                  </div>
                  <div className="space-y-2">
                    <Label>{t("settings.apiKeys.key")}</Label>
                    <div className="flex gap-2">
                      <code className="flex-1 rounded-md border bg-muted px-3 py-2 text-xs break-all">
                        {newKey}
                      </code>
                      <Button
                        size="icon-sm"
                        variant="outline"
                        onClick={() => handleCopy(newKey)}
                      >
                        <Copy className="size-4" />
                      </Button>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label>{t("settings.apiKeys.usage")}</Label>
                    <p className="text-sm text-muted-foreground">
                      {t("settings.apiKeys.usageDescription")}
                    </p>
                    <code className="block rounded-md border bg-muted px-3 py-2 text-xs">
                      X-API-Key: {newKey}
                    </code>
                  </div>
                </div>
              ) : (
                <form
                  onSubmit={(e) => {
                    e.preventDefault()
                    void handleCreate()
                  }}
                  className="space-y-4"
                >
                  <div className="space-y-2">
                    <Label htmlFor="api-key-name">{t("settings.apiKeys.name")}</Label>
                    <Input
                      id="api-key-name"
                      placeholder={t("settings.apiKeys.namePlaceholder")}
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      maxLength={100}
                      required
                    />
                  </div>
                  <Button size="sm" type="submit" disabled={creating || !name.trim()}>
                    {creating ? t("settings.apiKeys.creating") : t("settings.apiKeys.create")}
                  </Button>
                </form>
              )}
            </DialogContent>
          </Dialog>
        </div>

        <Separator />

        {!loading && keys.length === 0 && (
          <div className="py-8 text-center">
            <KeyRound className="mx-auto size-8 text-muted-foreground/50" />
            <p className="mt-2 text-sm font-medium text-muted-foreground">
              {t("settings.apiKeys.empty")}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("settings.apiKeys.emptyDescription")}
            </p>
          </div>
        )}

        {keys.length > 0 && (
          <div className="space-y-2">
            {keys.map((key) => (
              <div
                key={key.id}
                className="flex items-center justify-between rounded-md border px-3 py-2.5"
              >
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium">{key.name}</p>
                  <div className="mt-0.5 flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
                    <span>
                      {t("settings.apiKeys.prefix")}: <code>{key.prefix}...</code>
                    </span>
                    <span>
                      {t("settings.apiKeys.createdAt")}: {formatDate(key.created_at)}
                    </span>
                    <span>
                      {t("settings.apiKeys.lastUsed")}: {formatDate(key.last_used_at)}
                    </span>
                    {key.expires_at && (
                      <span>
                        {t("settings.apiKeys.expiresAt")}: {formatDate(key.expires_at)}
                      </span>
                    )}
                  </div>
                </div>
                <Button
                  size="icon-sm"
                  variant="ghost"
                  className="text-destructive hover:text-destructive"
                  disabled={deletingId === key.id}
                  onClick={() => void handleDelete(key.id)}
                >
                  <Trash2 className="size-4" />
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>
    </TabsContent>
  )
}
