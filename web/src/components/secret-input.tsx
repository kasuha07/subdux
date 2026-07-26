import { useState, type ComponentProps } from "react"

import { Input } from "@/components/ui/input"

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
  ...props
}: SecretInputProps) {
  const [editing, setEditing] = useState(false)

  const displayValue = editing
    ? value
    : value || (configured ? CONFIGURED_MASK_VALUE : "")

  return (
    <Input
      {...props}
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

