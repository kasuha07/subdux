import { Skeleton } from "@/components/ui/skeleton"

export function AdminUsersTableSkeleton() {
  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Skeleton className="h-9 w-28 rounded-md" />
      </div>

      <div className="rounded-md border bg-card overflow-hidden skeleton-shimmer">
        <div className="flex items-center gap-4 border-b bg-muted/40 px-4 py-3">
          <Skeleton className="h-4 w-28" />
          <Skeleton className="h-4 w-16" />
          <Skeleton className="h-4 w-16" />
          <Skeleton className="h-4 w-16" />
          <Skeleton className="h-4 w-24" />
          <Skeleton className="ml-auto size-6 rounded" />
        </div>
        {Array.from({ length: 5 }).map((_, i) => (
          <div
            key={i}
            className="flex items-center gap-4 border-b px-4 py-3 last:border-0 subscription-card-enter"
            style={{ "--card-delay": `${i * 45}ms` } as React.CSSProperties}
          >
            <div className="flex items-center gap-3 min-w-[180px] flex-1 sm:flex-initial">
              <Skeleton className="size-8 rounded-full shrink-0" />
              <div className="space-y-1.5">
                <Skeleton className="h-3.5 w-24" />
                <Skeleton className="h-3 w-36" />
              </div>
            </div>
            <Skeleton className="h-5 w-14 rounded-full" />
            <Skeleton className="h-5 w-16 rounded-full" />
            <div className="flex gap-1.5">
              <Skeleton className="size-4 rounded" />
              <Skeleton className="size-4 rounded" />
            </div>
            <Skeleton className="h-3.5 w-24" />
            <Skeleton className="ml-auto size-7 rounded-md" />
          </div>
        ))}
      </div>
    </div>
  )
}

export function AdminFormTabSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-3.5 w-64" />
        </div>
        <Skeleton className="h-9 w-28 rounded-md" />
      </div>

      <div className="rounded-lg border bg-card p-4 sm:p-6 space-y-6 skeleton-shimmer">
        <div className="space-y-1 border-b pb-4">
          <Skeleton className="h-4 w-36" />
          <Skeleton className="h-3 w-56" />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Skeleton className="h-3.5 w-20" />
            <Skeleton className="h-9 w-full rounded-md" />
          </div>
          <div className="space-y-2">
            <Skeleton className="h-3.5 w-24" />
            <Skeleton className="h-9 w-full rounded-md" />
          </div>
        </div>
        <div className="space-y-4 pt-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center justify-between gap-4 py-2 border-t first:border-0"
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
    </div>
  )
}

export function AdminTasksListSkeleton() {
  return (
    <div className="space-y-3">
      {Array.from({ length: 3 }).map((_, i) => (
        <div
          key={i}
          className="rounded-lg border bg-card p-4 space-y-4 skeleton-shimmer subscription-card-enter"
          style={{ "--card-delay": `${i * 50}ms` } as React.CSSProperties}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Skeleton className="size-4 rounded" />
              <Skeleton className="h-4 w-32" />
            </div>
            <Skeleton className="h-5 w-16 rounded-full" />
          </div>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            {Array.from({ length: 4 }).map((_, j) => (
              <div key={j} className="space-y-1">
                <Skeleton className="h-3 w-16" />
                <Skeleton className="h-4 w-24" />
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

export function AdminTasksTabSkeleton() {
  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between">
        <div className="space-y-1">
          <Skeleton className="h-5 w-48" />
          <Skeleton className="h-3.5 w-72" />
        </div>
        <Skeleton className="h-8 w-20 rounded-md" />
      </div>
      <AdminTasksListSkeleton />
    </div>
  )
}

export function AdminAuditTabSkeleton() {
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
        {Array.from({ length: 5 }).map((_, i) => (
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

export function AdminBackupTabSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div className="space-y-1">
          <Skeleton className="h-5 w-36" />
          <Skeleton className="h-3.5 w-64" />
        </div>
        <div className="flex gap-2">
          <Skeleton className="h-8 w-28 rounded-md" />
          <Skeleton className="h-8 w-24 rounded-md" />
        </div>
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="rounded-lg border bg-card p-4 space-y-3 skeleton-shimmer">
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-3 w-48" />
          <Skeleton className="h-9 w-full rounded-md mt-2" />
        </div>
        <div className="rounded-lg border bg-card p-4 space-y-3 skeleton-shimmer">
          <Skeleton className="h-4 w-28" />
          <Skeleton className="h-3 w-44" />
          <Skeleton className="h-9 w-full rounded-md mt-2" />
        </div>
      </div>
      <div className="rounded-md border bg-card p-4 space-y-3 skeleton-shimmer">
        <Skeleton className="h-4 w-36" />
        <Skeleton className="h-20 w-full rounded-md" />
      </div>
    </div>
  )
}

export function AdminTabSkeleton({
  tab,
}: {
  tab: "users" | "settings" | "smtp" | "auth" | "exchange-rates" | "background-tasks" | "audit" | "backup"
}) {
  switch (tab) {
    case "users":
      return <AdminUsersTableSkeleton />
    case "background-tasks":
      return <AdminTasksTabSkeleton />
    case "audit":
      return <AdminAuditTabSkeleton />
    case "backup":
      return <AdminBackupTabSkeleton />
    case "settings":
    case "smtp":
    case "auth":
    case "exchange-rates":
    default:
      return <AdminFormTabSkeleton />
  }
}

const TAB_WIDTHS = [56, 68, 52, 92, 96, 110, 56, 64]

export default function AdminLoadingSkeleton() {
  return (
    <div
      role="status"
      aria-label="Loading admin console"
      className="page-loading-enter space-y-6"
    >
      <div className="w-full overflow-x-auto pb-1">
        <div className="inline-flex h-9 w-max min-w-max items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground gap-1">
          {TAB_WIDTHS.map((width, i) => (
            <div
              key={i}
              className={`inline-flex items-center gap-2 rounded-md px-3 py-1 text-sm ${
                i === 0
                  ? "bg-background text-foreground shadow-xs"
                  : "text-muted-foreground/60"
              }`}
            >
              <Skeleton className={`size-4 rounded ${i > 0 ? "opacity-70" : ""}`} />
              <Skeleton
                className={`h-4 rounded ${i > 0 ? "opacity-70" : ""}`}
                style={{ width: `${width}px` }}
              />
            </div>
          ))}
        </div>
      </div>

      <AdminUsersTableSkeleton />
    </div>
  )
}
