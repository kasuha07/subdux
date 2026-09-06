import { useState, forwardRef, type ComponentProps } from "react"
import { useTranslation } from "react-i18next"
import { Eye, EyeOff } from "lucide-react"

import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

export const PasswordInput = forwardRef<HTMLInputElement, ComponentProps<typeof Input>>(
  function PasswordInput({ className, disabled, ...props }, ref) {
    const { t } = useTranslation()
    const [showPassword, setShowPassword] = useState(false)

    return (
      <div className="relative">
        <Input
          {...props}
          ref={ref}
          disabled={disabled}
          type={showPassword ? "text" : "password"}
          className={cn("pr-9", className)}
        />
        <button
          type="button"
          disabled={disabled}
          onClick={() => setShowPassword((prev) => !prev)}
          className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded-xs p-0.5 text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50"
          aria-label={t(showPassword ? "common.hidePassword" : "common.showPassword")}
          title={t(showPassword ? "common.hidePassword" : "common.showPassword")}
          tabIndex={-1}
        >
          {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
        </button>
      </div>
    )
  }
)
