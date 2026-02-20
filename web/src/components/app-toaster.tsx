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
          toast:
            "ulw-toast group toast group-[.toaster]:bg-background group-[.toaster]:text-foreground group-[.toaster]:border group-[.toaster]:border-border group-[.toaster]:shadow-lg",
          description: "group-[.toast]:text-muted-foreground",
          actionButton: "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
          cancelButton: "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground",
        },
      }}
    />
  )
}
