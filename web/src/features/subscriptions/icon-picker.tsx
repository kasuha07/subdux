import { useState, useRef, useMemo, type ChangeEvent, type KeyboardEvent, type ReactNode } from "react"
import { useTranslation } from "react-i18next"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { Upload, X, Image as ImageIcon } from "lucide-react"
import { brandIcons, getBrandIcon } from "@/lib/brand-icons"
import { emojiCategories } from "@/lib/emoji-data"

interface IconPickerProps {
  value: string
  onChange: (value: string) => void
  onFileSelected: (file: File) => void
  maxFileSizeKB?: number
  triggerSize?: "sm" | "md"
}

function renderPreview(value: string): ReactNode {
  if (!value) {
    return <ImageIcon className="size-5 text-muted-foreground" />
  }

  if (value.startsWith("si:")) {
    const brand = getBrandIcon(value.slice(3))
    if (brand) {
      return <brand.Icon size={20} color="default" />
    }
    return <ImageIcon className="size-5 text-muted-foreground" />
  }

  if (value.startsWith("http://") || value.startsWith("https://")) {
    return <img src={value} alt="" className="h-6 w-6 object-contain rounded" />
  }

  if (value.startsWith("assets/")) {
    const assetPath = value.slice("assets/".length)
    return <img src={`/uploads/${assetPath}`} alt="" className="h-6 w-6 object-contain rounded" />
  }

  return <span className="text-lg leading-none">{value}</span>
}

function isNonEmojiValue(v: string) {
  return v.startsWith("si:") || v.startsWith("http") || v.startsWith("assets/")
}

export default function IconPicker({
  value,
  onChange,
  onFileSelected,
  maxFileSizeKB = 64,
  triggerSize = "md",
}: IconPickerProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [brandSearch, setBrandSearch] = useState("")
  const [emojiCategory, setEmojiCategory] = useState(0)
  const [imageUrl, setImageUrl] = useState("")
  const [filePreview, setFilePreview] = useState<string | null>(null)
  const [fileError, setFileError] = useState("")
  const fileInputRef = useRef<HTMLInputElement>(null)

  const filteredIcons = useMemo(() => {
    if (!brandSearch.trim()) return brandIcons
    const term = brandSearch.toLowerCase()
    return brandIcons.filter((icon) =>
      icon.title.toLowerCase().includes(term) || icon.slug.includes(term)
    )
  }, [brandSearch])

  function handleFileChange(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setFileError("")

    if (file.type !== "image/png" && file.type !== "image/jpeg") {
      setFileError(t("subscription.form.iconPicker.invalidType"))
      return
    }

    if (file.size > maxFileSizeKB * 1024) {
      setFileError(t("subscription.form.iconPicker.fileTooLarge", { size: maxFileSizeKB }))
      return
    }

    const preview = URL.createObjectURL(file)
    setFilePreview(preview)
    onFileSelected(file)
  }

  function handleRemoveFile() {
    setFilePreview(null)
    setFileError("")
    if (fileInputRef.current) fileInputRef.current.value = ""
    onChange("")
  }

  function handleImageUrlSubmit() {
    const trimmed = imageUrl.trim()
    if (trimmed.startsWith("http://") || trimmed.startsWith("https://")) {
      onChange(trimmed)
      setOpen(false)
    }
  }

  function handleImageUrlKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      e.preventDefault()
      handleImageUrlSubmit()
    }
  }

  const emojiValue = isNonEmojiValue(value) ? "" : value
  const currentEmojis = emojiCategories[emojiCategory]?.emojis ?? []

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className={`rounded-lg border border-input flex items-center justify-center hover:border-primary/50 transition-colors cursor-pointer ${
            triggerSize === "sm" ? "h-9 w-9" : "h-10 w-10"
          }`}
        >
          {filePreview ? (
            <img src={filePreview} alt="" className="h-6 w-6 object-contain rounded" />
          ) : (
            renderPreview(value)
          )}
        </button>
      </PopoverTrigger>

      <PopoverContent className="w-80 p-0" align="start" onWheel={(e) => e.stopPropagation()}>
        <Tabs defaultValue="emoji">
          <TabsList className="w-full rounded-none border-b bg-transparent h-auto p-0">
            <TabsTrigger
              value="emoji"
              className="flex-1 rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent py-2 text-xs"
            >
              {t("subscription.form.iconPicker.tabs.emoji")}
            </TabsTrigger>
            <TabsTrigger
              value="brand"
              className="flex-1 rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent py-2 text-xs"
            >
              {t("subscription.form.iconPicker.tabs.brand")}
            </TabsTrigger>
            <TabsTrigger
              value="image"
              className="flex-1 rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent py-2 text-xs"
            >
              {t("subscription.form.iconPicker.tabs.image")}
            </TabsTrigger>
          </TabsList>

          {/* ── Emoji tab ── */}
          <TabsContent value="emoji" className="p-0 m-0">
            <div className="flex flex-col" style={{ maxHeight: "320px" }}>
              {/* category strip */}
              <div className="flex gap-0.5 px-2 pt-2 pb-1 border-b overflow-x-auto shrink-0">
                {emojiCategories.map((cat, idx) => (
                  <button
                    key={cat.key}
                    type="button"
                    onClick={() => setEmojiCategory(idx)}
                    className={`shrink-0 text-base px-1.5 py-0.5 rounded transition-colors hover:bg-accent ${
                      emojiCategory === idx ? "bg-accent" : ""
                    }`}
                  >
                    {cat.label}
                  </button>
                ))}
              </div>
              {/* emoji grid - scrollable */}
              <div 
                className="overflow-y-auto overflow-x-hidden p-2" 
                style={{ height: "240px" }}
                onWheel={(e) => {
                  e.stopPropagation()
                  const target = e.currentTarget
                  const isAtTop = target.scrollTop === 0
                  const isAtBottom = target.scrollTop + target.clientHeight >= target.scrollHeight - 1
                  if ((isAtTop && e.deltaY < 0) || (isAtBottom && e.deltaY > 0)) {
                    e.preventDefault()
                  }
                }}
              >
                <div className="grid grid-cols-8 gap-0.5">
                  {currentEmojis.map((emoji) => (
                    <button
                      key={emoji}
                      type="button"
                      title={emoji}
                      onClick={() => {
                        onChange(emoji)
                        setOpen(false)
                      }}
                      className={`flex items-center justify-center rounded text-xl h-9 w-9 hover:bg-accent transition-colors ${
                        emojiValue === emoji ? "bg-accent ring-2 ring-primary" : ""
                      }`}
                    >
                      {emoji}
                    </button>
                  ))}
                </div>
              </div>
              {/* current value display */}
              {emojiValue && (
                <div className="border-t px-3 py-2 flex items-center gap-2 shrink-0">
                  <span className="text-3xl leading-none">{emojiValue}</span>
                  <button
                    type="button"
                    onClick={() => onChange("")}
                    className="ml-auto text-muted-foreground hover:text-foreground"
                  >
                    <X className="size-3.5" />
                  </button>
                </div>
              )}
            </div>
          </TabsContent>

          {/* ── Brand tab ── */}
          <TabsContent value="brand" className="p-0 m-0">
            <div className="flex flex-col" style={{ maxHeight: "320px" }}>
              <div className="px-2 pt-2 pb-1 shrink-0">
                <Input
                  placeholder={t("subscription.form.iconPicker.searchPlaceholder")}
                  value={brandSearch}
                  onChange={(e) => setBrandSearch(e.target.value)}
                  className="h-8 text-sm"
                />
              </div>
              <div 
                className="overflow-y-auto overflow-x-hidden" 
                style={{ height: "272px" }}
                onWheel={(e) => {
                  e.stopPropagation()
                  const target = e.currentTarget
                  const isAtTop = target.scrollTop === 0
                  const isAtBottom = target.scrollTop + target.clientHeight >= target.scrollHeight - 1
                  if ((isAtTop && e.deltaY < 0) || (isAtBottom && e.deltaY > 0)) {
                    e.preventDefault()
                  }
                }}
              >
                {filteredIcons.length === 0 ? (
                  <div className="flex items-center justify-center h-full text-sm text-muted-foreground">
                    {t("subscription.form.iconPicker.noResults")}
                  </div>
                ) : (
                  <div className="grid grid-cols-6 gap-1 p-2">
                    {filteredIcons.map((brand) => {
                      const isSelected = value === `si:${brand.slug}`
                      return (
                        <button
                          key={brand.slug}
                          type="button"
                          title={brand.title}
                          className={`flex items-center justify-center rounded-md p-1.5 h-11 w-11 cursor-pointer transition-colors hover:bg-accent ${
                            isSelected ? "bg-accent ring-2 ring-primary" : ""
                          }`}
                          onClick={() => {
                            onChange(`si:${brand.slug}`)
                            setOpen(false)
                          }}
                        >
                          <brand.Icon size={22} color="default" />
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>
            </div>
          </TabsContent>

          {/* ── Image tab ── */}
          <TabsContent value="image" className="p-3 space-y-3">
            <div className="space-y-2">
              <Label className="text-xs">{t("subscription.form.iconPicker.urlLabel")}</Label>
              <Input
                type="url"
                placeholder={t("subscription.form.iconPicker.urlPlaceholder")}
                value={imageUrl}
                onChange={(e) => setImageUrl(e.target.value)}
                onBlur={handleImageUrlSubmit}
                onKeyDown={handleImageUrlKeyDown}
                className="h-8 text-sm"
              />
            </div>

            <div className="relative flex items-center gap-2">
              <Separator className="flex-1" />
              <span className="text-xs text-muted-foreground px-1">or</span>
              <Separator className="flex-1" />
            </div>

            {filePreview ? (
              <div className="flex items-center gap-3 rounded-lg border p-3">
                <img src={filePreview} alt="" className="h-10 w-10 object-contain rounded" />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  className="ml-auto"
                  onClick={handleRemoveFile}
                >
                  <X className="size-4" />
                </Button>
              </div>
            ) : (
              <label className="border-2 border-dashed border-muted-foreground/25 rounded-lg p-4 flex flex-col items-center gap-2 cursor-pointer hover:border-primary/50 transition-colors">
                <Upload className="size-5 text-muted-foreground" />
                <span className="text-sm text-muted-foreground">
                  {t("subscription.form.iconPicker.uploadLabel")}
                </span>
                <span className="text-xs text-muted-foreground">
                  {t("subscription.form.iconPicker.uploadHint", { size: maxFileSizeKB })}
                </span>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/png,image/jpeg"
                  className="hidden"
                  onChange={handleFileChange}
                />
              </label>
            )}

            {fileError && (
              <p className="text-xs text-destructive">{fileError}</p>
            )}
          </TabsContent>
        </Tabs>
      </PopoverContent>
    </Popover>
  )
}
