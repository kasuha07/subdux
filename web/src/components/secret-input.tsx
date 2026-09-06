import { useState, type ComponentProps } from "react"
import { useTranslation } from "react-i18next"
import { Eye, EyeOff } from "lucide-react"

import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

const CONFIGURED_MASK_VALUE = "••••••••"

interface SecretInputProps extends Omit<ComponentProps<typeof Input>, "value" | "onChange"> {
  value: string
  configured: boolean
  onValueChange: (value: string) => void
}

export function SecretInput({
  configured,
  onValueChange,
  value,
  onBlur,
  onFocus,
  type,
  className,
  disabled,
  ...props
}: SecretInputProps) {
  const { t } = useTranslation()
  const [editing, setEditing] = useState(false)
  const [showSecret, setShowSecret] = useState(false)

  const isPassword = type === "password"
  const displayValue = editing
    ? value
    : value || (configured ? CONFIGURED_MASK_VALUE : "")

  const actualType = isPassword
    ? (showSecret ? "text" : "password")
    : type

  if (!isPassword) {
    return (
      <Input
        {...props}
        type={type}
        disabled={disabled}
        className={className}
        value={displayValue}
        onFocus={(event) => {
          setEditing(true)
          onFocus?.(event)
        }}
        onBlur={(event) => {
          setEditing(false)
          onBlur?.(event)
        }}
        onChange={(event) => onValueChange(event.target.value)}
      />
    )
  }

  return (
    <div className="relative">
      <Input
        {...props}
        type={actualType}
        disabled={disabled}
        className={cn("pr-9", className)}
        value={displayValue}
        onFocus={(event) => {
          setEditing(true)
          onFocus?.(event)
        }}
        onBlur={(event) => {
          setEditing(false)
          onBlur?.(event)
        }}
        onChange={(event) => onValueChange(event.target.value)}
      />
      <button
        type="button"
        disabled={disabled}
        onClick={() => setShowSecret((prev) => !prev)}
        className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded-xs p-0.5 text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50"
        aria-label={t(showSecret ? "common.hidePassword" : "common.showPassword")}
        title={t(showSecret ? "common.hidePassword" : "common.showPassword")}
        tabIndex={-1}
      >
        {showSecret ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
      </button>
    </div>
  )
}
