import { useTranslation } from "react-i18next"
import { MoreHorizontal, Plus } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { TabsContent } from "@/components/ui/tabs"
import type { AdminUser } from "@/types"

interface AdminUsersTabProps {
  createDialogOpen: boolean
  newEmail: string
  newPassword: string
  newRole: "user" | "admin"
  newUsername: string
  onCreateDialogOpenChange: (open: boolean) => void
  onCreateUser: () => void | Promise<void>
  onDeleteUser: (id: number) => void | Promise<void>
  onNewEmailChange: (value: string) => void
  onNewPasswordChange: (value: string) => void
  onNewRoleChange: (role: "user" | "admin") => void
  onNewUsernameChange: (value: string) => void
  onToggleRole: (user: AdminUser) => void | Promise<void>
  onToggleStatus: (user: AdminUser) => void | Promise<void>
  users: AdminUser[]
}

export default function AdminUsersTab({
  createDialogOpen,
  newEmail,
  newPassword,
  newRole,
  newUsername,
  onCreateDialogOpenChange,
  onCreateUser,
  onDeleteUser,
  onNewEmailChange,
  onNewPasswordChange,
  onNewRoleChange,
  onNewUsernameChange,
  onToggleRole,
  onToggleStatus,
  users,
}: AdminUsersTabProps) {
  const { t } = useTranslation()

  return (
    <TabsContent value="users">
      <div className="mb-4 flex justify-end">
        <Dialog open={createDialogOpen} onOpenChange={onCreateDialogOpenChange}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 size-4" />
              {t("admin.users.createUser")}
            </Button>
          </DialogTrigger>
          <DialogContent className="flex max-h-[calc(100vh-1.5rem)] max-w-[425px] flex-col gap-0 overflow-hidden p-0 sm:max-h-[85vh]">
            <DialogHeader className="border-b px-5 pt-5 pb-4 sm:px-6">
              <DialogTitle>{t("admin.users.createUser")}</DialogTitle>
              <DialogDescription>{t("admin.users.createUserDescription")}</DialogDescription>
            </DialogHeader>
            <div className="flex min-h-0 flex-1 flex-col">
              <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4 sm:px-6">
                <div className="grid gap-4">
                  <div className="grid gap-2">
                    <Label htmlFor="username">{t("admin.users.username")}</Label>
                    <Input
                      id="username"
                      value={newUsername}
                      onChange={(event) => onNewUsernameChange(event.target.value)}
                      placeholder="johndoe"
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="email">{t("admin.users.email")}</Label>
                    <Input
                      id="email"
                      type="email"
                      value={newEmail}
                      onChange={(event) => onNewEmailChange(event.target.value)}
                      placeholder="john@example.com"
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="password">{t("admin.users.password")}</Label>
                    <Input
                      id="password"
                      type="password"
                      value={newPassword}
                      onChange={(event) => onNewPasswordChange(event.target.value)}
                      placeholder="••••••"
                    />
                    <p className="text-xs text-muted-foreground">
                      {t("admin.users.passwordMinLength")}
                    </p>
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="role">{t("admin.users.role")}</Label>
                    <select
                      id="role"
                      value={newRole}
                      onChange={(event) => onNewRoleChange(event.target.value as "user" | "admin")}
                      className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      <option value="user">{t("admin.users.roleUser")}</option>
                      <option value="admin">{t("admin.users.roleAdmin")}</option>
                    </select>
                  </div>
                </div>
              </div>
              <div className="sticky bottom-0 z-10 border-t bg-background/95 px-5 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6">
                <div className="flex justify-end">
                  <Button
                    type="submit"
                    onClick={() => void onCreateUser()}
                    disabled={!newUsername || !newEmail || !newPassword || newPassword.length < 6}
                  >
                    {t("admin.users.create")}
                  </Button>
                </div>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>
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
                      <DropdownMenuItem onClick={() => void onToggleRole(user)}>
                        {user.role === "admin"
                          ? t("admin.users.makeUser")
                          : t("admin.users.makeAdmin")}
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => void onToggleStatus(user)}>
                        {user.status === "active"
                          ? t("admin.users.disable")
                          : t("admin.users.enable")}
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        className="text-destructive focus:text-destructive"
                        onClick={() => void onDeleteUser(user.id)}
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
  )
}
