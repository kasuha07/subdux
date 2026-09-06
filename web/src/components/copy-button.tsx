import { useState, useEffect } from "react"
import { Check, Copy } from "lucide-react"

import { Button, type ButtonProps } from "@/components/ui/button"
import { toast } from "@/lib/toast"
import { cn } from "@/lib/utils"

export interface CopyButtonProps extends Omit<ButtonProps, "onClick"> {
  text: string
  label?: string
  copiedLabel?: string
  showToast?: boolean
  toastMessage?: string
  successToast?: string
  onCopied?: () => void
}

export function CopyButton({
  text,
  label = "Copy",
  copiedLabel = "Copied",
  showToast = false,
  toastMessage,
  successToast,
  variant = "outline",
  size = "icon-sm",
  className,
  onCopied,
  children,
  ...props
}: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!copied) return
    const timer = window.setTimeout(() => setCopied(false), 2000)
    return () => window.clearTimeout(timer)
  }, [copied])

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      if (successToast || showToast) {
        toast.success(successToast || toastMessage || copiedLabel)
      }
      onCopied?.()
    } catch {
      // Ignore copy error or unsupported environment
    }
  }

  return (
    <Button
      type="button"
      variant={variant}
      size={size}
      onClick={handleCopy}
      className={cn(
        "transition-colors",
        copied && "text-emerald-600 dark:text-emerald-400 border-emerald-500/30",
        className
      )}
      aria-label={copied ? copiedLabel : label}
      title={copied ? copiedLabel : label}
      {...props}
    >
      {children ?? (
        copied ? (
          <Check className="size-3.5 shrink-0" aria-hidden="true" />
        ) : (
          <Copy className="size-3.5 shrink-0" aria-hidden="true" />
        )
      )}
    </Button>
  )
}
