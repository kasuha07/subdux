import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type DragEvent,
  type FormEvent,
} from "react"
import { useTranslation } from "react-i18next"

import { api } from "@/lib/api"
import {
  DEFAULT_CURRENCY_FALLBACK,
  getPresetCurrencies,
  getPresetCurrencyMeta,
} from "@/lib/currencies"
import { getDefaultCurrency, setDefaultCurrency } from "@/lib/default-currency"
import { toast } from "sonner"
import type {
  CreateCurrencyInput,
  ReorderCurrencyItem,
  UpdateCurrencyInput,
  UserCurrency,
  UserPreference,
} from "@/types"

const customCodeOption = "__custom__"
const currencyPlaceholderFallbackCode = "USD"

interface UseSettingsPaymentOptions {
  active: boolean
}

interface UseSettingsPaymentResult {
  addAlias: string
  addAliasPlaceholder: string
  addCode: string
  addLoading: boolean
  addSymbol: string
  addSymbolPlaceholder: string
  addableCurrencyCodes: string[]
  currency: string
  customCode: string
  customCodeOption: string
  getCurrencyAliasPlaceholder: (code: string) => string
  getCurrencySymbolPlaceholder: (code: string) => string
  handleAddCurrency: (event: FormEvent<HTMLFormElement>) => Promise<void>
  handleCurrency: (value: string) => Promise<void>
  handleDeleteCurrency: (id: number) => Promise<void>
  handleDragOver: (event: DragEvent<HTMLDivElement>, index: number) => void
  handleDragStart: (index: number) => void
  handleDrop: () => void
  handleSaveOrder: () => Promise<void>
  handleUpdateCurrency: (id: number, input: UpdateCurrencyInput) => Promise<void>
  orderChanged: boolean
  orderSaving: boolean
  preferredCurrencyCodes: string[]
  setAddAlias: (value: string) => void
  setAddCode: (value: string) => void
  setAddSymbol: (value: string) => void
  setCustomCode: (value: string) => void
  userCurrencies: UserCurrency[]
}

function getIntlCurrencySymbolPlaceholder(code: string, locale: string): string | null {
  try {
    const parts = new Intl.NumberFormat(locale, {
      style: "currency",
      currency: code,
      currencyDisplay: "symbol",
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).formatToParts(0)
    return parts.find((part) => part.type === "currency")?.value ?? null
  } catch {
    return null
  }
}

function getIntlCurrencyAliasPlaceholder(code: string, locale: string): string | null {
  try {
    if (typeof Intl.DisplayNames !== "function") {
      return null
    }
    const alias = new Intl.DisplayNames([locale], { type: "currency" }).of(code)
    return alias ?? null
  } catch {
    return null
  }
}

export function useSettingsPayment({ active }: UseSettingsPaymentOptions): UseSettingsPaymentResult {
  const { i18n, t } = useTranslation()

  const [currency, setCurrency] = useState(getDefaultCurrency())
  const [userCurrencies, setUserCurrencies] = useState<UserCurrency[]>([])
  const [addCode, setAddCode] = useState("")
  const [customCode, setCustomCode] = useState("")
  const [addSymbol, setAddSymbol] = useState("")
  const [addAlias, setAddAlias] = useState("")
  const [addLoading, setAddLoading] = useState(false)
  const [orderChanged, setOrderChanged] = useState(false)
  const [orderSaving, setOrderSaving] = useState(false)

  const dragFrom = useRef<number | null>(null)
  const dragTo = useRef<number | null>(null)
  const paymentLoaded = useRef(false)

  useEffect(() => {
    if (!active || paymentLoaded.current) {
      return
    }

    paymentLoaded.current = true

    api.get<UserPreference>("/preferences/currency")
      .then((pref) => {
        if (pref?.preferred_currency) {
          setCurrency(pref.preferred_currency)
          setDefaultCurrency(pref.preferred_currency)
        }
      })
      .catch(() => void 0)

    api.get<UserCurrency[]>("/currencies")
      .then((list) => {
        setUserCurrencies(list ?? [])
      })
      .catch(() => void 0)
  }, [active])

  useEffect(() => {
    if (userCurrencies.length === 0) {
      return
    }

    if (!userCurrencies.some((item) => item.code === currency)) {
      const nextCurrency = userCurrencies[0].code
      setCurrency(nextCurrency)
      setDefaultCurrency(nextCurrency)
      api.put("/preferences/currency", { preferred_currency: nextCurrency }).catch(() => void 0)
    }
  }, [currency, userCurrencies])

  const addableCurrencyCodes = useMemo(
    () =>
      getPresetCurrencies(i18n.language)
        .map((item) => item.code)
        .filter((code) => !userCurrencies.some((item) => item.code === code)),
    [i18n.language, userCurrencies]
  )

  const addPlaceholderCode = useMemo(() => {
    if (addCode === customCodeOption) {
      return customCode.trim().toUpperCase() || currencyPlaceholderFallbackCode
    }
    return addCode || currencyPlaceholderFallbackCode
  }, [addCode, customCode])

  const intlFallbackSymbolPlaceholder = useMemo(
    () =>
      getIntlCurrencySymbolPlaceholder(currencyPlaceholderFallbackCode, i18n.language) ??
      currencyPlaceholderFallbackCode,
    [i18n.language]
  )

  const intlFallbackAliasPlaceholder = useMemo(
    () =>
      getIntlCurrencyAliasPlaceholder(currencyPlaceholderFallbackCode, i18n.language) ??
      currencyPlaceholderFallbackCode,
    [i18n.language]
  )

  const addSymbolPlaceholder = useMemo(() => {
    const preset = getPresetCurrencyMeta(addPlaceholderCode, i18n.language)
    return (
      preset?.symbol ??
      getIntlCurrencySymbolPlaceholder(addPlaceholderCode, i18n.language) ??
      intlFallbackSymbolPlaceholder
    )
  }, [addPlaceholderCode, i18n.language, intlFallbackSymbolPlaceholder])

  const addAliasPlaceholder = useMemo(() => {
    const preset = getPresetCurrencyMeta(addPlaceholderCode, i18n.language)
    return (
      preset?.alias ??
      getIntlCurrencyAliasPlaceholder(addPlaceholderCode, i18n.language) ??
      intlFallbackAliasPlaceholder
    )
  }, [addPlaceholderCode, i18n.language, intlFallbackAliasPlaceholder])

  const getCurrencySymbolPlaceholder = useCallback(
    (code: string): string => {
      const preset = getPresetCurrencyMeta(code, i18n.language)
      return (
        preset?.symbol ??
        getIntlCurrencySymbolPlaceholder(code, i18n.language) ??
        intlFallbackSymbolPlaceholder
      )
    },
    [i18n.language, intlFallbackSymbolPlaceholder]
  )

  const getCurrencyAliasPlaceholder = useCallback(
    (code: string): string => {
      const preset = getPresetCurrencyMeta(code, i18n.language)
      return (
        preset?.alias ??
        getIntlCurrencyAliasPlaceholder(code, i18n.language) ??
        intlFallbackAliasPlaceholder
      )
    },
    [i18n.language, intlFallbackAliasPlaceholder]
  )

  useEffect(() => {
    if (addCode === customCodeOption) {
      setAddSymbol("")
      setAddAlias("")
      return
    }

    if (addableCurrencyCodes.length === 0) {
      setAddCode(customCodeOption)
      return
    }

    if (!addCode || !addableCurrencyCodes.includes(addCode)) {
      setAddCode(addableCurrencyCodes[0])
    }
  }, [addCode, addableCurrencyCodes])

  useEffect(() => {
    if (addCode === customCodeOption) {
      return
    }

    const preset = getPresetCurrencyMeta(addCode, i18n.language)
    if (!preset) {
      setAddSymbol("")
      setAddAlias("")
      return
    }

    setAddSymbol(preset.symbol)
    setAddAlias(preset.alias)
  }, [addCode, i18n.language])

  async function handleCurrency(value: string) {
    setCurrency(value)
    setDefaultCurrency(value)
    try {
      await api.put("/preferences/currency", { preferred_currency: value })
    } catch {
      void 0
    }
  }

  async function handleAddCurrency(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const code = (addCode === customCodeOption ? customCode : addCode).trim().toUpperCase()
    const preset = getPresetCurrencyMeta(code, i18n.language)
    if (!code) {
      toast.error(t("settings.currencyManagement.invalidCode"))
      return
    }

    setAddLoading(true)
    try {
      const input: CreateCurrencyInput = {
        code,
        symbol: addSymbol.trim() || preset?.symbol || "",
        alias: addAlias.trim() || preset?.alias || "",
        sort_order: userCurrencies.length,
      }
      const created = await api.post<UserCurrency>("/currencies", input)
      setUserCurrencies((prev) => [...prev, created])
      if (addCode === customCodeOption) {
        setCustomCode("")
        setAddSymbol("")
        setAddAlias("")
      }
      toast.success(t("settings.currencyManagement.addSuccess"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("settings.currencyManagement.invalidCode"))
    } finally {
      setAddLoading(false)
    }
  }

  async function handleDeleteCurrency(id: number) {
    if (!window.confirm(t("settings.currencyManagement.deleteConfirm"))) {
      return
    }

    try {
      await api.delete(`/currencies/${id}`)
      setUserCurrencies((prev) => prev.filter((item) => item.id !== id))
      toast.success(t("settings.currencyManagement.deleteSuccess"))
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : t("settings.currencyManagement.cannotDeletePreferred")
      )
    }
  }

  async function handleUpdateCurrency(id: number, input: UpdateCurrencyInput) {
    try {
      const updated = await api.put<UserCurrency>(`/currencies/${id}`, input)
      setUserCurrencies((prev) => prev.map((item) => (item.id === id ? updated : item)))
      toast.success(t("settings.currencyManagement.updateSuccess"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.requestFailed"))
    }
  }

  function handleDragStart(index: number) {
    dragFrom.current = index
  }

  function handleDragOver(event: DragEvent<HTMLDivElement>, index: number) {
    event.preventDefault()
    dragTo.current = index
  }

  function handleDrop() {
    if (dragFrom.current === null || dragTo.current === null || dragFrom.current === dragTo.current) {
      return
    }

    const reordered = [...userCurrencies]
    const [moved] = reordered.splice(dragFrom.current, 1)
    reordered.splice(dragTo.current, 0, moved)
    setUserCurrencies(reordered)
    setOrderChanged(true)
    dragFrom.current = null
    dragTo.current = null
  }

  async function handleSaveOrder() {
    setOrderSaving(true)
    try {
      const payload: ReorderCurrencyItem[] = userCurrencies.map((item, index) => ({
        id: item.id,
        sort_order: index,
      }))
      await api.put("/currencies/reorder", payload)
      setUserCurrencies((prev) =>
        prev.map((item, index) => ({
          ...item,
          sort_order: index,
        }))
      )
      setOrderChanged(false)
      toast.success(t("settings.currencyManagement.orderSaved"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.requestFailed"))
    } finally {
      setOrderSaving(false)
    }
  }

  return {
    addAlias,
    addAliasPlaceholder,
    addCode,
    addLoading,
    addSymbol,
    addSymbolPlaceholder,
    addableCurrencyCodes,
    currency,
    customCode,
    customCodeOption,
    getCurrencyAliasPlaceholder,
    getCurrencySymbolPlaceholder,
    handleAddCurrency,
    handleCurrency,
    handleDeleteCurrency,
    handleDragOver,
    handleDragStart,
    handleDrop,
    handleSaveOrder,
    handleUpdateCurrency,
    orderChanged,
    orderSaving,
    preferredCurrencyCodes:
      userCurrencies.length > 0
        ? userCurrencies.map((item) => item.code)
        : DEFAULT_CURRENCY_FALLBACK,
    setAddAlias,
    setAddCode,
    setAddSymbol,
    setCustomCode,
    userCurrencies,
  }
}
