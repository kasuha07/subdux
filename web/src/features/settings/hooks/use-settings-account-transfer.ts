import { useRef, useState, type ChangeEvent, type RefObject } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "@/lib/toast"

import { api, getAPIErrorMessage } from "@/lib/api"
import type {
  ImportPreview,
  SubduxImportPreview,
} from "@/features/settings/settings-account-import-types"

interface UseSettingsAccountTransferResult {
  exportLoading: boolean
  exportSecretsConfirmOpen: boolean
  handleConfirmImport: (reauthTicket: string) => Promise<void>
  handleConfirmSubduxImport: (reauthTicket: string) => Promise<void>
  handleExport: (includeSecrets: boolean, reauthTicket: string) => Promise<void>
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

  async function downloadExport(path: string, reauthTicket: string) {
    try {
      const res = await api.response(path, {
        headers: { "X-Reauth-Ticket": reauthTicket },
        errorFallbackKey: "settings.account.exportFailed",
      })
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
    } catch (error) {
      const errorMsg = getAPIErrorMessage(error, "settings.account.exportFailed")
      toast.error(errorMsg)
      throw error
    }
  }

  async function handleExport(includeSecrets: boolean, reauthTicket: string) {
    setExportLoading(true)
    try {
      const path = includeSecrets ? "/export?include_secrets=1&confirm=include_secrets" : "/export"
      await downloadExport(path, reauthTicket)
      if (includeSecrets) {
        setExportSecretsConfirmOpen(false)
      }
    } catch {
      // downloadExport surfaces a toast for API and transport failures.
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

      const response = await api.post<{ preview: ImportPreview }>(
        "/import/wallos",
        { data, confirm: false },
        { errorFallbackKey: "settings.account.importFailed" }
      )

      setImportPreview(response.preview)
      setImportRawData(data)
      setImportPreviewOpen(true)
    } catch (error) {
      toast.error(getAPIErrorMessage(error, "settings.account.importFailed"))
    } finally {
      setImportLoading(false)
      if (importFileRef.current) {
        importFileRef.current.value = ""
      }
    }
  }

  async function handleConfirmImport(reauthTicket: string) {
    if (!importRawData) return

    setImportLoading(true)
    try {
      const response = await api.post<{ result: { imported: number; skipped: number } }>(
        "/import/wallos",
        { data: importRawData, confirm: true },
        {
          headers: { "X-Reauth-Ticket": reauthTicket },
          errorFallbackKey: "settings.account.importFailed",
        }
      )

      toast.success(
        t("settings.account.importSuccess", {
          imported: response.result.imported,
          skipped: response.result.skipped,
        })
      )
      resetImportPreview()
    } catch (error) {
      toast.error(getAPIErrorMessage(error, "settings.account.importFailed"))
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

      const response = await api.post<{ preview: SubduxImportPreview }>(
        "/import/subdux",
        { data, confirm: false },
        { errorFallbackKey: "settings.account.subduxImportFailed" }
      )

      setSubduxImportPreview(response.preview)
      setSubduxImportRawData(data as Record<string, unknown>)
      setSubduxImportPreviewOpen(true)
    } catch (error) {
      toast.error(getAPIErrorMessage(error, "settings.account.subduxImportFailed"))
    } finally {
      setSubduxImportLoading(false)
      if (subduxImportFileRef.current) {
        subduxImportFileRef.current.value = ""
      }
    }
  }

  async function handleConfirmSubduxImport(reauthTicket: string) {
    if (!subduxImportRawData) return

    setSubduxImportLoading(true)
    try {
      const response = await api.post<{ result: { imported: number; skipped: number } }>(
        "/import/subdux",
        { data: subduxImportRawData, confirm: true },
        {
          headers: { "X-Reauth-Ticket": reauthTicket },
          errorFallbackKey: "settings.account.subduxImportFailed",
        }
      )

      toast.success(
        t("settings.account.subduxImportSuccess", {
          imported: response.result.imported,
          skipped: response.result.skipped,
        })
      )
      resetSubduxImportPreview()
    } catch (error) {
      toast.error(getAPIErrorMessage(error, "settings.account.subduxImportFailed"))
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
