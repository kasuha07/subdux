import { useEffect, type ReactNode } from "react"
import { useLocation } from "react-router"
import { cn } from "@/lib/utils"

interface PageTransitionProps {
  children: ReactNode
  className?: string
}

export function PageTransition({ children, className }: PageTransitionProps) {
  const location = useLocation()

  useEffect(() => {
    window.scrollTo({ top: 0, left: 0, behavior: "instant" as ScrollBehavior })
  }, [location.pathname])

  return (
    <div
      key={location.pathname}
      className={cn("page-transition flex min-h-screen flex-col", className)}
      data-page-path={location.pathname}
    >
      {children}
    </div>
  )
}
