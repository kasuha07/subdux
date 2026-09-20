import {
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type KeyboardEvent,
  type ReactNode,
} from "react"
import { useTranslation } from "react-i18next"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { Tooltip } from "@/components/ui/tooltip"
import { AsyncBrandIcon } from "@/components/async-brand-icon"
import { Upload, X, Image as ImageIcon, Loader2, Smile, Sparkles, Search } from "lucide-react"
import { isAsyncBrandIconValue } from "@/lib/brand-icons/async-value"
import type { BrandIcon } from "@/lib/brand-icons/types"
import { emojiCategories } from "@/lib/emoji-data"
import { buildIconProxySuggestionURL, isRenderableImageURL } from "@/lib/icon-proxy"
import { cn } from "@/lib/utils"

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

  if (isAsyncBrandIconValue(value)) {
    return (
      <AsyncBrandIcon
        value={value}
        size={20}
        color="default"
        fallback={<ImageIcon className="size-5 text-muted-foreground" />}
      />
    )
  }

  if (isRenderableImageURL(value)) {
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
    v.startsWith("/api/icon-proxy/") ||
    v.startsWith("file:") ||
    isAsyncBrandIconValue(v)
  )
}

function loadBrandIconsCatalog(): Promise<BrandIcon[]> {
  return import("@/lib/brand-icons").then((module) => module.loadBrandIconsCatalog())
}

function isImageURLValue(value: string): boolean {
  return isRenderableImageURL(value)
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
  const [pickerTab, setPickerTab] = useState<"emoji" | "brand" | "image">("emoji")
  const [brandSearch, setBrandSearch] = useState("")
  const [brandIcons, setBrandIcons] = useState<BrandIcon[] | null>(null)
  const [emojiCategory, setEmojiCategory] = useState(0)
  const [imageUrl, setImageUrl] = useState("")
  const [filePreview, setFilePreview] = useState<string | null>(null)
  const [fileError, setFileError] = useState("")
  const fileInputRef = useRef<HTMLInputElement>(null)
  const tabsListRef = useRef<HTMLDivElement>(null)
  const categoryListRef = useRef<HTMLDivElement>(null)
  const emojiGridRef = useRef<HTMLDivElement>(null)

  const [indicatorStyle, setIndicatorStyle] = useState<{
    left: number
    top: number
    width: number
    height: number
  } | null>(null)
  const [isTransitionReady, setIsTransitionReady] = useState(false)

  const [categoryIndicator, setCategoryIndicator] = useState<{
    left: number
    top: number
    width: number
    height: number
  } | null>(null)
  const [isCategoryTransitionReady, setIsCategoryTransitionReady] = useState(false)

  useLayoutEffect(() => {
    const list = tabsListRef.current
    if (!list || !open) return

    const updateIndicator = () => {
      const activeTrigger = list.querySelector<HTMLElement>(
        '[data-slot="tabs-trigger"][data-state="active"]'
      )
      if (activeTrigger) {
        setIndicatorStyle({
          left: activeTrigger.offsetLeft,
          top: activeTrigger.offsetTop,
          width: activeTrigger.offsetWidth,
          height: activeTrigger.offsetHeight,
        })
      }
    }

    updateIndicator()

    const rafId = requestAnimationFrame(() => {
      setIsTransitionReady(true)
    })

    const observer = new ResizeObserver(updateIndicator)
    observer.observe(list)

    return () => {
      cancelAnimationFrame(rafId)
      observer.disconnect()
    }
  }, [pickerTab, open])

  useLayoutEffect(() => {
    const list = categoryListRef.current
    if (!list || pickerTab !== "emoji" || !open) return

    const updateCategoryIndicator = () => {
      const activeBtn = list.querySelector<HTMLElement>(`[data-category-index="${emojiCategory}"]`)
      if (activeBtn) {
        setCategoryIndicator({
          left: activeBtn.offsetLeft,
          top: activeBtn.offsetTop,
          width: activeBtn.offsetWidth,
          height: activeBtn.offsetHeight,
        })
      }
    }

    updateCategoryIndicator()

    const rafId = requestAnimationFrame(() => {
      setIsCategoryTransitionReady(true)
    })

    const observer = new ResizeObserver(updateCategoryIndicator)
    observer.observe(list)

    return () => {
      cancelAnimationFrame(rafId)
      observer.disconnect()
    }
  }, [emojiCategory, pickerTab, open])


  function handleCategoryClick(idx: number) {
    setEmojiCategory(idx)
    if (emojiGridRef.current) {
      emojiGridRef.current.scrollTop = 0
    }
    const btn = categoryListRef.current?.querySelector<HTMLElement>(`[data-category-index="${idx}"]`)
    btn?.scrollIntoView({ behavior: "smooth", block: "nearest", inline: "center" })
  }

  const filteredIcons = useMemo(() => {
    if (!brandIcons) return []
    if (!brandSearch.trim()) return brandIcons
    const term = brandSearch.toLowerCase()
    return brandIcons.filter((icon) =>
      icon.title.toLowerCase().includes(term) ||
      icon.slug.includes(term) ||
      icon.keywords.some((keyword) => keyword.toLowerCase().includes(term))
    )
  }, [brandIcons, brandSearch])

  useEffect(() => {
    if (!open || pickerTab !== "brand" || brandIcons) {
      return
    }

    let cancelled = false
    loadBrandIconsCatalog()
      .then((icons) => {
        if (!cancelled) {
          setBrandIcons(icons)
        }
      })
      .catch(() => {
        if (!cancelled) {
          setBrandIcons([])
        }
      })

    return () => {
      cancelled = true
    }
  }, [brandIcons, open, pickerTab])

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
        url: buildIconProxySuggestionURL("google", imageDomain),
      },
      {
        key: "iconHorse",
        label: t("subscription.form.iconPicker.suggestions.iconHorse"),
        url: buildIconProxySuggestionURL("icon-horse", imageDomain),
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

    if (!nextOpen) {
      setBrandSearch("")
      setIsTransitionReady(false)
      setIsCategoryTransitionReady(false)
      setIndicatorStyle(null)
      setCategoryIndicator(null)
      return
    }

    if (allowImageUrl) {
      setImageUrl(isImageURLValue(value) ? value : "")
    }

    if (isAsyncBrandIconValue(value)) {
      setPickerTab("brand")
    } else if (isNonEmojiValue(value)) {
      setPickerTab("image")
    } else {
      setPickerTab("emoji")
      if (value) {
        const catIdx = emojiCategories.findIndex((c) => c.emojis.includes(value))
        if (catIdx !== -1) {
          setEmojiCategory(catIdx)
        }
      }
    }
  }

  const emojiValue = isNonEmojiValue(value) ? "" : value
  const currentEmojis = emojiCategories[emojiCategory]?.emojis ?? []

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className={cn(
            "rounded-lg border border-input flex items-center justify-center transition-all duration-150 cursor-pointer hover:border-primary/50 hover:bg-accent/40 active:scale-95 focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-ring",
            open && "border-primary ring-2 ring-primary/20",
            triggerSize === "sm" ? "h-9 w-9" : "h-10 w-10"
          )}
        >
          {filePreview ? (
            <img src={filePreview} alt="" className="h-6 w-6 object-contain rounded" />
          ) : (
            renderPreview(value)
          )}
        </button>
      </PopoverTrigger>

      <PopoverContent
        className="w-80 p-0 overflow-hidden shadow-lg border border-border/80"
        align="start"
        onWheel={(e) => e.stopPropagation()}
      >
        <Tabs
          value={pickerTab}
          onValueChange={(value) => setPickerTab(value as "emoji" | "brand" | "image")}
          className="flex flex-col gap-0"
        >
          <div className="p-2 pb-1.5 border-b bg-muted/25">
            <TabsList
              ref={tabsListRef}
              className={cn(
                "relative grid w-full grid-cols-3 h-9 p-1 rounded-lg bg-muted/80 text-muted-foreground",
                indicatorStyle && "icon-picker-tabs-list"
              )}
            >
              {indicatorStyle && (
                <span
                  aria-hidden="true"
                  className={cn(
                    "pointer-events-none absolute rounded-md bg-background shadow-xs dark:bg-input/50 dark:border dark:border-border/60",
                    isTransitionReady && "transition-all duration-200 ease-[cubic-bezier(0.22,1,0.36,1)]"
                  )}
                  style={{
                    left: `${indicatorStyle.left}px`,
                    top: `${indicatorStyle.top}px`,
                    width: `${indicatorStyle.width}px`,
                    height: `${indicatorStyle.height}px`,
                  }}
                />
              )}
              <TabsTrigger
                value="emoji"
                className="relative z-10 flex items-center justify-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium select-none transition-colors hover:text-foreground data-[state=active]:text-foreground data-[state=active]:font-semibold [&[data-state=active]_svg]:scale-110 [&_svg]:transition-transform [&_svg]:duration-150"
              >
                <Smile className="size-3.5 shrink-0" />
                <span>{t("subscription.form.iconPicker.tabs.emoji")}</span>
              </TabsTrigger>
              <TabsTrigger
                value="brand"
                className="relative z-10 flex items-center justify-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium select-none transition-colors hover:text-foreground data-[state=active]:text-foreground data-[state=active]:font-semibold [&[data-state=active]_svg]:scale-110 [&_svg]:transition-transform [&_svg]:duration-150"
              >
                <Sparkles className="size-3.5 shrink-0" />
                <span>{t("subscription.form.iconPicker.tabs.brand")}</span>
              </TabsTrigger>
              <TabsTrigger
                value="image"
                className="relative z-10 flex items-center justify-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium select-none transition-colors hover:text-foreground data-[state=active]:text-foreground data-[state=active]:font-semibold [&[data-state=active]_svg]:scale-110 [&_svg]:transition-transform [&_svg]:duration-150"
              >
                <ImageIcon className="size-3.5 shrink-0" />
                <span>{t("subscription.form.iconPicker.tabs.image")}</span>
              </TabsTrigger>
            </TabsList>
          </div>

          {/* ── Emoji tab ── */}
          <TabsContent value="emoji" className="icon-picker-tab-content p-0 m-0 outline-none">
            <div className="flex flex-col h-[320px]">
              {/* category strip */}
              <div
                ref={categoryListRef}
                className="relative flex items-center gap-1 px-2 pt-2 pb-1.5 border-b bg-muted/10 overflow-x-auto scrollbar-none shrink-0"
              >
                {categoryIndicator && (
                  <span
                    aria-hidden="true"
                    className={cn(
                      "pointer-events-none absolute rounded-md bg-accent shadow-xs border border-border/40",
                      isCategoryTransitionReady && "transition-all duration-200 ease-[cubic-bezier(0.22,1,0.36,1)]"
                    )}
                    style={{
                      left: `${categoryIndicator.left}px`,
                      top: `${categoryIndicator.top}px`,
                      width: `${categoryIndicator.width}px`,
                      height: `${categoryIndicator.height}px`,
                    }}
                  />
                )}
                {emojiCategories.map((cat, idx) => (
                  <Tooltip
                    key={cat.key}
                    content={t(`subscription.form.iconPicker.categories.${cat.key}`, cat.label)}
                    side="bottom"
                  >
                    <button
                      type="button"
                      data-category-index={idx}
                      onClick={() => handleCategoryClick(idx)}
                      className={cn(
                        "relative z-10 shrink-0 text-base h-7 w-7.5 flex items-center justify-center rounded-md cursor-pointer select-none transition-all duration-150 active:scale-90 hover:scale-110",
                        emojiCategory === idx
                          ? "text-foreground font-semibold"
                          : "text-muted-foreground hover:text-foreground opacity-75 hover:opacity-100",
                        !categoryIndicator && emojiCategory === idx && "bg-accent"
                      )}
                      aria-label={t(`subscription.form.iconPicker.categories.${cat.key}`, cat.label)}
                    >
                      {cat.label}
                    </button>
                  </Tooltip>
                ))}
              </div>

              {/* emoji grid - scrollable */}
              <div
                ref={emojiGridRef}
                className="flex-1 min-h-0 overflow-y-auto overflow-x-hidden p-2"
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
                <div key={emojiCategory} className="grid grid-cols-8 gap-0.5 icon-picker-grid-enter">
                  {currentEmojis.map((emoji) => (
                    <button
                      key={emoji}
                      type="button"
                      title={emoji}
                      onClick={() => {
                        onChange(emoji)
                        setOpen(false)
                      }}
                      className={cn(
                        "flex items-center justify-center rounded-md text-xl h-9 w-9 cursor-pointer transition-all duration-150 hover:bg-accent hover:scale-115 active:scale-90",
                        emojiValue === emoji ? "bg-accent ring-2 ring-primary shadow-xs font-semibold" : ""
                      )}
                    >
                      {emoji}
                    </button>
                  ))}
                </div>
              </div>

              {/* current value display */}
              {emojiValue && (
                <div className="border-t px-3 py-1.5 flex items-center gap-2 shrink-0 bg-muted/15 animate-in fade-in-0 duration-150">
                  <span className="text-2xl leading-none">{emojiValue}</span>
                  <Tooltip content={t("common.clear")}>
                    <button
                      type="button"
                      onClick={() => onChange("")}
                      className="ml-auto rounded-md p-1 text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                      aria-label={t("common.clear")}
                    >
                      <X className="size-3.5" />
                    </button>
                  </Tooltip>
                </div>
              )}
            </div>
          </TabsContent>

          {/* ── Brand tab ── */}
          <TabsContent value="brand" className="icon-picker-tab-content p-0 m-0 outline-none">
            <div className="flex flex-col h-[320px]">
              <div className="relative px-2.5 pt-2 pb-1.5 shrink-0">
                <Search className="absolute left-4.5 top-1/2 mt-0.25 -translate-y-1/2 size-3.5 text-muted-foreground pointer-events-none" />
                <Input
                  placeholder={t("subscription.form.iconPicker.searchPlaceholder")}
                  value={brandSearch}
                  onChange={(e) => setBrandSearch(e.target.value)}
                  className={`h-8 text-xs pl-7 ${brandSearch ? "pr-7" : ""}`}
                />
                {brandSearch && (
                  <Tooltip content={t("common.clear")}>
                    <button
                      type="button"
                      onClick={() => setBrandSearch("")}
                      className="absolute right-4 top-1/2 mt-0.25 -translate-y-1/2 rounded-xs p-0.5 text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-ring"
                      aria-label={t("common.clear")}
                    >
                      <X className="size-3.5" />
                    </button>
                  </Tooltip>
                )}
              </div>
              <div
                className="flex-1 min-h-0 overflow-y-auto overflow-x-hidden p-2"
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
                {brandIcons === null ? (
                  <div className="flex items-center justify-center h-full gap-2 text-xs text-muted-foreground" role="status">
                    <Loader2 className="size-4 animate-spin" aria-hidden="true" />
                    <span>{t("common.loading")}</span>
                  </div>
                ) : filteredIcons.length === 0 ? (
                  <div className="flex items-center justify-center h-full text-xs text-muted-foreground">
                    {t("subscription.form.iconPicker.noResults")}
                  </div>
                ) : (
                  <div className="grid grid-cols-6 gap-1 p-0.5 icon-picker-grid-enter">
                    {filteredIcons.map((brand) => {
                      const isSelected = value === brand.value
                      return (
                        <button
                          key={brand.value}
                          type="button"
                          title={brand.title}
                          className={cn(
                            "flex items-center justify-center rounded-md p-1.5 h-11 w-11 cursor-pointer transition-all duration-150 hover:bg-accent hover:scale-108 active:scale-95",
                            isSelected ? "bg-accent ring-2 ring-primary shadow-xs" : ""
                          )}
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
          <TabsContent value="image" className="icon-picker-tab-content p-0 m-0 outline-none">
            <div
              className="flex flex-col h-[320px] overflow-y-auto overflow-x-hidden p-3 space-y-3"
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
              {allowImageUrl && (
                <>
                  <div className="space-y-2 shrink-0">
                    <Label className="text-xs font-medium">{t("subscription.form.iconPicker.urlLabel")}</Label>
                    <Input
                      type="url"
                      placeholder={t("subscription.form.iconPicker.urlPlaceholder")}
                      value={imageUrl}
                      onChange={(event) => setImageUrl(event.target.value)}
                      onBlur={handleImageUrlSubmit}
                      onKeyDown={handleImageUrlKeyDown}
                      className="h-8 text-xs"
                    />
                    {imageUrlSuggestions.length > 0 && (
                      <div className="space-y-1.5 pt-1">
                        <p className="text-[11px] text-muted-foreground">
                          {t("subscription.form.iconPicker.suggestions.title", { domain: imageDomain })}
                        </p>
                        <div className="space-y-1">
                          {imageUrlSuggestions.map((suggestion) => {
                            const isSelected = value === suggestion.url
                            return (
                              <button
                                key={suggestion.key}
                                type="button"
                                className={cn(
                                  "w-full flex items-center gap-2 rounded-md border px-2.5 py-1.5 text-left transition-all duration-150 hover:bg-accent hover:border-primary/50 active:scale-[0.99] cursor-pointer",
                                  isSelected ? "border-primary bg-accent ring-1 ring-primary/30" : "border-border"
                                )}
                                onPointerDown={(event) => event.preventDefault()}
                                onClick={() => applyImageUrl(suggestion.url)}
                              >
                                <img src={suggestion.url} alt="" className="h-4 w-4 rounded-xs object-contain shrink-0" />
                                <span className="text-xs text-foreground truncate">{suggestion.label}</span>
                              </button>
                            )
                          })}
                        </div>
                      </div>
                    )}
                  </div>

                  <div className="relative flex items-center gap-2 shrink-0 py-0.5">
                    <Separator className="flex-1" />
                    <span className="text-[11px] text-muted-foreground px-1 uppercase tracking-wider">or</span>
                    <Separator className="flex-1" />
                  </div>
                </>
              )}

              <div className="space-y-2 shrink-0">
                <Label className="text-xs font-medium">{t("subscription.form.iconPicker.uploadLabel")}</Label>

                {filePreview ? (
                  <div className="flex items-center gap-3 rounded-lg border p-2.5 bg-muted/15">
                    <img src={filePreview} alt="" className="h-10 w-10 object-contain rounded-md border bg-background p-0.5" />
                    <div className="min-w-0 flex-1">
                      <p className="text-xs font-medium text-foreground truncate">
                        {t("subscription.form.iconPicker.uploadLabel")}
                      </p>
                      <p className="text-[11px] text-muted-foreground">
                        {t("subscription.form.iconPicker.uploadHint", { size: maxFileSizeKB })}
                      </p>
                    </div>
                    <Tooltip content={t("common.clear")}>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        className="shrink-0 text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                        onClick={handleRemoveFile}
                        aria-label={t("common.clear")}
                      >
                        <X className="size-4" />
                      </Button>
                    </Tooltip>
                  </div>
                ) : (
                  <label className="group border-2 border-dashed border-muted-foreground/25 hover:border-primary/60 hover:bg-accent/20 rounded-lg p-4 flex flex-col items-center gap-2 cursor-pointer transition-all duration-200">
                    <div className="rounded-full bg-muted/60 p-2 text-muted-foreground group-hover:text-primary group-hover:bg-primary/10 transition-colors duration-200">
                      <Upload className="size-4.5 group-hover:-translate-y-0.5 transition-transform duration-200" />
                    </div>
                    <span className="text-xs font-medium text-foreground group-hover:text-primary transition-colors">
                      {t("subscription.form.iconPicker.uploadLabel")}
                    </span>
                    <span className="text-[11px] text-muted-foreground text-center">
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
                  <p className="text-xs text-destructive animate-in fade-in-0 duration-150">{fileError}</p>
                )}
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </PopoverContent>
    </Popover>
  )
}
