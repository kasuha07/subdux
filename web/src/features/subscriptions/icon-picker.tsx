import { useState, useRef, useMemo, type ChangeEvent, type KeyboardEvent, type ReactNode } from "react"
import { useTranslation } from "react-i18next"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { Upload, X, Image as ImageIcon } from "lucide-react"
import { brandIcons, getBrandIconFromValue } from "@/lib/brand-icons"
import { emojiCategories } from "@/lib/emoji-data"

interface IconPickerProps {
  value: string
  onChange: (value: string) => void
  onFileSelected: (file: File) => void
  maxFileSizeKB?: number
  triggerSize?: "sm" | "md"
  allowImageUrl?: boolean
}

function renderPreview(value: string): ReactNode {
  if (!value) {
    return <ImageIcon className="size-5 text-muted-foreground" />
  }

  const brand = getBrandIconFromValue(value)
  if (brand) {
    return <brand.Icon size={20} color="default" />
  }

  if (value.startsWith("http://") || value.startsWith("https://")) {
    return <img src={value} alt="" className="h-6 w-6 object-contain rounded" />
  }

  if (value.startsWith("file:")) {
    const filename = value.slice("file:".length)
    if (filename && !filename.includes("/") && !filename.includes("\\")) {
      return <img src={`/uploads/icons/${filename}`} alt="" className="h-6 w-6 object-contain rounded" />
    }
  }

  if (value.includes(":")) {
    return <ImageIcon className="size-5 text-muted-foreground" />
  }

  return <span className="text-lg leading-none">{value}</span>
}

function isNonEmojiValue(v: string) {
  return (
    v.startsWith("http://") ||
    v.startsWith("https://") ||
    v.startsWith("file:") ||
    Boolean(getBrandIconFromValue(v))
  )
}

function isImageURLValue(value: string): boolean {
  return value.startsWith("http://") || value.startsWith("https://")
}

const suggestionServiceDomains = new Set(["google.com", "www.google.com", "icon.horse"])

function isValidDomain(hostname: string): boolean {
  if (!hostname || hostname.length > 253 || !hostname.includes(".")) {
    return false
  }

  return hostname.split(".").every((part) =>
    part.length > 0 &&
    part.length <= 63 &&
    /^[a-z0-9-]+$/i.test(part) &&
    !part.startsWith("-") &&
    !part.endsWith("-")
  )
}

function extractDomain(input: string): string | null {
  const trimmed = input.trim()
  if (!trimmed || trimmed.includes(" ")) {
    return null
  }

  const candidate = /^[a-z][a-z\d+.-]*:\/\//i.test(trimmed) ? trimmed : `https://${trimmed}`

  try {
    const hostname = new URL(candidate).hostname.toLowerCase().replace(/\.$/, "")
    if (!isValidDomain(hostname)) {
      return null
    }
    return hostname
  } catch {
    return null
  }
}

type UploadedImageFormat = "png" | "jpg" | "ico"

const allowedMimeByFormat: Record<UploadedImageFormat, string[]> = {
  png: ["image/png"],
  jpg: ["image/jpeg", "image/jpg", "image/pjpeg"],
  ico: ["image/x-icon", "image/vnd.microsoft.icon"],
}

function getFileExtension(name: string): string {
  const index = name.lastIndexOf(".")
  if (index < 0) return ""
  return name.slice(index).toLowerCase()
}

function detectFileFormat(headerBytes: Uint8Array): UploadedImageFormat | null {
  const isPNG = headerBytes.length >= 8 &&
    headerBytes[0] === 0x89 &&
    headerBytes[1] === 0x50 &&
    headerBytes[2] === 0x4E &&
    headerBytes[3] === 0x47 &&
    headerBytes[4] === 0x0D &&
    headerBytes[5] === 0x0A &&
    headerBytes[6] === 0x1A &&
    headerBytes[7] === 0x0A
  if (isPNG) return "png"

  const isJPG = headerBytes.length >= 3 &&
    headerBytes[0] === 0xFF &&
    headerBytes[1] === 0xD8 &&
    headerBytes[2] === 0xFF
  if (isJPG) return "jpg"

  const isICO = headerBytes.length >= 4 &&
    headerBytes[0] === 0x00 &&
    headerBytes[1] === 0x00 &&
    headerBytes[2] === 0x01 &&
    headerBytes[3] === 0x00
  if (isICO) return "ico"

  return null
}

function extensionMatchesFormat(extension: string, format: UploadedImageFormat): boolean {
  switch (format) {
    case "png":
      return extension === ".png"
    case "jpg":
      return extension === ".jpg" || extension === ".jpeg"
    case "ico":
      return extension === ".ico"
    default:
      return false
  }
}

export default function IconPicker({
  value,
  onChange,
  onFileSelected,
  maxFileSizeKB = 64,
  triggerSize = "md",
  allowImageUrl = false,
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
      icon.title.toLowerCase().includes(term) ||
      icon.slug.includes(term) ||
      icon.keywords.some((keyword) => keyword.toLowerCase().includes(term))
    )
  }, [brandSearch])

  const imageDomain = useMemo(() => {
    const domain = extractDomain(imageUrl)
    if (!domain || suggestionServiceDomains.has(domain)) {
      return null
    }
    return domain
  }, [imageUrl])
  const imageUrlSuggestions = useMemo(() => {
    if (!imageDomain) return []
    return [
      {
        key: "google",
        label: t("subscription.form.iconPicker.suggestions.google"),
        url: `https://www.google.com/s2/favicons?domain=${encodeURIComponent(imageDomain)}&sz=64`,
      },
      {
        key: "iconHorse",
        label: t("subscription.form.iconPicker.suggestions.iconHorse"),
        url: `https://icon.horse/icon/${encodeURIComponent(imageDomain)}`,
      },
    ]
  }, [imageDomain, t])

  function handleFileChange(e: ChangeEvent<HTMLInputElement>) {
    void validateAndSelectFile(e)
  }

  async function validateAndSelectFile(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setFileError("")

    const extension = getFileExtension(file.name)
    if (!extension) {
      setFileError(t("subscription.form.iconPicker.invalidType"))
      return
    }

    if (file.size > maxFileSizeKB * 1024) {
      setFileError(t("subscription.form.iconPicker.fileTooLarge", { size: maxFileSizeKB }))
      return
    }

    const headerBytes = new Uint8Array(await file.slice(0, 16).arrayBuffer())
    const detectedFormat = detectFileFormat(headerBytes)
    if (!detectedFormat || !extensionMatchesFormat(extension, detectedFormat)) {
      setFileError(t("subscription.form.iconPicker.invalidType"))
      return
    }

    const allowedMimes = allowedMimeByFormat[detectedFormat]
    if (file.type && !allowedMimes.includes(file.type)) {
      setFileError(t("subscription.form.iconPicker.invalidType"))
      return
    }

    const preview = URL.createObjectURL(file)
    setFilePreview(preview)
    onFileSelected(file)
  }

  function applyImageUrl(url: string) {
    setImageUrl(url)
    setFilePreview(null)
    setFileError("")
    onChange(url)
    setOpen(false)
  }

  function handleRemoveFile() {
    setFilePreview(null)
    setFileError("")
    if (fileInputRef.current) fileInputRef.current.value = ""
    onChange("")
  }

  function handleImageUrlSubmit() {
    const trimmed = imageUrl.trim()
    if (isImageURLValue(trimmed)) {
      applyImageUrl(trimmed)
    }
  }

  function handleImageUrlKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Enter") {
      event.preventDefault()
      handleImageUrlSubmit()
    }
  }

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen)

    if (!nextOpen || !allowImageUrl) {
      return
    }

    setImageUrl(isImageURLValue(value) ? value : "")
  }

  const emojiValue = isNonEmojiValue(value) ? "" : value
  const currentEmojis = emojiCategories[emojiCategory]?.emojis ?? []

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
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
                      const isSelected = value === brand.value
                      return (
                        <button
                          key={brand.value}
                          type="button"
                          title={brand.title}
                          className={`flex items-center justify-center rounded-md p-1.5 h-11 w-11 cursor-pointer transition-colors hover:bg-accent ${
                            isSelected ? "bg-accent ring-2 ring-primary" : ""
                          }`}
                          onClick={() => {
                            onChange(brand.value)
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
            {allowImageUrl && (
              <>
                <div className="space-y-2">
                  <Label className="text-xs">{t("subscription.form.iconPicker.urlLabel")}</Label>
                  <Input
                    type="url"
                    placeholder={t("subscription.form.iconPicker.urlPlaceholder")}
                    value={imageUrl}
                    onChange={(event) => setImageUrl(event.target.value)}
                    onBlur={handleImageUrlSubmit}
                    onKeyDown={handleImageUrlKeyDown}
                    className="h-8 text-sm"
                  />
                  {imageUrlSuggestions.length > 0 && (
                    <div className="space-y-1">
                      <p className="text-xs text-muted-foreground">
                        {t("subscription.form.iconPicker.suggestions.title", { domain: imageDomain })}
                      </p>
                      <div className="space-y-1">
                        {imageUrlSuggestions.map((suggestion) => {
                          const isSelected = value === suggestion.url
                          return (
                            <button
                              key={suggestion.key}
                              type="button"
                              className={`w-full flex items-center gap-2 rounded-md border px-2 py-1.5 text-left transition-colors hover:bg-accent ${
                                isSelected ? "border-primary bg-accent" : "border-border"
                              }`}
                              onPointerDown={(event) => event.preventDefault()}
                              onClick={() => applyImageUrl(suggestion.url)}
                            >
                              <img src={suggestion.url} alt="" className="h-4 w-4 rounded-sm object-contain" />
                              <span className="text-xs text-foreground">{suggestion.label}</span>
                            </button>
                          )
                        })}
                      </div>
                    </div>
                  )}
                </div>

                <div className="relative flex items-center gap-2">
                  <Separator className="flex-1" />
                  <span className="text-xs text-muted-foreground px-1">or</span>
                  <Separator className="flex-1" />
                </div>
              </>
            )}

            <Label className="text-xs">{t("subscription.form.iconPicker.uploadLabel")}</Label>

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
                  accept=".png,.jpg,.jpeg,.ico,image/png,image/jpeg,image/x-icon,image/vnd.microsoft.icon"
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
