import { createElement, type ComponentType, type SVGProps, useEffect, useState } from "react"

type SvgIconComponent = ComponentType<SVGProps<SVGSVGElement>>
type IconValuePrefix = "bl" | "lg"

export type BrandIconComponent = ComponentType<{ size?: number | string; color?: string; className?: string }>

interface BrandIconSpec {
  prefix: IconValuePrefix
  slug: string
  title: string
  hex: string
  keywords?: string[]
  loadIcon: () => Promise<SvgIconComponent>
}

export interface BrandIcon {
  slug: string
  value: string
  title: string
  hex: string
  keywords: string[]
  Icon: BrandIconComponent
}

const brandSpecs: BrandIconSpec[] = [
  {
    prefix: "lg",
    slug: "paypal",
    title: "PayPal",
    hex: "#003087",
    keywords: ["payment"],
    loadIcon: () => import("@icongo/lg/esm/LGPaypal").then((module) => module.LGPaypal),
  },
  {
    prefix: "lg",
    slug: "stripe",
    title: "Stripe",
    hex: "#635BFF",
    keywords: ["payment"],
    loadIcon: () => import("@icongo/lg/esm/LGStripe").then((module) => module.LGStripe),
  },
  {
    prefix: "lg",
    slug: "visa",
    title: "Visa",
    hex: "#1434CB",
    keywords: ["payment", "card"],
    loadIcon: () => import("@icongo/lg/esm/LGVisa").then((module) => module.LGVisa),
  },
  {
    prefix: "lg",
    slug: "mastercard",
    title: "Mastercard",
    hex: "#EB001B",
    keywords: ["payment", "card"],
    loadIcon: () => import("@icongo/lg/esm/LGMastercard").then((module) => module.LGMastercard),
  },
  {
    prefix: "lg",
    slug: "amex",
    title: "American Express",
    hex: "#2E77BC",
    keywords: ["payment", "card", "american express"],
    loadIcon: () => import("@icongo/lg/esm/LGAmex").then((module) => module.LGAmex),
  },
  {
    prefix: "lg",
    slug: "jcb",
    title: "JCB",
    hex: "#0B4EA2",
    keywords: ["payment", "card"],
    loadIcon: () => import("@icongo/lg/esm/LGJcb").then((module) => module.LGJcb),
  },
  {
    prefix: "lg",
    slug: "unionpay",
    title: "UnionPay",
    hex: "#00447C",
    keywords: ["payment", "card", "银联"],
    loadIcon: () => import("@icongo/lg/esm/LGUnionpay").then((module) => module.LGUnionpay),
  },
  {
    prefix: "lg",
    slug: "applepay",
    title: "Apple Pay",
    hex: "#000000",
    keywords: ["payment", "wallet"],
    loadIcon: () => import("@icongo/lg/esm/LGApplePay").then((module) => module.LGApplePay),
  },
  {
    prefix: "lg",
    slug: "googlepay",
    title: "Google Pay",
    hex: "#4285F4",
    keywords: ["payment", "wallet"],
    loadIcon: () => import("@icongo/lg/esm/LGGooglePayIcon").then((module) => module.LGGooglePayIcon),
  },
  {
    prefix: "lg",
    slug: "openai",
    title: "OpenAI",
    hex: "#412991",
    keywords: ["chatgpt", "gpt", "ai"],
    loadIcon: () => import("@icongo/lg/esm/LGOpenaiIcon").then((module) => module.LGOpenaiIcon),
  },
  {
    prefix: "lg",
    slug: "google",
    title: "Google",
    hex: "#4285F4",
    keywords: ["search", "workspace"],
    loadIcon: () => import("@icongo/lg/esm/LGGoogleIcon").then((module) => module.LGGoogleIcon),
  },
  {
    prefix: "lg",
    slug: "googlecloud",
    title: "Google Cloud",
    hex: "#4285F4",
    keywords: ["gcp", "cloud"],
    loadIcon: () => import("@icongo/lg/esm/LGGoogleCloud").then((module) => module.LGGoogleCloud),
  },
  {
    prefix: "lg",
    slug: "cloudflare",
    title: "Cloudflare",
    hex: "#F38020",
    keywords: ["cloud", "cdn"],
    loadIcon: () => import("@icongo/lg/esm/LGCloudflare").then((module) => module.LGCloudflare),
  },
  {
    prefix: "lg",
    slug: "digitalocean",
    title: "DigitalOcean",
    hex: "#0080FF",
    keywords: ["cloud"],
    loadIcon: () => import("@icongo/vl/esm/VLDigitalocean").then((module) => module.VLDigitalocean),
  },
  {
    prefix: "lg",
    slug: "vercel",
    title: "Vercel",
    hex: "#000000",
    keywords: ["hosting"],
    loadIcon: () => import("@icongo/lg/esm/LGVercelIcon").then((module) => module.LGVercelIcon),
  },
  {
    prefix: "lg",
    slug: "netlify",
    title: "Netlify",
    hex: "#00C7B7",
    keywords: ["hosting"],
    loadIcon: () => import("@icongo/lg/esm/LGNetlify").then((module) => module.LGNetlify),
  },
  {
    prefix: "lg",
    slug: "aws",
    title: "AWS",
    hex: "#FF9900",
    keywords: ["amazon cloud", "cloud"],
    loadIcon: () => import("@icongo/lg/esm/LGAws").then((module) => module.LGAws),
  },
  {
    prefix: "lg",
    slug: "oracle",
    title: "Oracle",
    hex: "#F80000",
    keywords: ["cloud"],
    loadIcon: () => import("@icongo/vl/esm/VLOracle").then((module) => module.VLOracle),
  },
  {
    prefix: "lg",
    slug: "ibm",
    title: "IBM",
    hex: "#1261FE",
    keywords: ["cloud"],
    loadIcon: () => import("@icongo/lg/esm/LGIbm").then((module) => module.LGIbm),
  },
  {
    prefix: "lg",
    slug: "namecheap",
    title: "Namecheap",
    hex: "#FF6A00",
    keywords: ["domain"],
    loadIcon: () => import("@icongo/lg/esm/LGNamecheap").then((module) => module.LGNamecheap),
  },
  {
    prefix: "lg",
    slug: "github",
    title: "GitHub",
    hex: "#181717",
    keywords: ["code", "git"],
    loadIcon: () => import("@icongo/lg/esm/LGGithubIcon").then((module) => module.LGGithubIcon),
  },
  {
    prefix: "lg",
    slug: "gitlab",
    title: "GitLab",
    hex: "#FC6D26",
    keywords: ["code", "git"],
    loadIcon: () => import("@icongo/lg/esm/LGGitlab").then((module) => module.LGGitlab),
  },
  {
    prefix: "lg",
    slug: "docker",
    title: "Docker",
    hex: "#2496ED",
    keywords: ["container"],
    loadIcon: () => import("@icongo/lg/esm/LGDockerIcon").then((module) => module.LGDockerIcon),
  },
  {
    prefix: "lg",
    slug: "figma",
    title: "Figma",
    hex: "#A259FF",
    keywords: ["design"],
    loadIcon: () => import("@icongo/lg/esm/LGFigma").then((module) => module.LGFigma),
  },
  {
    prefix: "lg",
    slug: "postman",
    title: "Postman",
    hex: "#FF6C37",
    keywords: ["api"],
    loadIcon: () => import("@icongo/lg/esm/LGPostmanIcon").then((module) => module.LGPostmanIcon),
  },
  {
    prefix: "lg",
    slug: "insomnia",
    title: "Insomnia",
    hex: "#4000BF",
    keywords: ["api"],
    loadIcon: () => import("@icongo/lg/esm/LGInsomnia").then((module) => module.LGInsomnia),
  },
  {
    prefix: "lg",
    slug: "jira",
    title: "Jira",
    hex: "#0052CC",
    keywords: ["project"],
    loadIcon: () => import("@icongo/lg/esm/LGJira").then((module) => module.LGJira),
  },
  {
    prefix: "lg",
    slug: "trello",
    title: "Trello",
    hex: "#0052CC",
    keywords: ["project"],
    loadIcon: () => import("@icongo/lg/esm/LGTrello").then((module) => module.LGTrello),
  },
  {
    prefix: "lg",
    slug: "asana",
    title: "Asana",
    hex: "#F06A6A",
    keywords: ["project"],
    loadIcon: () => import("@icongo/lg/esm/LGAsanaIcon").then((module) => module.LGAsanaIcon),
  },
  {
    prefix: "lg",
    slug: "youtube",
    title: "YouTube",
    hex: "#FF0000",
    keywords: ["video"],
    loadIcon: () => import("@icongo/lg/esm/LGYoutubeIcon").then((module) => module.LGYoutubeIcon),
  },
  {
    prefix: "lg",
    slug: "spotify",
    title: "Spotify",
    hex: "#1DB954",
    keywords: ["music"],
    loadIcon: () => import("@icongo/lg/esm/LGSpotifyIcon").then((module) => module.LGSpotifyIcon),
  },
  {
    prefix: "lg",
    slug: "netflix",
    title: "Netflix",
    hex: "#E50914",
    keywords: ["video"],
    loadIcon: () => import("@icongo/vl/esm/VLNetflix").then((module) => module.VLNetflix),
  },
  {
    prefix: "lg",
    slug: "apple",
    title: "Apple",
    hex: "#000000",
    keywords: ["ios", "mac"],
    loadIcon: () => import("@icongo/lg/esm/LGAppleAppStore").then((module) => module.LGAppleAppStore),
  },
  {
    prefix: "lg",
    slug: "tiktok",
    title: "TikTok",
    hex: "#000000",
    keywords: ["short video"],
    loadIcon: () => import("@icongo/lg/esm/LGTiktokIcon").then((module) => module.LGTiktokIcon),
  },
  {
    prefix: "lg",
    slug: "discord",
    title: "Discord",
    hex: "#5865F2",
    keywords: ["chat"],
    loadIcon: () => import("@icongo/lg/esm/LGDiscordIcon").then((module) => module.LGDiscordIcon),
  },
  {
    prefix: "lg",
    slug: "telegram",
    title: "Telegram",
    hex: "#26A5E4",
    keywords: ["chat"],
    loadIcon: () => import("@icongo/lg/esm/LGTelegram").then((module) => module.LGTelegram),
  },
  {
    prefix: "lg",
    slug: "facebook",
    title: "Facebook",
    hex: "#1877F2",
    keywords: ["social"],
    loadIcon: () => import("@icongo/lg/esm/LGFacebook").then((module) => module.LGFacebook),
  },
  {
    prefix: "lg",
    slug: "instagram",
    title: "Instagram",
    hex: "#E4405F",
    keywords: ["social"],
    loadIcon: () => import("@icongo/lg/esm/LGInstagramIcon").then((module) => module.LGInstagramIcon),
  },
  {
    prefix: "lg",
    slug: "bitcoin",
    title: "Bitcoin",
    hex: "#F7931A",
    keywords: ["btc", "crypto"],
    loadIcon: () => import("@icongo/lg/esm/LGBitcoin").then((module) => module.LGBitcoin),
  },
  {
    prefix: "lg",
    slug: "ethereum",
    title: "Ethereum",
    hex: "#3C3C3D",
    keywords: ["eth", "crypto"],
    loadIcon: () => import("@icongo/lg/esm/LGEthereum").then((module) => module.LGEthereum),
  },
  {
    prefix: "lg",
    slug: "monero",
    title: "Monero",
    hex: "#FF6600",
    keywords: ["xmr", "crypto"],
    loadIcon: () => import("@icongo/lg/esm/LGMonero").then((module) => module.LGMonero),
  },
  {
    prefix: "lg",
    slug: "kraken",
    title: "Kraken",
    hex: "#5741D9",
    keywords: ["exchange", "crypto"],
    loadIcon: () => import("@icongo/lg/esm/LGKraken").then((module) => module.LGKraken),
  },
  {
    prefix: "bl",
    slug: "bankofamerica",
    title: "Bank of America",
    hex: "#E31837",
    keywords: ["bank", "boa"],
    loadIcon: () => import("@icongo/bl/esm/BLBankofamericaRect").then((module) => module.BLBankofamericaRect),
  },
  {
    prefix: "bl",
    slug: "hsbc",
    title: "HSBC",
    hex: "#DB0011",
    keywords: ["bank"],
    loadIcon: () => import("@icongo/bl/esm/BLHsbcRect").then((module) => module.BLHsbcRect),
  },
  {
    prefix: "bl",
    slug: "bankofchina",
    title: "Bank of China",
    hex: "#A3121A",
    keywords: ["boc", "中国银行", "bank"],
    loadIcon: () => import("@icongo/bl/esm/BLBocRect").then((module) => module.BLBocRect),
  },
  {
    prefix: "bl",
    slug: "icbc",
    title: "ICBC",
    hex: "#C8102E",
    keywords: ["工商银行", "bank"],
    loadIcon: () => import("@icongo/bl/esm/BLIcbcRect").then((module) => module.BLIcbcRect),
  },
  {
    prefix: "bl",
    slug: "ccb",
    title: "China Construction Bank",
    hex: "#005BAC",
    keywords: ["建设银行", "bank"],
    loadIcon: () => import("@icongo/bl/esm/BLCcbRect").then((module) => module.BLCcbRect),
  },
  {
    prefix: "bl",
    slug: "abc",
    title: "Agricultural Bank of China",
    hex: "#007D49",
    keywords: ["农业银行", "bank"],
    loadIcon: () => import("@icongo/bl/esm/BLAbchinaRect").then((module) => module.BLAbchinaRect),
  },
  {
    prefix: "bl",
    slug: "cmbchina",
    title: "China Merchants Bank",
    hex: "#D71920",
    keywords: ["招商银行", "cmb", "bank"],
    loadIcon: () => import("@icongo/bl/esm/BLCmbchinaRect").then((module) => module.BLCmbchinaRect),
  },
  {
    prefix: "bl",
    slug: "cmbc",
    title: "China Minsheng Bank",
    hex: "#005BAC",
    keywords: ["民生银行", "bank"],
    loadIcon: () => import("@icongo/bl/esm/BLCmbcRect").then((module) => module.BLCmbcRect),
  },
  {
    prefix: "bl",
    slug: "citicbank",
    title: "China CITIC Bank",
    hex: "#D32F2F",
    keywords: ["中信银行", "bank"],
    loadIcon: () => import("@icongo/bl/esm/BLCiticbankRect").then((module) => module.BLCiticbankRect),
  },
  {
    prefix: "bl",
    slug: "bankcomm",
    title: "Bank of Communications",
    hex: "#005BAC",
    keywords: ["交通银行", "bank"],
    loadIcon: () => import("@icongo/bl/esm/BLBankcommRect").then((module) => module.BLBankcommRect),
  },
  {
    prefix: "bl",
    slug: "cebbank",
    title: "China Everbright Bank",
    hex: "#6A9E3B",
    keywords: ["光大银行", "bank"],
    loadIcon: () => import("@icongo/bl/esm/BLCebbankRect").then((module) => module.BLCebbankRect),
  },
  {
    prefix: "bl",
    slug: "cib",
    title: "Industrial Bank",
    hex: "#004098",
    keywords: ["兴业银行", "bank"],
    loadIcon: () => import("@icongo/bl/esm/BLCibRect").then((module) => module.BLCibRect),
  },
]

export const brandIcons: BrandIcon[] = brandSpecs.map((spec) => ({
  slug: spec.slug,
  value: `${spec.prefix}:${spec.slug}`,
  title: spec.title,
  hex: spec.hex,
  keywords: spec.keywords ?? [],
  Icon: createLazySvgIcon(spec.loadIcon),
}))

const brandIconMap = new Map(brandIcons.map((icon) => [icon.slug, icon] as const))
const brandIconValueMap = new Map(brandIcons.map((icon) => [icon.value, icon] as const))

export function getBrandIcon(slug: string): BrandIcon | undefined {
  return brandIconMap.get(slug)
}

export function getBrandIconFromValue(value: string): BrandIcon | undefined {
  if (!value) {
    return undefined
  }

  return brandIconValueMap.get(value)
}

function createLazySvgIcon(loadIcon: () => Promise<SvgIconComponent>): BrandIconComponent {
	let loadedIcon: SvgIconComponent | null = null
	let loadingPromise: Promise<SvgIconComponent> | null = null

	function LazySvgIcon({ size = 20, color, className }: { size?: number | string; color?: string; className?: string }) {
		const [Icon, setIcon] = useState<SvgIconComponent | null>(() => loadedIcon)
		const resolvedIcon = Icon ?? loadedIcon

		useEffect(() => {
			let cancelled = false

			if (resolvedIcon) {
				return
			}

      if (!loadingPromise) {
        loadingPromise = loadIcon().then((nextIcon) => {
          loadedIcon = nextIcon
          return nextIcon
        })
      }

      loadingPromise
        .then((nextIcon) => {
          if (!cancelled) {
            setIcon(() => nextIcon)
          }
        })
        .catch(() => {
          if (!cancelled) {
            setIcon(null)
          }
        })

			return () => {
				cancelled = true
			}
		}, [resolvedIcon])

    const resolvedSize = normalizeSize(size)

		if (!resolvedIcon) {
			return createElement("span", {
        className,
        style: {
          width: resolvedSize,
          height: resolvedSize,
          display: "inline-block",
          borderRadius: 4,
          backgroundColor: "var(--muted)",
        },
      })
    }

    const resolvedColor = color === "default" ? undefined : color

		return createElement(resolvedIcon, {
			width: resolvedSize,
			height: resolvedSize,
			color: resolvedColor,
			className,
		})
	}

	return LazySvgIcon
}

function normalizeSize(size: number | string): number {
  if (typeof size === "number") {
    return Number.isFinite(size) ? size : 20
  }

  const parsed = Number.parseFloat(size)
  return Number.isFinite(parsed) ? parsed : 20
}
