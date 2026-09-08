import { Skeleton } from "@/components/ui/skeleton"
import { TabsContent } from "@/components/ui/tabs"
import { useTranslation } from "react-i18next"

export function SettingsGeneralTabSkeleton() {
  return (
    <div className="space-y-6">
      {/* Theme section */}
      <div className="space-y-3">
        <div className="space-y-1">
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-3.5 w-64" />
        </div>
        <div className="grid grid-cols-3 gap-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div
              key={i}
              className="rounded-lg border bg-card p-3 space-y-2 skeleton-shimmer"
            >
              <div className="flex items-center gap-2">
                <Skeleton className="size-4 rounded" />
                <Skeleton className="h-4 w-16" />
              </div>
              <Skeleton className="h-10 w-full rounded" />
            </div>
          ))}
        </div>
      </div>

      {/* Color scheme section */}
      <div className="space-y-3 pt-2">
        <div className="space-y-1">
          <Skeleton className="h-5 w-28" />
          <Skeleton className="h-3.5 w-52" />
        </div>
        <div className="flex flex-wrap gap-2.5">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="size-8 rounded-full" />
          ))}
        </div>
      </div>

      {/* Display options section */}
      <div className="space-y-3 pt-2">
        <div className="space-y-1">
          <Skeleton className="h-5 w-36" />
          <Skeleton className="h-3.5 w-56" />
        </div>
        <div className="rounded-lg border bg-card divide-y skeleton-shimmer">
          {Array.from({ length: 4 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center justify-between p-3.5 gap-4"
            >
              <div className="space-y-1">
                <Skeleton className="h-4 w-44" />
                <Skeleton className="h-3 w-64" />
              </div>
              <Skeleton className="h-5 w-9 rounded-full shrink-0" />
            </div>
          ))}
        </div>
      </div>

      {/* Language */}
      <div className="space-y-2 pt-2">
        <Skeleton className="h-4 w-20" />
        <Skeleton className="h-9 w-48 rounded-md" />
      </div>
    </div>
  )
}

export function SettingsPaymentTabSkeleton() {
  return (
    <div className="space-y-6">
      {/* Primary currency */}
      <div className="space-y-3">
        <div className="space-y-1">
          <Skeleton className="h-5 w-36" />
          <Skeleton className="h-3.5 w-64" />
        </div>
        <Skeleton className="h-9 w-56 rounded-md" />
      </div>

      {/* Currency list */}
      <div className="space-y-3 pt-2">
        <div className="space-y-1">
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-3.5 w-56" />
        </div>
        <div className="flex gap-2">
          <Skeleton className="h-9 w-32 rounded-md" />
          <Skeleton className="h-9 w-24 rounded-md" />
          <Skeleton className="h-9 w-20 rounded-md" />
        </div>
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center gap-3 rounded-md border bg-card px-3 py-2.5 skeleton-shimmer subscription-card-enter"
              style={{ "--card-delay": `${i * 40}ms` } as React.CSSProperties}
            >
              <Skeleton className="size-4 rounded shrink-0" />
              <Skeleton className="h-4 w-12" />
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-4 w-24" />
              <Skeleton className="ml-auto size-7 rounded" />
            </div>
          ))}
        </div>
      </div>

      {/* Categories & Payment methods */}
      <div className="grid gap-6 pt-2 md:grid-cols-2">
        <div className="space-y-3 rounded-lg border bg-card p-4 skeleton-shimmer">
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-3.5 w-48" />
          <div className="flex gap-2">
            <Skeleton className="h-9 flex-1 rounded-md" />
            <Skeleton className="h-9 w-16 rounded-md" />
          </div>
          <div className="space-y-2 pt-1">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex items-center justify-between border-b pb-2 last:border-0">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="size-6 rounded" />
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-3 rounded-lg border bg-card p-4 skeleton-shimmer">
          <Skeleton className="h-5 w-36" />
          <Skeleton className="h-3.5 w-48" />
          <div className="flex gap-2">
            <Skeleton className="h-9 flex-1 rounded-md" />
            <Skeleton className="h-9 w-16 rounded-md" />
          </div>
          <div className="space-y-2 pt-1">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex items-center justify-between border-b pb-2 last:border-0">
                <div className="flex items-center gap-2">
                  <Skeleton className="size-5 rounded" />
                  <Skeleton className="h-4 w-20" />
                </div>
                <Skeleton className="size-6 rounded" />
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

export function SettingsNotificationTabSkeleton() {
  return (
    <div className="space-y-6">
      {/* Policy card */}
      <div className="rounded-lg border bg-card p-4 space-y-4 skeleton-shimmer">
        <div className="space-y-1">
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-3.5 w-64" />
        </div>
        <div className="space-y-3 pt-1">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="flex items-center justify-between py-1.5 border-b last:border-0">
              <div className="space-y-1">
                <Skeleton className="h-4 w-48" />
                <Skeleton className="h-3 w-56" />
              </div>
              <Skeleton className="h-5 w-9 rounded-full shrink-0" />
            </div>
          ))}
        </div>
      </div>

      {/* Channels section */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <Skeleton className="h-5 w-44" />
            <Skeleton className="h-3.5 w-56" />
          </div>
          <Skeleton className="h-8 w-28 rounded-md" />
        </div>
        <div className="space-y-2">
          {Array.from({ length: 2 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center justify-between rounded-md border bg-card p-3 skeleton-shimmer subscription-card-enter"
              style={{ "--card-delay": `${i * 50}ms` } as React.CSSProperties}
            >
              <div className="space-y-1">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-3 w-48" />
              </div>
              <div className="flex items-center gap-2">
                <Skeleton className="h-5 w-9 rounded-full" />
                <Skeleton className="size-7 rounded" />
                <Skeleton className="size-7 rounded" />
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Logs section */}
      <div className="space-y-3 pt-2">
        <Skeleton className="h-5 w-36" />
        <div className="rounded-md border bg-card overflow-hidden skeleton-shimmer">
          <div className="flex items-center gap-4 border-b bg-muted/40 px-4 py-2.5">
            <Skeleton className="h-3.5 w-20" />
            <Skeleton className="h-3.5 w-24" />
            <Skeleton className="h-3.5 w-16" />
            <Skeleton className="ml-auto h-3.5 w-20" />
          </div>
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="flex items-center gap-4 border-b px-4 py-2.5 last:border-0">
              <Skeleton className="h-3.5 w-20" />
              <Skeleton className="h-3.5 w-24" />
              <Skeleton className="h-5 w-14 rounded-full" />
              <Skeleton className="ml-auto h-3.5 w-20" />
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export function SettingsAccountTabSkeleton() {
  return (
    <div className="space-y-6">
      {/* Account info */}
      <div className="space-y-4">
        <div className="space-y-1">
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-3.5 w-60" />
        </div>
        <div className="space-y-3 rounded-lg border bg-card p-4 skeleton-shimmer">
          <div className="space-y-1">
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-4 w-28" />
          </div>
          <div className="space-y-1">
            <Skeleton className="h-3 w-12" />
            <div className="flex items-center gap-3">
              <Skeleton className="h-4 w-44" />
              <Skeleton className="h-7 w-24 rounded-md" />
            </div>
          </div>
        </div>
      </div>

      {/* Password change */}
      <div className="space-y-3 rounded-lg border bg-card p-4 skeleton-shimmer">
        <Skeleton className="h-4 w-32" />
        <div className="grid gap-3 sm:grid-cols-3">
          <Skeleton className="h-9 w-full rounded-md" />
          <Skeleton className="h-9 w-full rounded-md" />
          <Skeleton className="h-9 w-full rounded-md" />
        </div>
        <div className="flex justify-end">
          <Skeleton className="h-8 w-28 rounded-md" />
        </div>
      </div>

      {/* Passkeys & OIDC */}
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="rounded-lg border bg-card p-4 space-y-3 skeleton-shimmer">
          <div className="flex items-center justify-between">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-7 w-24 rounded-md" />
          </div>
          <Skeleton className="h-12 w-full rounded-md" />
        </div>
        <div className="rounded-lg border bg-card p-4 space-y-3 skeleton-shimmer">
          <Skeleton className="h-4 w-28" />
          <Skeleton className="h-12 w-full rounded-md" />
        </div>
      </div>

      {/* Export / Import */}
      <div className="rounded-lg border bg-card p-4 space-y-3 skeleton-shimmer">
        <Skeleton className="h-4 w-36" />
        <div className="flex flex-wrap gap-2">
          <Skeleton className="h-8 w-24 rounded-md" />
          <Skeleton className="h-8 w-24 rounded-md" />
        </div>
      </div>
    </div>
  )
}

export function SettingsAPIKeyTabSkeleton() {
  return (
    <div className="space-y-6">
      {/* Create key section */}
      <div className="rounded-lg border bg-card p-4 space-y-4 skeleton-shimmer">
        <div className="space-y-1">
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-3.5 w-64" />
        </div>
        <div className="grid gap-3 sm:grid-cols-3">
          <Skeleton className="h-9 w-full rounded-md" />
          <Skeleton className="h-9 w-full rounded-md" />
          <Skeleton className="h-9 w-28 rounded-md" />
        </div>
      </div>

      {/* Keys list */}
      <div className="space-y-3">
        <Skeleton className="h-5 w-36" />
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center justify-between rounded-md border bg-card px-3 py-2.5 skeleton-shimmer subscription-card-enter"
              style={{ "--card-delay": `${i * 45}ms` } as React.CSSProperties}
            >
              <div className="space-y-1.5 min-w-0 flex-1">
                <Skeleton className="h-4 w-32" />
                <div className="flex gap-2">
                  <Skeleton className="h-3 w-28" />
                  <Skeleton className="h-4 w-16 rounded-full" />
                </div>
              </div>
              <Skeleton className="size-8 rounded-md" />
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export function SettingsAuditTabSkeleton() {
  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between">
        <div className="space-y-1">
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-3.5 w-60" />
        </div>
        <Skeleton className="h-8 w-20 rounded-md" />
      </div>
      <div className="space-y-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="rounded-md border bg-card px-3 py-2.5 space-y-2 skeleton-shimmer subscription-card-enter"
            style={{ "--card-delay": `${i * 35}ms` } as React.CSSProperties}
          >
            <div className="flex items-center gap-3">
              <Skeleton className="h-4 w-28" />
              <Skeleton className="h-5 w-16 rounded-full" />
              <Skeleton className="h-3 w-32" />
            </div>
            <Skeleton className="h-3 w-48" />
          </div>
        ))}
      </div>
    </div>
  )
}

export function SettingsAboutTabSkeleton() {
  return (
    <div className="space-y-6 max-w-lg mx-auto py-4">
      <div className="text-center space-y-3">
        <Skeleton className="size-16 rounded-2xl mx-auto" />
        <Skeleton className="h-6 w-32 mx-auto" />
        <Skeleton className="h-5 w-20 rounded-full mx-auto" />
      </div>

      <div className="rounded-lg border bg-card p-4 space-y-3 skeleton-shimmer">
        <div className="flex justify-between border-b pb-2.5">
          <Skeleton className="h-3.5 w-20" />
          <Skeleton className="h-3.5 w-28" />
        </div>
        <div className="flex justify-between border-b pb-2.5">
          <Skeleton className="h-3.5 w-24" />
          <Skeleton className="h-3.5 w-32" />
        </div>
        <div className="flex justify-between">
          <Skeleton className="h-3.5 w-20" />
          <Skeleton className="h-3.5 w-24" />
        </div>
      </div>

      <div className="flex justify-center">
        <Skeleton className="h-9 w-36 rounded-md" />
      </div>
    </div>
  )
}

export type SettingsTabType =
  | "general"
  | "payment"
  | "notification"
  | "account"
  | "apikey"
  | "audit"
  | "about"

export function SettingsTabSkeleton({ tab }: { tab: SettingsTabType }) {
  switch (tab) {
    case "payment":
      return <SettingsPaymentTabSkeleton />
    case "notification":
      return <SettingsNotificationTabSkeleton />
    case "account":
      return <SettingsAccountTabSkeleton />
    case "apikey":
      return <SettingsAPIKeyTabSkeleton />
    case "audit":
      return <SettingsAuditTabSkeleton />
    case "about":
      return <SettingsAboutTabSkeleton />
    case "general":
    default:
      return <SettingsGeneralTabSkeleton />
  }
}

export function SettingsTabLoading({ value }: { value: SettingsTabType }) {
  const { t } = useTranslation()
  return (
    <TabsContent
      value={value}
      role="status"
      aria-label={t("common.loading")}
      className="settings-tab-content outline-none"
    >
      <SettingsTabSkeleton tab={value} />
    </TabsContent>
  )
}
