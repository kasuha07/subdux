import { useState, useEffect, type ChangeEvent } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import {
  ArrowLeft,
  Users,
  Settings,
  BarChart3,
  Database,
  MoreHorizontal,
  Download,
  AlertTriangle,
  DollarSign,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"

import { api } from "@/lib/api"
import type { User, AdminStats, SystemSettings } from "@/types"

export default function AdminPage() {
  const { t } = useTranslation()

  const [users, setUsers] = useState<User[]>([])
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [siteName, setSiteName] = useState("")
  const [registrationEnabled, setRegistrationEnabled] = useState(true)
  const [restoreFile, setRestoreFile] = useState<File | null>(null)
  const [restoreConfirmOpen, setRestoreConfirmOpen] = useState(false)

  useEffect(() => {
    Promise.all([
      api.get<User[]>("/admin/users"),
      api.get<SystemSettings>("/admin/settings"),
      api.get<AdminStats>("/admin/stats"),
    ])
      .then(([usersData, settingsData, statsData]) => {
        setUsers(usersData || [])
        setSiteName(settingsData?.site_name || "Subdux")
        setRegistrationEnabled(settingsData?.registration_enabled ?? true)
        setStats(statsData)
      })
      .catch(() => void 0)
      .finally(() => setLoading(false))
  }, [])

  async function handleToggleRole(user: User) {
    const newRole = user.role === "admin" ? "user" : "admin"
    try {
      await api.put(`/admin/users/${user.id}/role`, { role: newRole })
      setUsers(prev => prev.map(u => (u.id === user.id ? { ...u, role: newRole } : u)))
    } catch {
      void 0
    }
  }

  async function handleToggleStatus(user: User) {
    const newStatus = user.status === "active" ? "disabled" : "active"
    try {
      await api.put(`/admin/users/${user.id}/status`, { status: newStatus })
      setUsers(prev => prev.map(u => (u.id === user.id ? { ...u, status: newStatus } : u)))
    } catch {
      void 0
    }
  }

  async function handleDeleteUser(id: number) {
    if (!confirm(t("admin.users.deleteConfirm"))) return
    try {
      await api.delete(`/admin/users/${id}`)
      setUsers(prev => prev.filter(u => u.id !== id))
    } catch {
      void 0
    }
  }

  async function handleSaveSettings() {
    try {
      await api.put("/admin/settings", {
        registration_enabled: registrationEnabled,
        site_name: siteName,
      })
      const fresh = await api.get<SystemSettings>("/admin/settings")
      setSiteName(fresh.site_name)
      setRegistrationEnabled(fresh.registration_enabled)
    } catch {
      void 0
    }
  }

  async function handleDownloadBackup() {
    try {
      const token = localStorage.getItem("token")
      const res = await fetch("/api/admin/backup", {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!res.ok) throw new Error()

      const blob = await res.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = `subdux-backup-${new Date().toISOString().split("T")[0]}.db`
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)
    } catch {
      void 0
    }
  }

  async function handleRestore() {
    if (!restoreFile) return

    const formData = new FormData()
    formData.append("backup", restoreFile)

    try {
      const token = localStorage.getItem("token")
      const res = await fetch("/api/admin/restore", {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        body: formData,
      })
      if (!res.ok) throw new Error()

      setRestoreConfirmOpen(false)
      alert(t("admin.backup.restoreSuccess"))
    } catch {
      void 0
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-6xl items-center px-4 gap-3">
          <Button variant="ghost" size="icon-sm" asChild>
            <Link to="/">
              <ArrowLeft className="size-4" />
            </Link>
          </Button>
          <h1 className="text-lg font-bold tracking-tight">{t("admin.title")}</h1>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-6">
        {loading ? (
          <div className="space-y-6">
            <div className="flex gap-2">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-9 w-28 rounded-md" />
              ))}
            </div>

            <div className="rounded-md border">
              <div className="border-b px-4 py-3 flex gap-8">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-4 w-16" />
                <Skeleton className="h-4 w-16" />
                <Skeleton className="h-4 w-20" />
              </div>
              {Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="flex items-center gap-8 border-b last:border-0 px-4 py-3">
                  <Skeleton className="h-4 w-40" />
                  <Skeleton className="h-5 w-14 rounded-full" />
                  <Skeleton className="h-5 w-14 rounded-full" />
                  <Skeleton className="h-4 w-24" />
                  <Skeleton className="ml-auto size-6 rounded" />
                </div>
              ))}
            </div>
          </div>
        ) : (
        <Tabs defaultValue="users" className="space-y-6">
          <TabsList>
            <TabsTrigger value="users" className="gap-2">
              <Users className="size-4" />
              {t("admin.tabs.users")}
            </TabsTrigger>
            <TabsTrigger value="settings" className="gap-2">
              <Settings className="size-4" />
              {t("admin.tabs.settings")}
            </TabsTrigger>
            <TabsTrigger value="stats" className="gap-2">
              <BarChart3 className="size-4" />
              {t("admin.tabs.statistics")}
            </TabsTrigger>
            <TabsTrigger value="backup" className="gap-2">
              <Database className="size-4" />
              {t("admin.tabs.backup")}
            </TabsTrigger>
          </TabsList>

          <TabsContent value="users">
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("admin.users.email")}</TableHead>
                    <TableHead>{t("admin.users.role")}</TableHead>
                    <TableHead>{t("admin.users.status")}</TableHead>
                    <TableHead>{t("admin.users.created")}</TableHead>
                    <TableHead className="w-[50px]"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {users.map((user) => (
                    <TableRow key={user.id}>
                      <TableCell>{user.email}</TableCell>
                      <TableCell>
                        <Badge variant={user.role === "admin" ? "default" : "secondary"}>
                          {user.role}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={user.status === "active" ? "outline" : "destructive"}>
                          {user.status}
                        </Badge>
                      </TableCell>
                      <TableCell>{new Date(user.created_at).toLocaleDateString()}</TableCell>
                      <TableCell>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon-sm">
                              <MoreHorizontal className="size-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => handleToggleRole(user)}>
                              {user.role === "admin"
                                ? t("admin.users.makeUser")
                                : t("admin.users.makeAdmin")}
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => handleToggleStatus(user)}>
                              {user.status === "active"
                                ? t("admin.users.disable")
                                : t("admin.users.enable")}
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                              className="text-destructive focus:text-destructive"
                              onClick={() => handleDeleteUser(user.id)}
                            >
                              {t("admin.users.delete")}
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </TabsContent>

          <TabsContent value="settings">
            <Card>
              <CardContent className="p-6 space-y-6">
                <div className="space-y-2">
                  <Label htmlFor="site-name">{t("admin.settings.siteName")}</Label>
                  <Input
                    id="site-name"
                    value={siteName}
                    onChange={(e) => setSiteName(e.target.value)}
                    placeholder="Subdux"
                  />
                </div>

                <Separator />

                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label htmlFor="registration">{t("admin.settings.registrationEnabled")}</Label>
                    <p className="text-sm text-muted-foreground">
                      {t("admin.settings.registrationDescription")}
                    </p>
                  </div>
                  <Switch
                    id="registration"
                    checked={registrationEnabled}
                    onCheckedChange={setRegistrationEnabled}
                  />
                </div>

                <Separator />

                <Button onClick={handleSaveSettings}>
                  {t("admin.settings.save")}
                </Button>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="stats">
            <div className="grid gap-4 md:grid-cols-3">
              <Card>
                <CardContent className="p-6">
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <Users className="size-4" />
                    <span className="text-xs font-medium uppercase tracking-wider">
                      {t("admin.stats.totalUsers")}
                    </span>
                  </div>
                  <p className="mt-2 text-3xl font-bold tabular-nums">
                    {stats?.total_users ?? 0}
                  </p>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-6">
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <BarChart3 className="size-4" />
                    <span className="text-xs font-medium uppercase tracking-wider">
                      {t("admin.stats.totalSubscriptions")}
                    </span>
                  </div>
                  <p className="mt-2 text-3xl font-bold tabular-nums">
                    {stats?.total_subscriptions ?? 0}
                  </p>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-6">
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <DollarSign className="size-4" />
                    <span className="text-xs font-medium uppercase tracking-wider">
                      {t("admin.stats.monthlySpend")}
                    </span>
                  </div>
                  <p className="mt-2 text-3xl font-bold tabular-nums">
                    ${(stats?.total_monthly_spend ?? 0).toFixed(2)}
                  </p>
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          <TabsContent value="backup" className="space-y-4">
            <Card>
              <CardContent className="p-6">
                <h3 className="text-sm font-medium">{t("admin.backup.download")}</h3>
                <p className="text-sm text-muted-foreground mt-0.5">
                  {t("admin.backup.downloadDescription")}
                </p>
                <Button variant="outline" className="mt-4" onClick={handleDownloadBackup}>
                  <Download className="size-4" />
                  {t("admin.backup.downloadButton")}
                </Button>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="p-6 space-y-4">
                <div>
                  <h3 className="text-sm font-medium">{t("admin.backup.restore")}</h3>
                  <p className="text-sm text-muted-foreground mt-0.5">
                    {t("admin.backup.restoreDescription")}
                  </p>
                </div>
                <Input
                  type="file"
                  accept=".db"
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    setRestoreFile(e.target.files?.[0] ?? null)
                  }
                />
                <Button
                  variant="destructive"
                  disabled={!restoreFile}
                  onClick={() => setRestoreConfirmOpen(true)}
                >
                  {t("admin.backup.restoreButton")}
                </Button>

                {restoreConfirmOpen && (
                  <div className="rounded-md border border-destructive bg-destructive/10 p-4">
                    <div className="flex items-center gap-2 text-destructive font-medium mb-2">
                      <AlertTriangle className="size-4" />
                      {t("admin.backup.restoreConfirm")}
                    </div>
                    <div className="flex gap-2 mt-3">
                      <Button size="sm" variant="destructive" onClick={handleRestore}>
                        {t("admin.backup.confirm")}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setRestoreConfirmOpen(false)}
                      >
                        {t("admin.backup.cancel")}
                      </Button>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
        )}
      </main>
    </div>
  )
}
