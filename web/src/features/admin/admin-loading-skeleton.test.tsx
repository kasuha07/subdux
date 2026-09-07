import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it } from "vitest"

import AdminLoadingSkeleton, {
  AdminAuditTabSkeleton,
  AdminBackupTabSkeleton,
  AdminFormTabSkeleton,
  AdminTabSkeleton,
  AdminTasksListSkeleton,
  AdminTasksTabSkeleton,
  AdminUsersTableSkeleton,
} from "./admin-loading-skeleton"

describe("AdminLoadingSkeleton", () => {
  it("renders with root accessibility role, page loading animation, 8 tabs skeleton and shimmer table", () => {
    const markup = renderToStaticMarkup(<AdminLoadingSkeleton />)

    expect(markup).toContain('role="status"')
    expect(markup).toContain('aria-label="Loading admin console"')
    expect(markup).toContain("page-loading-enter")
    expect(markup).toContain("skeleton-shimmer")
    expect(markup).toContain("subscription-card-enter")
    expect(markup).toContain("--card-delay")
  })

  it("renders AdminUsersTableSkeleton with header and staggered rows", () => {
    const markup = renderToStaticMarkup(<AdminUsersTableSkeleton />)

    expect(markup).toContain("skeleton-shimmer")
    expect(markup).toContain("subscription-card-enter")
    expect(markup).toContain("--card-delay")
  })

  it("renders AdminTasksTabSkeleton and AdminTasksListSkeleton", () => {
    const listMarkup = renderToStaticMarkup(<AdminTasksListSkeleton />)
    expect(listMarkup).toContain("skeleton-shimmer")
    expect(listMarkup).toContain("subscription-card-enter")

    const tabMarkup = renderToStaticMarkup(<AdminTasksTabSkeleton />)
    expect(tabMarkup).toContain("skeleton-shimmer")
  })

  it("renders AdminAuditTabSkeleton", () => {
    const markup = renderToStaticMarkup(<AdminAuditTabSkeleton />)
    expect(markup).toContain("skeleton-shimmer")
    expect(markup).toContain("subscription-card-enter")
  })

  it("renders AdminBackupTabSkeleton", () => {
    const markup = renderToStaticMarkup(<AdminBackupTabSkeleton />)
    expect(markup).toContain("skeleton-shimmer")
  })

  it("renders AdminFormTabSkeleton", () => {
    const markup = renderToStaticMarkup(<AdminFormTabSkeleton />)
    expect(markup).toContain("skeleton-shimmer")
  })

  it("dispatches AdminTabSkeleton correctly for all tab kinds", () => {
    expect(renderToStaticMarkup(<AdminTabSkeleton tab="users" />)).toContain("rounded-full")
    expect(renderToStaticMarkup(<AdminTabSkeleton tab="background-tasks" />)).toContain("subscription-card-enter")
    expect(renderToStaticMarkup(<AdminTabSkeleton tab="audit" />)).toContain("subscription-card-enter")
    expect(renderToStaticMarkup(<AdminTabSkeleton tab="backup" />)).toContain("skeleton-shimmer")
    expect(renderToStaticMarkup(<AdminTabSkeleton tab="settings" />)).toContain("skeleton-shimmer")
    expect(renderToStaticMarkup(<AdminTabSkeleton tab="smtp" />)).toContain("skeleton-shimmer")
    expect(renderToStaticMarkup(<AdminTabSkeleton tab="auth" />)).toContain("skeleton-shimmer")
    expect(renderToStaticMarkup(<AdminTabSkeleton tab="exchange-rates" />)).toContain("skeleton-shimmer")
  })
})
