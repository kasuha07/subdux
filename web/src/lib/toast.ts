import type { ToasterProps } from "sonner"

export { toast } from "sonner"

export const appToasterProps = {
  richColors: true,
  position: "top-right",
  toastOptions: {
    duration: 4000,
    classNames: {
      toast: "ulw-toast group toast",
      description: "group-[.toast]:text-muted-foreground",
      actionButton: "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
      cancelButton: "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground",
    },
  },
} satisfies Pick<ToasterProps, "position" | "richColors" | "toastOptions">
