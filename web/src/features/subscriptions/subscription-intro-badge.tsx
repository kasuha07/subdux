import { Badge } from "@/components/ui/badge"
import { useSubscriptionIntroText } from "@/features/subscriptions/hooks/use-subscription-intro-text"
import { cn } from "@/lib/utils"
import type { Subscription } from "@/types"

const introBadgeStyles = {
  trial: "bg-teal-500/10 text-teal-700 border-teal-200 dark:text-teal-300 dark:border-teal-800",
  intro: "bg-rose-500/10 text-rose-700 border-rose-200 dark:text-rose-300 dark:border-rose-800",
} as const

export default function SubscriptionIntroBadge({
  className,
  subscription,
}: {
  className?: string
  subscription: Subscription
}) {
  const text = useSubscriptionIntroText(subscription)
  if (!text) {
    return null
  }

  return (
    <Badge
      variant="outline"
      className={cn("max-w-[12rem] truncate", introBadgeStyles[text.kind], className)}
      title={text.after}
    >
      {text.label}
    </Badge>
  )
}
