import { useRef, useState, type ChangeEvent, type RefObject } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { api } from "@/lib/api"
import type {
  ImportPreview,
  SubduxImportPreview,
} from "@/features/settings/settings-account-import-types"

interface UseSettingsAccountTransferResult {
  exportLoading: boolean
  exportSecretsConfirmOpen: boolean
  handleConfirmImport: () => Promise<void>
  handleConfirmSubduxImport: () => Promise<void>
  handleExport: (includeSecrets?: boolean) => Promise<void>
  handleImportSubdux: (event: ChangeEvent<HTMLInputElement>) => Promise<void>
  handleImportWallos: (event: ChangeEvent<HTMLInputElement>) => Promise<void>
  importFileRef: RefObject<HTMLInputElement | null>
  importLoading: boolean
  importPreview: ImportPreview | null
  importPreviewOpen: boolean
  resetImportPreview: () => void
  resetSubduxImportPreview: () => void
  setExportSecretsConfirmOpen: (open: boolean) => void
  setImportPreviewOpen: (open: boolean) => void
  setSubduxImportPreviewOpen: (open: boolean) => void
  subduxImportFileRef: RefObject<HTMLInputElement | null>
  subduxImportLoading: boolean
  subduxImportPreview: SubduxImportPreview | null
  subduxImportPreviewOpen: boolean
}

export function useSettingsAccountTransfer(): UseSettingsAccountTransferResult {
  const { t } = useTranslation()
  const [exportLoading, setExportLoading] = useState(false)
  const [importLoading, setImportLoading] = useState(false)
  const importFileRef = useRef<HTMLInputElement>(null)
  const [importPreviewOpen, setImportPreviewOpen] = useState(false)
  const [importPreview, setImportPreview] = useState<ImportPreview | null>(null)
  const [importRawData, setImportRawData] = useState<unknown[] | null>(null)
  const [subduxImportLoading, setSubduxImportLoading] = useState(false)
  const subduxImportFileRef = useRef<HTMLInputElement>(null)
  const [subduxImportPreviewOpen, setSubduxImportPreviewOpen] = useState(false)
  const [subduxImportPreview, setSubduxImportPreview] = useState<SubduxImportPreview | null>(null)
  const [subduxImportRawData, setSubduxImportRawData] = useState<Record<string, unknown> | null>(null)
  const [exportSecretsConfirmOpen, setExportSecretsConfirmOpen] = useState(false)

  function resetImportPreview() {
    setImportPreviewOpen(false)
    setImportPreview(null)
    setImportRawData(null)
  }

  function resetSubduxImportPreview() {
    setSubduxImportPreviewOpen(false)
    setSubduxImportPreview(null)
    setSubduxImportRawData(null)
  }

  async function downloadExport(path: string) {
    const res = await api.fetch(path)
    if (!res.ok) throw new Error("Export failed")
    const blob = await res.blob()
    const disposition = res.headers.get("Content-Disposition")
    let filename = "subdux-export.json"
    if (disposition) {
      const match = disposition.match(/filename="?([^"]+)"?/)
      if (match) filename = match[1]
    }
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    URL.revokeObjectURL(url)
    a.remove()
  }

  async function handleExport(includeSecrets = false) {
    setExportLoading(true)
    try {
      const path = includeSecrets ? "/export?include_secrets=1&confirm=include_secrets" : "/export"
      await downloadExport(path)
      if (includeSecrets) {
        setExportSecretsConfirmOpen(false)
      }
    } catch {
      // error toast is handled by the fetch failure
    } finally {
      setExportLoading(false)
    }
  }

  async function handleImportWallos(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) return

    setImportLoading(true)
    try {
      const text = await file.text()
      const data = JSON.parse(text)

      if (!Array.isArray(data)) {
        toast.error(t("settings.account.importInvalidFormat"))
        return
      }

      const res = await api.fetch("/import/wallos", {
        method: "POST",
        body: JSON.stringify({ data, confirm: false }),
      })

      if (!res.ok) {
        const err = await res.json()
        toast.error(err.error || t("settings.account.importFailed"))
        return
      }

      const preview: ImportPreview = (await res.json()).preview
      setImportPreview(preview)
      setImportRawData(data)
      setImportPreviewOpen(true)
    } catch {
      toast.error(t("settings.account.importFailed"))
    } finally {
      setImportLoading(false)
      if (importFileRef.current) {
        importFileRef.current.value = ""
      }
    }
  }

  async function handleConfirmImport() {
    if (!importRawData) return

    setImportLoading(true)
    try {
      const res = await api.fetch("/import/wallos", {
        method: "POST",
        body: JSON.stringify({ data: importRawData, confirm: true }),
      })

      if (!res.ok) {
        const err = await res.json()
        toast.error(err.error || t("settings.account.importFailed"))
        return
      }

      const { result } = await res.json()
      toast.success(
        t("settings.account.importSuccess", {
          imported: result.imported,
          skipped: result.skipped,
        })
      )
      resetImportPreview()
    } catch {
      toast.error(t("settings.account.importFailed"))
    } finally {
      setImportLoading(false)
    }
  }

  async function handleImportSubdux(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) return

    setSubduxImportLoading(true)
    try {
      const text = await file.text()
      const data = JSON.parse(text)

      if (typeof data !== "object" || data === null || Array.isArray(data)) {
        toast.error(t("settings.account.subduxImportInvalidFormat"))
        return
      }

      const res = await api.fetch("/import/subdux", {
        method: "POST",
        body: JSON.stringify({ data, confirm: false }),
      })

      if (!res.ok) {
        const err = await res.json()
        toast.error(err.error || t("settings.account.subduxImportFailed"))
        return
      }

      const preview: SubduxImportPreview = (await res.json()).preview
      setSubduxImportPreview(preview)
      setSubduxImportRawData(data as Record<string, unknown>)
      setSubduxImportPreviewOpen(true)
    } catch {
      toast.error(t("settings.account.subduxImportFailed"))
    } finally {
      setSubduxImportLoading(false)
      if (subduxImportFileRef.current) {
        subduxImportFileRef.current.value = ""
      }
    }
  }

  async function handleConfirmSubduxImport() {
    if (!subduxImportRawData) return

    setSubduxImportLoading(true)
    try {
      const res = await api.fetch("/import/subdux", {
        method: "POST",
        body: JSON.stringify({ data: subduxImportRawData, confirm: true }),
      })

      if (!res.ok) {
        const err = await res.json()
        toast.error(err.error || t("settings.account.subduxImportFailed"))
        return
      }

      const { result } = await res.json()
      toast.success(
        t("settings.account.subduxImportSuccess", {
          imported: result.imported,
          skipped: result.skipped,
        })
      )
      resetSubduxImportPreview()
    } catch {
      toast.error(t("settings.account.subduxImportFailed"))
    } finally {
      setSubduxImportLoading(false)
    }
  }

  return {
    exportLoading,
    exportSecretsConfirmOpen,
    handleConfirmImport,
    handleConfirmSubduxImport,
    handleExport,
    handleImportSubdux,
    handleImportWallos,
    importFileRef,
    importLoading,
    importPreview,
    importPreviewOpen,
    resetImportPreview,
    resetSubduxImportPreview,
    setExportSecretsConfirmOpen,
    setImportPreviewOpen,
    setSubduxImportPreviewOpen,
    subduxImportFileRef,
    subduxImportLoading,
    subduxImportPreview,
    subduxImportPreviewOpen,
  }
}
