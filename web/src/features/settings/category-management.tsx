import { useState, useEffect, useRef, type DragEvent, type FormEvent } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { GripVertical, Trash2 } from "lucide-react"
import { api } from "@/lib/api"
import { toast } from "sonner"
import type { Category, CreateCategoryInput, ReorderCategoryItem } from "@/types"

export default function CategoryManagement() {
  const { t } = useTranslation()

  const [categories, setCategories] = useState<Category[]>([])
  const [addName, setAddName] = useState("")
  const [addLoading, setAddLoading] = useState(false)
  const [orderChanged, setOrderChanged] = useState(false)
  const [orderSaving, setOrderSaving] = useState(false)

  const dragFrom = useRef<number | null>(null)
  const dragTo = useRef<number | null>(null)

  useEffect(() => {
    api.get<Category[]>("/categories").then((list) => {
      setCategories(list ?? [])
    }).catch(() => void 0)
  }, [])

  async function handleAddCategory(e: FormEvent) {
    e.preventDefault()

    const name = addName.trim()
    if (!name) {
      toast.error(t("settings.categoryManagement.invalidName"))
      return
    }

    setAddLoading(true)
    try {
      const input: CreateCategoryInput = {
        name,
        display_order: categories.length,
      }
      const created = await api.post<Category>("/categories", input)
      setCategories((prev) => [...prev, created])
      setAddName("")
      toast.success(t("settings.categoryManagement.addSuccess"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("settings.categoryManagement.invalidName"))
    } finally {
      setAddLoading(false)
    }
  }

  async function handleDeleteCategory(id: number) {
    if (!window.confirm(t("settings.categoryManagement.deleteConfirm"))) {
      return
    }

    try {
      await api.delete(`/categories/${id}`)
      setCategories((prev) => prev.filter((item) => item.id !== id))
      toast.success(t("settings.categoryManagement.deleteSuccess"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.requestFailed"))
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

    const reordered = [...categories]
    const [moved] = reordered.splice(dragFrom.current, 1)
    reordered.splice(dragTo.current, 0, moved)
    setCategories(reordered)
    setOrderChanged(true)
    dragFrom.current = null
    dragTo.current = null
  }

  async function handleSaveOrder() {
    setOrderSaving(true)
    try {
      const payload: ReorderCategoryItem[] = categories.map((item, index) => ({
        id: item.id,
        sort_order: index,
      }))
      await api.put("/categories/reorder", payload)
      setCategories((prev) =>
        prev.map((item, index) => ({
          ...item,
          display_order: index,
        }))
      )
      setOrderChanged(false)
      toast.success(t("settings.categoryManagement.orderSaved"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.requestFailed"))
    } finally {
      setOrderSaving(false)
    }
  }

  return (
    <div>
      <h2 className="text-sm font-medium">
        {t("settings.categoryManagement.title")}
      </h2>
      <p className="text-sm text-muted-foreground mt-0.5">
        {t("settings.categoryManagement.description")}
      </p>

      <div className="mt-4 space-y-2">
        {categories.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t("settings.categoryManagement.empty")}
          </p>
        ) : (
          <>
            {categories.map((item, index) => (
              <div
                key={item.id}
                draggable
                onDragStart={() => handleDragStart(index)}
                onDragOver={(e) => handleDragOver(e, index)}
                onDrop={handleDrop}
                className="flex items-center gap-2 rounded-md border bg-card p-3"
              >
                <GripVertical className="size-4 text-muted-foreground cursor-grab" />
                <div className="flex-1">
                  <p className="text-sm font-medium">{item.name}</p>
                </div>
                <Button
                  size="icon-sm"
                  variant="ghost"
                  onClick={() => handleDeleteCategory(item.id)}
                >
                  <Trash2 className="size-4" />
                </Button>
              </div>
            ))}
            {orderChanged && (
              <Button
                size="sm"
                onClick={handleSaveOrder}
                disabled={orderSaving}
              >
                {orderSaving
                  ? t("settings.categoryManagement.savingOrder")
                  : t("settings.categoryManagement.saveOrder")}
              </Button>
            )}
          </>
        )}
      </div>

      <form onSubmit={handleAddCategory} className="mt-4 space-y-3">
        <div className="space-y-1.5">
          <Label htmlFor="category-name">
            {t("settings.categoryManagement.nameLabel")}
          </Label>
          <Input
            id="category-name"
            placeholder={t("settings.categoryManagement.namePlaceholder")}
            value={addName}
            onChange={(e) => setAddName(e.target.value)}
          />
        </div>
        <Button type="submit" size="sm" disabled={addLoading}>
          {addLoading
            ? t("settings.categoryManagement.adding")
            : t("settings.categoryManagement.addButton")}
        </Button>
      </form>
    </div>
  )
}
