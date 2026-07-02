import { describe, expect, it } from "vitest"

import type { SystemSettings } from "@/types"

import {
  buildAdminSettingsPayload,
  createAdminSettingsForm,
  mergeAdminSettingsFormScope,
  type AdminSettingsFormState,
} from "./admin-settings-form"

function settings(overrides: Partial<SystemSettings> = {}): SystemSettings {
  return {
    allow_image_upload: true,
    audit_enabled: true,
    backup_encrypt_enabled: false,
    backup_encryption_password_configured: false,
    backup_include_assets: false,
    backup_last_error: "",
    backup_last_run_at: "",
    backup_last_status: "",
    backup_local_dir: "",
    backup_retention_count: 7,
    backup_schedule_enabled: false,
    backup_time_of_day: "03:00",
    currencyapi_key_configured: false,
    email_domain_whitelist: "",
    exchange_rate_source: "auto",
    icon_proxy_domain_whitelist: "",
    icon_proxy_enabled: true,
    max_icon_file_size: 65536,
    mcp_enabled: false,
    oidc_audience: "",
    oidc_authorization_endpoint: "",
    oidc_auto_create_user: false,
    oidc_client_id: "",
    oidc_client_secret_configured: false,
    oidc_enabled: false,
    oidc_extra_auth_params: "",
    oidc_issuer_url: "",
    oidc_provider_name: "OIDC",
    oidc_redirect_url: "",
    oidc_resource: "",
    oidc_scopes: "openid profile email",
    oidc_token_endpoint: "",
    oidc_userinfo_endpoint: "",
    registration_email_verification_enabled: false,
    registration_enabled: false,
    site_name: "Subdux",
    site_url: "",
    smtp_auth_method: "auto",
    smtp_enabled: false,
    smtp_encryption: "starttls",
    smtp_from_email: "",
    smtp_from_name: "",
    smtp_helo_name: "",
    smtp_host: "",
    smtp_password_configured: false,
    smtp_port: 587,
    smtp_rate_limit_seconds: 0,
    smtp_skip_tls_verify: false,
    smtp_timeout_seconds: 10,
    smtp_username: "",
    ssrf_allow_private_ip: false,
    ssrf_domain_filter_list: "",
    ssrf_domain_filter_mode: "blacklist",
    ssrf_filter_resolved_ips: true,
    ssrf_ip_filter_list: "",
    ssrf_ip_filter_mode: "blacklist",
    ssrf_protection_enabled: true,
    system_proxy_enabled: false,
    system_proxy_type: "http",
    system_proxy_url_configured: false,
    ...overrides,
  }
}

function form(overrides: Partial<AdminSettingsFormState> = {}): AdminSettingsFormState {
  return {
    ...createAdminSettingsForm(settings()),
    backupEncryptionPassword: " backup-secret ",
    currencyApiKey: " currency-secret ",
    oidcClientSecret: " oidc-secret ",
    smtpPassword: " smtp-secret ",
    systemProxyUrl: " http://proxy.example ",
    ...overrides,
  }
}

function payloadKeys(scope: Parameters<typeof buildAdminSettingsPayload>[1]) {
  return Object.keys(buildAdminSettingsPayload(form(), scope)).sort()
}

describe("buildAdminSettingsPayload", () => {
  it("builds only general settings for the general save button", () => {
    expect(payloadKeys("general")).toEqual([
      "allow_image_upload",
      "audit_enabled",
      "icon_proxy_domain_whitelist",
      "icon_proxy_enabled",
      "max_icon_file_size",
      "mcp_enabled",
      "site_name",
      "site_url",
      "ssrf_allow_private_ip",
      "ssrf_domain_filter_list",
      "ssrf_domain_filter_mode",
      "ssrf_filter_resolved_ips",
      "ssrf_ip_filter_list",
      "ssrf_ip_filter_mode",
      "ssrf_protection_enabled",
      "system_proxy_enabled",
      "system_proxy_type",
      "system_proxy_url",
    ])
    expect(buildAdminSettingsPayload(form(), "general").system_proxy_url).toBe(
      "http://proxy.example"
    )
  })

  it("builds only SMTP settings for the SMTP save button", () => {
    expect(payloadKeys("smtp")).toEqual([
      "smtp_auth_method",
      "smtp_enabled",
      "smtp_encryption",
      "smtp_from_email",
      "smtp_from_name",
      "smtp_helo_name",
      "smtp_host",
      "smtp_password",
      "smtp_port",
      "smtp_rate_limit_seconds",
      "smtp_skip_tls_verify",
      "smtp_timeout_seconds",
      "smtp_username",
    ])
    expect(buildAdminSettingsPayload(form(), "smtp").smtp_password).toBe("smtp-secret")
  })

  it("builds only auth and OIDC settings for the auth save button", () => {
    expect(payloadKeys("auth")).toEqual([
      "email_domain_whitelist",
      "oidc_audience",
      "oidc_authorization_endpoint",
      "oidc_auto_create_user",
      "oidc_client_id",
      "oidc_client_secret",
      "oidc_enabled",
      "oidc_extra_auth_params",
      "oidc_issuer_url",
      "oidc_provider_name",
      "oidc_redirect_url",
      "oidc_resource",
      "oidc_scopes",
      "oidc_token_endpoint",
      "oidc_userinfo_endpoint",
      "registration_email_verification_enabled",
      "registration_enabled",
    ])
    expect(buildAdminSettingsPayload(form(), "auth").oidc_client_secret).toBe("oidc-secret")
  })

  it("builds only exchange-rate settings for the exchange-rate save button", () => {
    expect(payloadKeys("exchange-rates")).toEqual(["currencyapi_key", "exchange_rate_source"])
    expect(buildAdminSettingsPayload(form(), "exchange-rates").currencyapi_key).toBe(
      "currency-secret"
    )
  })

  it("builds only backup settings for the reauth-protected backup save button", () => {
    expect(payloadKeys("backup")).toEqual([
      "backup_encrypt_enabled",
      "backup_encryption_password",
      "backup_include_assets",
      "backup_local_dir",
      "backup_retention_count",
      "backup_schedule_enabled",
      "backup_time_of_day",
    ])
    expect(buildAdminSettingsPayload(form(), "backup").backup_encryption_password).toBe(
      "backup-secret"
    )
  })

  it("omits secret values when the secret input is blank", () => {
    expect(
      buildAdminSettingsPayload(form({ systemProxyUrl: "  " }), "general")
    ).not.toHaveProperty("system_proxy_url")
    expect(buildAdminSettingsPayload(form({ smtpPassword: "  " }), "smtp")).not.toHaveProperty(
      "smtp_password"
    )
    expect(buildAdminSettingsPayload(form({ oidcClientSecret: "  " }), "auth")).not.toHaveProperty(
      "oidc_client_secret"
    )
    expect(
      buildAdminSettingsPayload(form({ currencyApiKey: "  " }), "exchange-rates")
    ).not.toHaveProperty("currencyapi_key")
    expect(
      buildAdminSettingsPayload(form({ backupEncryptionPassword: "  " }), "backup")
    ).not.toHaveProperty("backup_encryption_password")
  })
})

describe("mergeAdminSettingsFormScope", () => {
  it("merges normalized settings only for the saved scope", () => {
    const current = form({
      backupTimeOfDay: "09:30",
      siteName: "draft site",
      smtpHost: "draft.smtp.example",
    })
    const fresh = settings({
      backup_time_of_day: "03:00",
      site_name: "server site",
      smtp_host: "server.smtp.example",
    })

    const merged = mergeAdminSettingsFormScope(current, fresh, "smtp")

    expect(merged.smtpHost).toBe("server.smtp.example")
    expect(merged.backupTimeOfDay).toBe("09:30")
    expect(merged.siteName).toBe("draft site")
  })

  it("clears only the saved scope's secret draft after a successful save", () => {
    const current = form({
      backupEncryptionPassword: "backup draft",
      backupEncryptionPasswordConfigured: false,
      smtpPassword: "smtp draft",
    })
    const fresh = settings({ backup_encryption_password_configured: true })

    const merged = mergeAdminSettingsFormScope(current, fresh, "backup")

    expect(merged.backupEncryptionPassword).toBe("")
    expect(merged.backupEncryptionPasswordConfigured).toBe(true)
    expect(merged.smtpPassword).toBe("smtp draft")
  })
})
