import { useEffect } from "react"

// Landing route for the OIDC step-up ("reauth") popup. The OIDC callback
// redirects the popup here with ?oidc_action=reauth after setting the
// reauth-scoped session cookie. This route is intentionally operation-agnostic
// and guard-free: the popup's only job is to notify its opener (the page that
// started the step-up, which is still open in the background and owns the
// reauth dialog) so it can mint its ticket, then close. It must NOT sit behind
// ProtectedRoute/AdminRoute — those redirect before this can run, which would
// strand the opener waiting for a signal that never arrives.
export default function OIDCReauthCallback() {
  useEffect(() => {
    if (window.opener && window.opener !== window) {
      window.opener.postMessage({ type: "oidc-reauth" }, window.location.origin)
      window.close()
    }
    // Opened directly (no opener, e.g. a stale bookmark): nothing to signal.
    // Leave the minimal loading text in place; there is no meaningful UI here.
  }, [])

  return (
    <div className="flex min-h-screen items-center justify-center px-4 text-sm text-muted-foreground">
      Loading...
    </div>
  )
}
