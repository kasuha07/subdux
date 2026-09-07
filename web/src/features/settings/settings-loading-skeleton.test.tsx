import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it } from "vitest"

import {
  SettingsAboutTabSkeleton,
  SettingsAccountTabSkeleton,
  SettingsAPIKeyTabSkeleton,
  SettingsAuditTabSkeleton,
  SettingsGeneralTabSkeleton,
  SettingsNotificationTabSkeleton,
  SettingsPaymentTabSkeleton,
  SettingsTabSkeleton,
} from "./settings-loading-skeleton"

describe("SettingsLoadingSkeletons", () => {
  it("renders SettingsGeneralTabSkeleton with theme, colors, and display options", () => {
    const markup = renderToStaticMarkup(<SettingsGeneralTabSkeleton />)
    expect(markup).toContain("skeleton-shimmer")
    expect(markup).toContain("rounded-full")
  })

  it("renders SettingsPaymentTabSkeleton with currency, category and payment skeletons", () => {
    const markup = renderToStaticMarkup(<SettingsPaymentTabSkeleton />)
    expect(markup).toContain("skeleton-shimmer")
    expect(markup).toContain("subscription-card-enter")
    expect(markup).toContain("--card-delay")
  })

  it("renders SettingsNotificationTabSkeleton with policy, channels, and logs skeletons", () => {
    const markup = renderToStaticMarkup(<SettingsNotificationTabSkeleton />)
    expect(markup).toContain("skeleton-shimmer")
    expect(markup).toContain("subscription-card-enter")
  })

  it("renders SettingsAccountTabSkeleton with info, password, passkeys, and transfer skeletons", () => {
    const markup = renderToStaticMarkup(<SettingsAccountTabSkeleton />)
    expect(markup).toContain("skeleton-shimmer")
  })

  it("renders SettingsAPIKeyTabSkeleton with create form and keys list skeletons", () => {
    const markup = renderToStaticMarkup(<SettingsAPIKeyTabSkeleton />)
    expect(markup).toContain("skeleton-shimmer")
    expect(markup).toContain("subscription-card-enter")
  })

  it("renders SettingsAuditTabSkeleton with audit event item skeletons", () => {
    const markup = renderToStaticMarkup(<SettingsAuditTabSkeleton />)
    expect(markup).toContain("skeleton-shimmer")
    expect(markup).toContain("subscription-card-enter")
  })

  it("renders SettingsAboutTabSkeleton with logo and info card skeletons", () => {
    const markup = renderToStaticMarkup(<SettingsAboutTabSkeleton />)
    expect(markup).toContain("skeleton-shimmer")
  })

  it("dispatches SettingsTabSkeleton correctly across all settings tabs", () => {
    expect(renderToStaticMarkup(<SettingsTabSkeleton tab="general" />)).toContain("skeleton-shimmer")
    expect(renderToStaticMarkup(<SettingsTabSkeleton tab="payment" />)).toContain("skeleton-shimmer")
    expect(renderToStaticMarkup(<SettingsTabSkeleton tab="notification" />)).toContain("skeleton-shimmer")
    expect(renderToStaticMarkup(<SettingsTabSkeleton tab="account" />)).toContain("skeleton-shimmer")
    expect(renderToStaticMarkup(<SettingsTabSkeleton tab="apikey" />)).toContain("skeleton-shimmer")
    expect(renderToStaticMarkup(<SettingsTabSkeleton tab="audit" />)).toContain("skeleton-shimmer")
    expect(renderToStaticMarkup(<SettingsTabSkeleton tab="about" />)).toContain("skeleton-shimmer")
  })
})
