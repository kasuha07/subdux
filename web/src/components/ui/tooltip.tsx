"use client"

import * as React from "react"
import { Tooltip as TooltipPrimitive } from "radix-ui"

import { cn } from "@/lib/utils"

const TooltipProviderContext = React.createContext(false)

function TooltipProvider({
  delayDuration = 200,
  children,
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Provider>) {
  return (
    <TooltipProviderContext.Provider value={true}>
      <TooltipPrimitive.Provider
        data-slot="tooltip-provider"
        delayDuration={delayDuration}
        {...props}
      >
        {children}
      </TooltipPrimitive.Provider>
    </TooltipProviderContext.Provider>
  )
}

export interface TooltipProps extends React.ComponentProps<typeof TooltipPrimitive.Root> {
  content?: React.ReactNode
  side?: React.ComponentProps<typeof TooltipPrimitive.Content>["side"]
  sideOffset?: number
  align?: React.ComponentProps<typeof TooltipPrimitive.Content>["align"]
  contentClassName?: string
  showArrow?: boolean
}

function Tooltip({
  content,
  side,
  sideOffset = 6,
  align,
  contentClassName,
  showArrow,
  children,
  ...props
}: TooltipProps) {
  const hasProvider = React.useContext(TooltipProviderContext)

  const contentElement = (
    content !== undefined && content !== null && content !== false ? (
      <TooltipPrimitive.Root data-slot="tooltip" {...props}>
        <TooltipPrimitive.Trigger data-slot="tooltip-trigger" asChild>
          {children as React.ReactElement}
        </TooltipPrimitive.Trigger>
        <TooltipContent
          side={side}
          sideOffset={sideOffset}
          align={align}
          showArrow={showArrow}
          className={contentClassName}
        >
          {content}
        </TooltipContent>
      </TooltipPrimitive.Root>
    ) : (
      <TooltipPrimitive.Root data-slot="tooltip" {...props}>{children}</TooltipPrimitive.Root>
    )
  )

  if (!hasProvider) {
    return <TooltipProvider>{contentElement}</TooltipProvider>
  }

  return contentElement
}

function TooltipTrigger({
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Trigger>) {
  return <TooltipPrimitive.Trigger data-slot="tooltip-trigger" {...props} />
}

function TooltipArrow({
  className,
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Arrow>) {
  return (
    <TooltipPrimitive.Arrow
      data-slot="tooltip-arrow"
      className={cn("fill-popover", className)}
      {...props}
    />
  )
}

export interface TooltipContentProps
  extends React.ComponentProps<typeof TooltipPrimitive.Content> {
  showArrow?: boolean
}

function TooltipContent({
  className,
  sideOffset = 6,
  children,
  showArrow = false,
  ...props
}: TooltipContentProps) {
  return (
    <TooltipPrimitive.Portal>
      <TooltipPrimitive.Content
        data-slot="tooltip-content"
        sideOffset={sideOffset}
        className={cn(
          "motion-surface bg-popover text-popover-foreground z-50 overflow-hidden rounded-md border px-2.5 py-1 text-xs shadow-md select-none",
          className
        )}
        {...props}
      >
        {children}
        {showArrow && <TooltipArrow />}
      </TooltipPrimitive.Content>
    </TooltipPrimitive.Portal>
  )
}

export {
  Tooltip,
  Tooltip as SimpleTooltip,
  TooltipTrigger,
  TooltipContent,
  TooltipProvider,
  TooltipArrow,
}
