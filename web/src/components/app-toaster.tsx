import { Toaster as SonnerToaster, type ToasterProps } from "sonner"
import { useTheme } from "@/lib/theme"

export function AppToaster() {
  const theme = useTheme()

  return (
    <SonnerToaster
      theme={theme as ToasterProps["theme"]}
      richColors
      closeButton
      position="top-right"
      toastOptions={{
        duration: 4000,
        classNames: {
          toast: "ulw-toast group toast",
          description: "group-[.toast]:text-muted-foreground",
          actionButton: "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
          cancelButton: "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground",
        },
      }}
    />
  )
}
