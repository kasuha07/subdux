import { useState, useEffect, useRef, type DragEvent, type FormEvent } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { GripVertical, Trash2 } from "lucide-react"
import { api } from "@/lib/api"
import { toast } from "sonner"
import IconPicker from "@/features/subscriptions/icon-picker"
import type {
  PaymentMethod,
  CreatePaymentMethodInput,
  UpdatePaymentMethodInput,
  ReorderPaymentMethodItem,
  UploadIconResponse,
} from "@/types"

export default function PaymentMethodManagement() {
  const { t } = useTranslation()

  const [methods, setMethods] = useState<PaymentMethod[]>([])
  const [addName, setAddName] = useState("")
  const [addIcon, setAddIcon] = useState("")
  const [addIconFile, setAddIconFile] = useState<File | null>(null)
  const [addLoading, setAddLoading] = useState(false)
  const [orderChanged, setOrderChanged] = useState(false)
  const [orderSaving, setOrderSaving] = useState(false)
  const [addIconPickerKey, setAddIconPickerKey] = useState(0)

  const dragFrom = useRef<number | null>(null)
  const dragTo = useRef<number | null>(null)

  useEffect(() => {
    api.get<PaymentMethod[]>("/payment-methods").then((list) => {
      setMethods(list ?? [])
    }).catch(() => void 0)
  }, [])

  async function uploadMethodIcon(id: number, file: File): Promise<string> {
    const formData = new FormData()
    formData.append("icon", file)
    const result = await api.uploadFile<UploadIconResponse>(
      `/payment-methods/${id}/icon`,
      formData
    )
    return result.icon
  }

  async function handleAddMethod(e: FormEvent) {
    e.preventDefault()

    const name = addName.trim()
    if (!name) {
      toast.error(t("settings.paymentMethodManagement.invalidName"))
      return
    }

    setAddLoading(true)
    try {
      const input: CreatePaymentMethodInput = {
        name,
        icon: addIconFile ? "" : addIcon,
        sort_order: methods.length,
      }
      const created = await api.post<PaymentMethod>("/payment-methods", input)

      let nextMethod = created
      if (addIconFile) {
        try {
          const icon = await uploadMethodIcon(created.id, addIconFile)
          nextMethod = { ...created, icon }
        } catch {
          toast.error(t("settings.paymentMethodManagement.iconUploadFailed"))
        }
      }

      setMethods((prev) => [...prev, nextMethod])
      setAddName("")
      setAddIcon("")
      setAddIconFile(null)
      setAddIconPickerKey((prev) => prev + 1)
      toast.success(t("settings.paymentMethodManagement.addSuccess"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("settings.paymentMethodManagement.invalidName"))
    } finally {
      setAddLoading(false)
    }
  }

  async function handleDeleteMethod(id: number) {
    if (!window.confirm(t("settings.paymentMethodManagement.deleteConfirm"))) {
      return
    }

    try {
      await api.delete(`/payment-methods/${id}`)
      setMethods((prev) => prev.filter((item) => item.id !== id))
      toast.success(t("settings.paymentMethodManagement.deleteSuccess"))
    } catch {
      void 0
    }
  }

  async function handleUpdateMethod(id: number, input: UpdatePaymentMethodInput) {
    try {
      const updated = await api.put<PaymentMethod>(`/payment-methods/${id}`, input)
      setMethods((prev) => prev.map((item) => (item.id === id ? updated : item)))
      toast.success(t("settings.paymentMethodManagement.updateSuccess"))
    } catch {
      void 0
    }
  }

  async function handleUploadMethodIcon(id: number, file: File) {
    try {
      const icon = await uploadMethodIcon(id, file)
      setMethods((prev) =>
        prev.map((item) => (item.id === id ? { ...item, icon } : item))
      )
      toast.success(t("settings.paymentMethodManagement.updateSuccess"))
    } catch {
      void 0
    }
  }

  function handleDragStart(index: number) {
    dragFrom.current = index
  }

  function handleDragOver(e: DragEvent<HTMLDivElement>, index: number) {
    e.preventDefault()
    dragTo.current = index
  }

  function handleDrop() {
    if (dragFrom.current === null || dragTo.current === null || dragFrom.current === dragTo.current) {
      return
    }

    const reordered = [...methods]
    const [moved] = reordered.splice(dragFrom.current, 1)
    reordered.splice(dragTo.current, 0, moved)
    setMethods(reordered)
    setOrderChanged(true)
    dragFrom.current = null
    dragTo.current = null
  }

  async function handleSaveOrder() {
    setOrderSaving(true)
    try {
      const payload: ReorderPaymentMethodItem[] = methods.map((item, index) => ({
        id: item.id,
        sort_order: index,
      }))
      await api.put("/payment-methods/reorder", payload)
      setMethods((prev) =>
        prev.map((item, index) => ({
          ...item,
          sort_order: index,
        }))
      )
      setOrderChanged(false)
      toast.success(t("settings.paymentMethodManagement.orderSaved"))
    } catch {
      void 0
    } finally {
      setOrderSaving(false)
    }
  }

  return (
    <div>
      <h2 className="text-sm font-medium">
        {t("settings.paymentMethodManagement.title")}
      </h2>
      <p className="text-sm text-muted-foreground mt-0.5">
        {t("settings.paymentMethodManagement.description")}
      </p>

      <div className="mt-4 space-y-2">
        {methods.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t("settings.paymentMethodManagement.empty")}
          </p>
        ) : (
          <>
            {methods.map((item, index) => (
              <div
                key={item.id}
                draggable
                onDragStart={() => handleDragStart(index)}
                onDragOver={(e) => handleDragOver(e, index)}
                onDrop={handleDrop}
                className="grid grid-cols-[1rem_2.5rem_minmax(0,1fr)_1.75rem] items-center gap-2 rounded-md border bg-card px-2 py-1.5"
              >
                <GripVertical className="size-4 text-muted-foreground shrink-0 cursor-grab" />

                <IconPicker
                  value={item.icon}
                  onChange={(value) => {
                    if (value !== item.icon) {
                      void handleUpdateMethod(item.id, { icon: value })
                    }
                  }}
                  onFileSelected={(file) => {
                    void handleUploadMethodIcon(item.id, file)
                  }}
                  maxFileSizeKB={64}
                />

                <Input
                  className="h-8 w-full text-sm"
                  defaultValue={item.name}
                  maxLength={50}
                  placeholder={t("settings.paymentMethodManagement.namePlaceholder")}
                  onBlur={(e) => {
                    const value = e.target.value.trim()
                    if (!value) {
                      e.target.value = item.name
                      toast.error(t("settings.paymentMethodManagement.invalidName"))
                      return
                    }
                    if (value !== item.name) {
                      void handleUpdateMethod(item.id, { name: value })
                    }
                  }}
                />

                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="size-7 text-muted-foreground hover:text-destructive"
                  onClick={() => void handleDeleteMethod(item.id)}
                >
                  <Trash2 className="size-3.5" />
                </Button>
              </div>
            ))}

            {orderChanged && (
              <Button
                size="sm"
                variant="outline"
                disabled={orderSaving}
                onClick={() => void handleSaveOrder()}
              >
                {orderSaving
                  ? t("settings.paymentMethodManagement.savingOrder")
                  : t("settings.paymentMethodManagement.saveOrder")}
              </Button>
            )}
          </>
        )}
      </div>

      <form onSubmit={handleAddMethod} className="mt-4 space-y-2">
        <div className="grid gap-1 sm:grid-cols-[minmax(0,1fr)_3rem_auto]">
          <Label className="text-xs text-muted-foreground">
            {t("settings.paymentMethodManagement.nameLabel")}
          </Label>
          <Label className="text-xs text-muted-foreground">
            {t("settings.paymentMethodManagement.iconLabel")}
          </Label>
          <Label className="text-xs text-transparent">
            {t("settings.paymentMethodManagement.addButton")}
          </Label>
        </div>

        <div className="grid items-center gap-2 sm:grid-cols-[minmax(0,1fr)_3rem_auto]">
          <Input
            className="w-full text-sm"
            value={addName}
            maxLength={50}
            placeholder={t("settings.paymentMethodManagement.namePlaceholder")}
            onChange={(e) => setAddName(e.target.value)}
          />

          <div className="flex justify-center">
            <IconPicker
              key={addIconPickerKey}
              value={addIcon}
              onChange={(value) => {
                setAddIcon(value)
                setAddIconFile(null)
              }}
              onFileSelected={(file) => {
                setAddIcon("")
                setAddIconFile(file)
              }}
              maxFileSizeKB={64}
            />
          </div>

          <Button type="submit" className="sm:min-w-20" disabled={addLoading || addName.trim() === ""}>
            {addLoading
              ? t("settings.paymentMethodManagement.adding")
              : t("settings.paymentMethodManagement.addButton")}
          </Button>
        </div>
      </form>
    </div>
  )
}
