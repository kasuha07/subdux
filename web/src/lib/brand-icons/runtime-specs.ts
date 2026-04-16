import { loadCustomSvgIcon } from "./custom-loaders"

import type { IconValuePrefix, SvgIconComponent } from "./types"

export interface BrandIconRuntimeSpec {
  prefix: IconValuePrefix
  slug: string
  loadIcon: () => Promise<SvgIconComponent>
}

export const brandRuntimeSpecs: BrandIconRuntimeSpec[] = [
  {
    prefix: "lg",
    slug: "paypal",
    loadIcon: () => import("@icongo/lg/esm/LGPaypal").then((module) => module.LGPaypal),
  },
  {
    prefix: "lg",
    slug: "stripe",
    loadIcon: () => import("@icongo/lg/esm/LGStripe").then((module) => module.LGStripe),
  },
  {
    prefix: "lg",
    slug: "visa",
    loadIcon: () => import("@icongo/lg/esm/LGVisa").then((module) => module.LGVisa),
  },
  {
    prefix: "lg",
    slug: "mastercard",
    loadIcon: () => import("@icongo/lg/esm/LGMastercard").then((module) => module.LGMastercard),
  },
  {
    prefix: "lg",
    slug: "amex",
    loadIcon: () => import("@icongo/lg/esm/LGAmex").then((module) => module.LGAmex),
  },
  {
    prefix: "lg",
    slug: "jcb",
    loadIcon: () => import("@icongo/lg/esm/LGJcb").then((module) => module.LGJcb),
  },
  {
    prefix: "lg",
    slug: "unionpay",
    loadIcon: () => import("@icongo/lg/esm/LGUnionpay").then((module) => module.LGUnionpay),
  },
  {
    prefix: "lg",
    slug: "applepay",
    loadIcon: () => import("@icongo/lg/esm/LGApplePay").then((module) => module.LGApplePay),
  },
  {
    prefix: "lg",
    slug: "googlepay",
    loadIcon: () => import("@icongo/lg/esm/LGGooglePayIcon").then((module) => module.LGGooglePayIcon),
  },
  {
    prefix: "lg",
    slug: "googleplay",
    loadIcon: () => import("@icongo/vl/esm/VLGooglePlay").then((module) => module.VLGooglePlay),
  },
  {
    prefix: "custom",
    slug: "ecny",
    loadIcon: loadCustomSvgIcon("ecny", "SXEcny"),
  },
  {
    prefix: "custom",
    slug: "cash",
    loadIcon: loadCustomSvgIcon("cash", "SXCash"),
  },
  {
    prefix: "custom",
    slug: "alipay",
    loadIcon: loadCustomSvgIcon("alipay", "SXAlipay"),
  },
  {
    prefix: "custom",
    slug: "wechatpay",
    loadIcon: loadCustomSvgIcon("wechatpay", "SXWechatPay"),
  },
  {
    prefix: "lg",
    slug: "openai",
    loadIcon: () => import("@icongo/lg/esm/LGOpenaiIcon").then((module) => module.LGOpenaiIcon),
  },
  {
    prefix: "custom",
    slug: "cursor",
    loadIcon: loadCustomSvgIcon("cursor", "SXCursor"),
  },
  {
    prefix: "custom",
    slug: "anthropic",
    loadIcon: loadCustomSvgIcon("anthropic", "SXAnthropic"),
  },
  {
    prefix: "custom",
    slug: "claude",
    loadIcon: loadCustomSvgIcon("claude", "SXClaude"),
  },
  {
    prefix: "custom",
    slug: "gemini",
    loadIcon: loadCustomSvgIcon("gemini", "SXGemini"),
  },
  {
    prefix: "custom",
    slug: "newapi",
    loadIcon: loadCustomSvgIcon("newapi", "SXNewapi"),
  },
  {
    prefix: "custom",
    slug: "grok",
    loadIcon: loadCustomSvgIcon("grok", "SXGrok"),
  },
  {
    prefix: "custom",
    slug: "mistral",
    loadIcon: loadCustomSvgIcon("mistral", "SXMistral"),
  },
  {
    prefix: "custom",
    slug: "perplexity",
    loadIcon: loadCustomSvgIcon("perplexity", "SXPerplexity"),
  },
  {
    prefix: "custom",
    slug: "kimi",
    loadIcon: loadCustomSvgIcon("kimi", "SXKimi"),
  },
  {
    prefix: "custom",
    slug: "moonshot",
    loadIcon: loadCustomSvgIcon("moonshot", "SXMoonshot"),
  },
  {
    prefix: "custom",
    slug: "zhipu",
    loadIcon: loadCustomSvgIcon("zhipu", "SXZhipu"),
  },
  {
    prefix: "custom",
    slug: "zai",
    loadIcon: loadCustomSvgIcon("zai", "SXZai"),
  },
  {
    prefix: "custom",
    slug: "qwen",
    loadIcon: loadCustomSvgIcon("qwen", "SXQwen"),
  },
  {
    prefix: "custom",
    slug: "deepseek",
    loadIcon: loadCustomSvgIcon("deepseek", "SXDeepseek"),
  },
  {
    prefix: "custom",
    slug: "huggingface",
    loadIcon: loadCustomSvgIcon("huggingface", "SXHuggingFace"),
  },
  {
    prefix: "lg",
    slug: "google",
    loadIcon: () => import("@icongo/lg/esm/LGGoogleIcon").then((module) => module.LGGoogleIcon),
  },
  {
    prefix: "lg",
    slug: "googlecloud",
    loadIcon: () => import("@icongo/lg/esm/LGGoogleCloud").then((module) => module.LGGoogleCloud),
  },
  {
    prefix: "lg",
    slug: "cloudflare",
    loadIcon: () => import("@icongo/lg/esm/LGCloudflare").then((module) => module.LGCloudflare),
  },
  {
    prefix: "lg",
    slug: "digitalocean",
    loadIcon: () => import("@icongo/vl/esm/VLDigitalocean").then((module) => module.VLDigitalocean),
  },
  {
    prefix: "lg",
    slug: "vercel",
    loadIcon: () => import("@icongo/lg/esm/LGVercelIcon").then((module) => module.LGVercelIcon),
  },
  {
    prefix: "lg",
    slug: "netlify",
    loadIcon: () => import("@icongo/lg/esm/LGNetlify").then((module) => module.LGNetlify),
  },
  {
    prefix: "lg",
    slug: "aws",
    loadIcon: () => import("@icongo/lg/esm/LGAws").then((module) => module.LGAws),
  },
  {
    prefix: "lg",
    slug: "oracle",
    loadIcon: () => import("@icongo/vl/esm/VLOracle").then((module) => module.VLOracle),
  },
  {
    prefix: "lg",
    slug: "ibm",
    loadIcon: () => import("@icongo/lg/esm/LGIbm").then((module) => module.LGIbm),
  },
  {
    prefix: "lg",
    slug: "namecheap",
    loadIcon: () => import("@icongo/lg/esm/LGNamecheap").then((module) => module.LGNamecheap),
  },
  {
    prefix: "custom",
    slug: "netcup",
    loadIcon: loadCustomSvgIcon("netcup", "SXNetcup"),
  },
  {
    prefix: "custom",
    slug: "namecom",
    loadIcon: loadCustomSvgIcon("namecom", "SXNameCom"),
  },
  {
    prefix: "custom",
    slug: "porkbun",
    loadIcon: loadCustomSvgIcon("porkbun", "SXPorkbun"),
  },
  {
    prefix: "custom",
    slug: "spaceship",
    loadIcon: loadCustomSvgIcon("spaceship", "SXSpaceship"),
  },
  {
    prefix: "lg",
    slug: "ovh",
    loadIcon: () => import("@icongo/si/esm/SIOvh").then((module) => module.SIOvh),
  },
  {
    prefix: "lg",
    slug: "scaleway",
    loadIcon: () => import("@icongo/si/esm/SIScaleway").then((module) => module.SIScaleway),
  },
  {
    prefix: "custom",
    slug: "tencentcloud",
    loadIcon: loadCustomSvgIcon("tencentcloud", "SXTencentCloud"),
  },
  {
    prefix: "custom",
    slug: "baiducloud",
    loadIcon: loadCustomSvgIcon("baiducloud", "SXBaiduCloud"),
  },
  {
    prefix: "custom",
    slug: "westcn",
    loadIcon: loadCustomSvgIcon("westcn", "SXWestCn"),
  },
  {
    prefix: "custom",
    slug: "frp",
    loadIcon: loadCustomSvgIcon("frp", "SXFrp"),
  },
  {
    prefix: "custom",
    slug: "linuxdo",
    loadIcon: loadCustomSvgIcon("linuxdo", "SXLinuxDo"),
  },
  {
    prefix: "lg",
    slug: "github",
    loadIcon: () => import("@icongo/lg/esm/LGGithubIcon").then((module) => module.LGGithubIcon),
  },
  {
    prefix: "lg",
    slug: "gitlab",
    loadIcon: () => import("@icongo/lg/esm/LGGitlab").then((module) => module.LGGitlab),
  },
  {
    prefix: "lg",
    slug: "ipfs",
    loadIcon: () => import("@icongo/si/esm/SIIpfs").then((module) => module.SIIpfs),
  },
  {
    prefix: "lg",
    slug: "docker",
    loadIcon: () => import("@icongo/lg/esm/LGDockerIcon").then((module) => module.LGDockerIcon),
  },
  {
    prefix: "lg",
    slug: "figma",
    loadIcon: () => import("@icongo/lg/esm/LGFigma").then((module) => module.LGFigma),
  },
  {
    prefix: "lg",
    slug: "postman",
    loadIcon: () => import("@icongo/lg/esm/LGPostmanIcon").then((module) => module.LGPostmanIcon),
  },
  {
    prefix: "lg",
    slug: "insomnia",
    loadIcon: () => import("@icongo/lg/esm/LGInsomnia").then((module) => module.LGInsomnia),
  },
  {
    prefix: "lg",
    slug: "jira",
    loadIcon: () => import("@icongo/lg/esm/LGJira").then((module) => module.LGJira),
  },
  {
    prefix: "lg",
    slug: "trello",
    loadIcon: () => import("@icongo/lg/esm/LGTrello").then((module) => module.LGTrello),
  },
  {
    prefix: "lg",
    slug: "asana",
    loadIcon: () => import("@icongo/lg/esm/LGAsanaIcon").then((module) => module.LGAsanaIcon),
  },
  {
    prefix: "lg",
    slug: "adguard",
    loadIcon: () => import("@icongo/vl/esm/VLAdguard").then((module) => module.VLAdguard),
  },
  {
    prefix: "lg",
    slug: "alibabacloud",
    loadIcon: () => import("@icongo/vl/esm/VLAlibabacloud").then((module) => module.VLAlibabacloud),
  },
  {
    prefix: "lg",
    slug: "alpinelinux",
    loadIcon: () => import("@icongo/vl/esm/VLAlpinelinux").then((module) => module.VLAlpinelinux),
  },
  {
    prefix: "lg",
    slug: "android",
    loadIcon: () => import("@icongo/vl/esm/VLAndroid").then((module) => module.VLAndroid),
  },
  {
    prefix: "lg",
    slug: "archlinux",
    loadIcon: () => import("@icongo/vl/esm/VLArchlinux").then((module) => module.VLArchlinux),
  },
  {
    prefix: "lg",
    slug: "baidu",
    loadIcon: () => import("@icongo/vl/esm/VLBaidu").then((module) => module.VLBaidu),
  },
  {
    prefix: "lg",
    slug: "bitbucket",
    loadIcon: () => import("@icongo/lg/esm/LGBitbucket").then((module) => module.LGBitbucket),
  },
  {
    prefix: "lg",
    slug: "browserstack",
    loadIcon: () => import("@icongo/lg/esm/LGBrowserstack").then((module) => module.LGBrowserstack),
  },
  {
    prefix: "lg",
    slug: "centos",
    loadIcon: () => import("@icongo/vl/esm/VLCentos").then((module) => module.VLCentos),
  },
  {
    prefix: "lg",
    slug: "cloudboostio",
    loadIcon: () => import("@icongo/vl/esm/VLCloudboostio").then((module) => module.VLCloudboostio),
  },
  {
    prefix: "lg",
    slug: "debian",
    loadIcon: () => import("@icongo/vl/esm/VLDebian").then((module) => module.VLDebian),
  },
  {
    prefix: "lg",
    slug: "email",
    loadIcon: loadCustomSvgIcon("email", "SXEmail"),
  },
  {
    prefix: "lg",
    slug: "elsevier",
    loadIcon: () => import("@icongo/vl/esm/VLElsevier").then((module) => module.VLElsevier),
  },
  {
    prefix: "lg",
    slug: "fedora",
    loadIcon: () => import("@icongo/vl/esm/VLGetfedora").then((module) => module.VLGetfedora),
  },
  {
    prefix: "lg",
    slug: "googledrive",
    loadIcon: () => import("@icongo/vl/esm/VLGoogleDrive").then((module) => module.VLGoogleDrive),
  },
  {
    prefix: "lg",
    slug: "googlephotos",
    loadIcon: () => import("@icongo/lg/esm/LGGooglePhotos").then((module) => module.LGGooglePhotos),
  },
  {
    prefix: "lg",
    slug: "huawei",
    loadIcon: () => import("@icongo/vl/esm/VLHuawei").then((module) => module.VLHuawei),
  },
  {
    prefix: "lg",
    slug: "intel",
    loadIcon: () => import("@icongo/vl/esm/VLIntel").then((module) => module.VLIntel),
  },
  {
    prefix: "lg",
    slug: "kubernetes",
    loadIcon: () => import("@icongo/vl/esm/VLKubernetes").then((module) => module.VLKubernetes),
  },
  {
    prefix: "lg",
    slug: "linux",
    loadIcon: () => import("@icongo/vl/esm/VLLinux").then((module) => module.VLLinux),
  },
  {
    prefix: "lg",
    slug: "manjaro",
    loadIcon: () => import("@icongo/lg/esm/LGManjaro").then((module) => module.LGManjaro),
  },
  {
    prefix: "custom",
    slug: "giffgaff",
    loadIcon: loadCustomSvgIcon("giffgaff", "SXGiffgaff"),
  },
  {
    prefix: "lg",
    slug: "microsoft",
    loadIcon: () => import("@icongo/vl/esm/VLMicrosoft").then((module) => module.VLMicrosoft),
  },
  {
    prefix: "custom",
    slug: "microsoft365",
    loadIcon: loadCustomSvgIcon("microsoft365", "SXMicrosoft365"),
  },
  {
    prefix: "lg",
    slug: "azure",
    loadIcon: () => import("@icongo/lg/esm/LGMicrosoftAzure").then((module) => module.LGMicrosoftAzure),
  },
  {
    prefix: "lg",
    slug: "windows",
    loadIcon: () => import("@icongo/lg/esm/LGMicrosoftWindows").then((module) => module.LGMicrosoftWindows),
  },
  {
    prefix: "lg",
    slug: "onedrive",
    loadIcon: () => import("@icongo/lg/esm/LGMicrosoftOnedrive").then((module) => module.LGMicrosoftOnedrive),
  },
  {
    prefix: "lg",
    slug: "mongodb",
    loadIcon: () => import("@icongo/lg/esm/LGMongodbIcon").then((module) => module.LGMongodbIcon),
  },
  {
    prefix: "lg",
    slug: "nextcloud",
    loadIcon: () => import("@icongo/vl/esm/VLNextcloud").then((module) => module.VLNextcloud),
  },
  {
    prefix: "lg",
    slug: "nvidia",
    loadIcon: () => import("@icongo/vl/esm/VLNvidia").then((module) => module.VLNvidia),
  },
  {
    prefix: "lg",
    slug: "raspberrypi",
    loadIcon: () => import("@icongo/vl/esm/VLRaspberrypi").then((module) => module.VLRaspberrypi),
  },
  {
    prefix: "lg",
    slug: "server",
    loadIcon: loadCustomSvgIcon("server", "SXServer"),
  },
  {
    prefix: "custom",
    slug: "simcard",
    loadIcon: loadCustomSvgIcon("simcard", "SXSimCard"),
  },
  {
    prefix: "lg",
    slug: "shopify",
    loadIcon: () => import("@icongo/vl/esm/VLShopify").then((module) => module.VLShopify),
  },
  {
    prefix: "lg",
    slug: "stackoverflow",
    loadIcon: () => import("@icongo/lg/esm/LGStackoverflowIcon").then((module) => module.LGStackoverflowIcon),
  },
  {
    prefix: "lg",
    slug: "supabase",
    loadIcon: () => import("@icongo/vl/esm/VLSupabase").then((module) => module.VLSupabase),
  },
  {
    prefix: "lg",
    slug: "termius",
    loadIcon: () => import("@icongo/lg/esm/LGTerminal").then((module) => module.LGTerminal),
  },
  {
    prefix: "lg",
    slug: "unity",
    loadIcon: () => import("@icongo/lg/esm/LGUnity").then((module) => module.LGUnity),
  },
  {
    prefix: "lg",
    slug: "ubuntu",
    loadIcon: () => import("@icongo/vl/esm/VLUbuntu").then((module) => module.VLUbuntu),
  },
  {
    prefix: "lg",
    slug: "visualstudio",
    loadIcon: () => import("@icongo/lg/esm/LGVisualStudio").then((module) => module.LGVisualStudio),
  },
  {
    prefix: "lg",
    slug: "vpn",
    loadIcon: loadCustomSvgIcon("vpn", "SXVpn"),
  },
  {
    prefix: "custom",
    slug: "clash",
    loadIcon: loadCustomSvgIcon("clash", "SXClash"),
  },
  {
    prefix: "custom",
    slug: "v2ray",
    loadIcon: loadCustomSvgIcon("v2ray", "SXV2ray"),
  },
  {
    prefix: "lg",
    slug: "whatsapp",
    loadIcon: () => import("@icongo/lg/esm/LGWhatsappIcon").then((module) => module.LGWhatsappIcon),
  },
  {
    prefix: "lg",
    slug: "wordpress",
    loadIcon: () => import("@icongo/lg/esm/LGWordpressIconAlt").then((module) => module.LGWordpressIconAlt),
  },
  {
    prefix: "lg",
    slug: "onepassword",
    loadIcon: () => import("@icongo/vl/esm/VL1Password").then((module) => module.VL1Password),
  },
  {
    prefix: "lg",
    slug: "adidas",
    loadIcon: () => import("@icongo/vl/esm/VLAdidas").then((module) => module.VLAdidas),
  },
  {
    prefix: "custom",
    slug: "adobecreativecloud",
    loadIcon: loadCustomSvgIcon("adobecreativecloud", "SXAdobeCreativeCloud"),
  },
  {
    prefix: "lg",
    slug: "adobeaftereffects",
    loadIcon: () => import("@icongo/lg/esm/LGAdobeAfterEffects").then((module) => module.LGAdobeAfterEffects),
  },
  {
    prefix: "lg",
    slug: "adobeanimate",
    loadIcon: () => import("@icongo/lg/esm/LGAdobeAnimate").then((module) => module.LGAdobeAnimate),
  },
  {
    prefix: "lg",
    slug: "adobedreamweaver",
    loadIcon: () => import("@icongo/lg/esm/LGAdobeDreamweaver").then((module) => module.LGAdobeDreamweaver),
  },
  {
    prefix: "lg",
    slug: "adobeillustrator",
    loadIcon: () => import("@icongo/lg/esm/LGAdobeIllustrator").then((module) => module.LGAdobeIllustrator),
  },
  {
    prefix: "lg",
    slug: "adobeincopy",
    loadIcon: () => import("@icongo/lg/esm/LGAdobeIncopy").then((module) => module.LGAdobeIncopy),
  },
  {
    prefix: "lg",
    slug: "adobeindesign",
    loadIcon: () => import("@icongo/lg/esm/LGAdobeIndesign").then((module) => module.LGAdobeIndesign),
  },
  {
    prefix: "lg",
    slug: "adobelightroom",
    loadIcon: () => import("@icongo/lg/esm/LGAdobeLightroom").then((module) => module.LGAdobeLightroom),
  },
  {
    prefix: "lg",
    slug: "adobephotoshop",
    loadIcon: () => import("@icongo/lg/esm/LGAdobePhotoshop").then((module) => module.LGAdobePhotoshop),
  },
  {
    prefix: "lg",
    slug: "adobepremiere",
    loadIcon: () => import("@icongo/lg/esm/LGAdobePremiere").then((module) => module.LGAdobePremiere),
  },
  {
    prefix: "lg",
    slug: "adobexd",
    loadIcon: () => import("@icongo/lg/esm/LGAdobeXd").then((module) => module.LGAdobeXd),
  },
  {
    prefix: "lg",
    slug: "airbnb",
    loadIcon: () => import("@icongo/lg/esm/LGAirbnbIcon").then((module) => module.LGAirbnbIcon),
  },
  {
    prefix: "lg",
    slug: "behance",
    loadIcon: () => import("@icongo/vl/esm/VLBehance").then((module) => module.VLBehance),
  },
  {
    prefix: "custom",
    slug: "bitwarden",
    loadIcon: loadCustomSvgIcon("bitwarden", "SXBitwarden"),
  },
  {
    prefix: "lg",
    slug: "box",
    loadIcon: () => import("@icongo/lg/esm/LGBox").then((module) => module.LGBox),
  },
  {
    prefix: "custom",
    slug: "baidunetdisk",
    loadIcon: loadCustomSvgIcon("baidunetdisk", "SXBaiduNetdisk"),
  },
  {
    prefix: "lg",
    slug: "canva",
    loadIcon: () => import("@icongo/vl/esm/VLCanva").then((module) => module.VLCanva),
  },
  {
    prefix: "custom",
    slug: "coinbase",
    loadIcon: loadCustomSvgIcon("coinbase", "SXCoinbase"),
  },
  {
    prefix: "lg",
    slug: "dropbox",
    loadIcon: () => import("@icongo/lg/esm/LGDropbox").then((module) => module.LGDropbox),
  },
  {
    prefix: "custom",
    slug: "quarkdrive",
    loadIcon: loadCustomSvgIcon("quarkdrive", "SXQuarkDrive"),
  },
  {
    prefix: "custom",
    slug: "aliyundrive",
    loadIcon: loadCustomSvgIcon("aliyundrive", "SXAliyunDrive"),
  },
  {
    prefix: "custom",
    slug: "drive115",
    loadIcon: loadCustomSvgIcon("drive115", "SXDrive115"),
  },
  {
    prefix: "custom",
    slug: "drive123",
    loadIcon: loadCustomSvgIcon("drive123", "SXDrive123"),
  },
  {
    prefix: "custom",
    slug: "weiyun",
    loadIcon: loadCustomSvgIcon("weiyun", "SXWeiyun"),
  },
  {
    prefix: "custom",
    slug: "xiaomicloud",
    loadIcon: loadCustomSvgIcon("xiaomicloud", "SXXiaomiCloud"),
  },
  {
    prefix: "custom",
    slug: "xunlei",
    loadIcon: loadCustomSvgIcon("xunlei", "SXXunlei"),
  },
  {
    prefix: "custom",
    slug: "pikpak",
    loadIcon: loadCustomSvgIcon("pikpak", "SXPikpak"),
  },
  {
    prefix: "lg",
    slug: "evernote",
    loadIcon: () => import("@icongo/vl/esm/VLEvernote").then((module) => module.VLEvernote),
  },
  {
    prefix: "lg",
    slug: "expressvpn",
    loadIcon: () => import("@icongo/si/esm/SIExpressvpn").then((module) => module.SIExpressvpn),
  },
  {
    prefix: "lg",
    slug: "feedly",
    loadIcon: () => import("@icongo/vl/esm/VLFeedly").then((module) => module.VLFeedly),
  },
  {
    prefix: "custom",
    slug: "feishu",
    loadIcon: loadCustomSvgIcon("feishu", "SXFeishu"),
  },
  {
    prefix: "custom",
    slug: "feiniu",
    loadIcon: loadCustomSvgIcon("feiniu", "SXFeiniu"),
  },
  {
    prefix: "lg",
    slug: "fontawesome",
    loadIcon: () => import("@icongo/lg/esm/LGFontAwesome").then((module) => module.LGFontAwesome),
  },
  {
    prefix: "custom",
    slug: "camscanner",
    loadIcon: loadCustomSvgIcon("camscanner", "SXCamscanner"),
  },
  {
    prefix: "custom",
    slug: "godaddy",
    loadIcon: loadCustomSvgIcon("godaddy", "SXGodaddy"),
  },
  {
    prefix: "lg",
    slug: "googleone",
    loadIcon: () => import("@icongo/lg/esm/LGGoogleOne").then((module) => module.LGGoogleOne),
  },
  {
    prefix: "lg",
    slug: "grammarly",
    loadIcon: () => import("@icongo/lg/esm/LGGrammarlyIcon").then((module) => module.LGGrammarlyIcon),
  },
  {
    prefix: "custom",
    slug: "wpsoffice",
    loadIcon: loadCustomSvgIcon("wpsoffice", "SXWpsOffice"),
  },
  {
    prefix: "custom",
    slug: "yuque",
    loadIcon: loadCustomSvgIcon("yuque", "SXYuque"),
  },
  {
    prefix: "custom",
    slug: "ticktick",
    loadIcon: loadCustomSvgIcon("ticktick", "SXTickTick"),
  },
  {
    prefix: "custom",
    slug: "hellobike",
    loadIcon: loadCustomSvgIcon("hellobike", "SXHelloBike"),
  },
  {
    prefix: "custom",
    slug: "china12306",
    loadIcon: loadCustomSvgIcon("china12306", "SXChina12306"),
  },
  {
    prefix: "lg",
    slug: "udemy",
    loadIcon: () => import("@icongo/lg/esm/LGUdemyIcon").then((module) => module.LGUdemyIcon),
  },
  {
    prefix: "custom",
    slug: "leetcode",
    loadIcon: loadCustomSvgIcon("leetcode", "SXLeetcode"),
  },
  {
    prefix: "lg",
    slug: "jetbrains",
    loadIcon: () => import("@icongo/lg/esm/LGJetbrainsIcon").then((module) => module.LGJetbrainsIcon),
  },
  {
    prefix: "custom",
    slug: "juejin",
    loadIcon: loadCustomSvgIcon("juejin", "SXJuejin"),
  },
  {
    prefix: "custom",
    slug: "csdn",
    loadIcon: loadCustomSvgIcon("csdn", "SXCsdn"),
  },
  {
    prefix: "lg",
    slug: "intellijidea",
    loadIcon: () => import("@icongo/lg/esm/LGIntellijIdea").then((module) => module.LGIntellijIdea),
  },
  {
    prefix: "lg",
    slug: "lastpass",
    loadIcon: () => import("@icongo/vl/esm/VLLastpass").then((module) => module.VLLastpass),
  },
  {
    prefix: "lg",
    slug: "linkedin",
    loadIcon: () => import("@icongo/lg/esm/LGLinkedinIcon").then((module) => module.LGLinkedinIcon),
  },
  {
    prefix: "lg",
    slug: "medium",
    loadIcon: () => import("@icongo/lg/esm/LGMediumIcon").then((module) => module.LGMediumIcon),
  },
  {
    prefix: "custom",
    slug: "meituan",
    loadIcon: loadCustomSvgIcon("meituan", "SXMeituan"),
  },
  {
    prefix: "lg",
    slug: "meta",
    loadIcon: () => import("@icongo/si/esm/SIMeta").then((module) => module.SIMeta),
  },
  {
    prefix: "lg",
    slug: "nytimes",
    loadIcon: () => import("@icongo/vl/esm/VLNytimes").then((module) => module.VLNytimes),
  },
  {
    prefix: "lg",
    slug: "patreon",
    loadIcon: () => import("@icongo/lg/esm/LGPatreon").then((module) => module.LGPatreon),
  },
  {
    prefix: "custom",
    slug: "afdian",
    loadIcon: loadCustomSvgIcon("afdian", "SXAfdian"),
  },
  {
    prefix: "custom",
    slug: "onlyfans",
    loadIcon: loadCustomSvgIcon("onlyfans", "SXOnlyFans"),
  },
  {
    prefix: "lg",
    slug: "pixiv",
    loadIcon: () => import("@icongo/si/esm/SIPixiv").then((module) => module.SIPixiv),
  },
  {
    prefix: "lg",
    slug: "pinterest",
    loadIcon: () => import("@icongo/lg/esm/LGPinterest").then((module) => module.LGPinterest),
  },
  {
    prefix: "lg",
    slug: "proton",
    loadIcon: () => import("@icongo/vl/esm/VLProtonmail").then((module) => module.VLProtonmail),
  },
  {
    prefix: "lg",
    slug: "revolut",
    loadIcon: () => import("@icongo/vl/esm/VLRevolut").then((module) => module.VLRevolut),
  },
  {
    prefix: "lg",
    slug: "twitter",
    loadIcon: () => import("@icongo/lg/esm/LGTwitter").then((module) => module.LGTwitter),
  },
  {
    prefix: "custom",
    slug: "x",
    loadIcon: loadCustomSvgIcon("x", "SXX"),
  },
  {
    prefix: "lg",
    slug: "uber",
    loadIcon: () => import("@icongo/si/esm/SIUber").then((module) => module.SIUber),
  },
  {
    prefix: "lg",
    slug: "walmart",
    loadIcon: () => import("@icongo/vl/esm/VLWalmart").then((module) => module.VLWalmart),
  },
  {
    prefix: "custom",
    slug: "samsclub",
    loadIcon: loadCustomSvgIcon("samsclub", "SXSamsClub"),
  },
  {
    prefix: "custom",
    slug: "pinduoduo",
    loadIcon: loadCustomSvgIcon("pinduoduo", "SXPinduoduo"),
  },
  {
    prefix: "custom",
    slug: "taobao",
    loadIcon: loadCustomSvgIcon("taobao", "SXTaobao"),
  },
  {
    prefix: "custom",
    slug: "jd",
    loadIcon: loadCustomSvgIcon("jd", "SXJd"),
  },
  {
    prefix: "custom",
    slug: "vip88",
    loadIcon: loadCustomSvgIcon("vip88", "SXVip88"),
  },
  {
    prefix: "lg",
    slug: "wise",
    loadIcon: () => import("@icongo/vl/esm/VLTransferwise").then((module) => module.VLTransferwise),
  },
  {
    prefix: "lg",
    slug: "xbox",
    loadIcon: () => import("@icongo/vl/esm/VLXbox").then((module) => module.VLXbox),
  },
  {
    prefix: "lg",
    slug: "zoom",
    loadIcon: () => import("@icongo/vl/esm/VLZoomus").then((module) => module.VLZoomus),
  },
  {
    prefix: "custom",
    slug: "zhihu",
    loadIcon: loadCustomSvgIcon("zhihu", "SXZhihu"),
  },
  {
    prefix: "custom",
    slug: "icloud",
    loadIcon: loadCustomSvgIcon("icloud", "SXIcloud"),
  },
  {
    prefix: "lg",
    slug: "audible",
    loadIcon: () => import("@icongo/si/esm/SIAudible").then((module) => module.SIAudible),
  },
  {
    prefix: "lg",
    slug: "bitly",
    loadIcon: () => import("@icongo/si/esm/SIBitly").then((module) => module.SIBitly),
  },
  {
    prefix: "lg",
    slug: "cashapp",
    loadIcon: () => import("@icongo/si/esm/SICashapp").then((module) => module.SICashapp),
  },
  {
    prefix: "lg",
    slug: "chase",
    loadIcon: () => import("@icongo/si/esm/SIChase").then((module) => module.SIChase),
  },
  {
    prefix: "lg",
    slug: "dhl",
    loadIcon: () => import("@icongo/si/esm/SIDhl").then((module) => module.SIDhl),
  },
  {
    prefix: "lg",
    slug: "duolingo",
    loadIcon: () => import("@icongo/si/esm/SIDuolingo").then((module) => module.SIDuolingo),
  },
  {
    prefix: "lg",
    slug: "epicgames",
    loadIcon: () => import("@icongo/si/esm/SIEpicgames").then((module) => module.SIEpicgames),
  },
  {
    prefix: "lg",
    slug: "hbomax",
    loadIcon: () => import("@icongo/si/esm/SIHbo").then((module) => module.SIHbo),
  },
  {
    prefix: "lg",
    slug: "max",
    loadIcon: () => import("@icongo/si/esm/SIMax").then((module) => module.SIMax),
  },
  {
    prefix: "lg",
    slug: "hulu",
    loadIcon: () => import("@icongo/si/esm/SIHulu").then((module) => module.SIHulu),
  },
  {
    prefix: "lg",
    slug: "instapaper",
    loadIcon: () => import("@icongo/si/esm/SIInstapaper").then((module) => module.SIInstapaper),
  },
  {
    prefix: "lg",
    slug: "mega",
    loadIcon: () => import("@icongo/si/esm/SIMega").then((module) => module.SIMega),
  },
  {
    prefix: "custom",
    slug: "nintendo",
    loadIcon: loadCustomSvgIcon("nintendo", "SXNintendo"),
  },
  {
    prefix: "custom",
    slug: "chinamobile",
    loadIcon: loadCustomSvgIcon("chinamobile", "SXChinaMobile"),
  },
  {
    prefix: "custom",
    slug: "chinatelecom",
    loadIcon: loadCustomSvgIcon("chinatelecom", "SXChinaTelecom"),
  },
  {
    prefix: "custom",
    slug: "chinaunicom",
    loadIcon: loadCustomSvgIcon("chinaunicom", "SXChinaUnicom"),
  },
  {
    prefix: "custom",
    slug: "chinabroadcast",
    loadIcon: loadCustomSvgIcon("chinabroadcast", "SXChinaBroadcast"),
  },
  {
    prefix: "lg",
    slug: "nordvpn",
    loadIcon: () => import("@icongo/si/esm/SINordvpn").then((module) => module.SINordvpn),
  },
  {
    prefix: "lg",
    slug: "notion",
    loadIcon: () => import("@icongo/si/esm/SINotion").then((module) => module.SINotion),
  },
  {
    prefix: "lg",
    slug: "obsidian",
    loadIcon: () => import("@icongo/si/esm/SIObsidian").then((module) => module.SIObsidian),
  },
  {
    prefix: "lg",
    slug: "openvpn",
    loadIcon: () => import("@icongo/si/esm/SIOpenvpn").then((module) => module.SIOpenvpn),
  },
  {
    prefix: "lg",
    slug: "playstation",
    loadIcon: () => import("@icongo/si/esm/SIPlaystation").then((module) => module.SIPlaystation),
  },
  {
    prefix: "lg",
    slug: "tmobile",
    loadIcon: () => import("@icongo/si/esm/SITmobile").then((module) => module.SITmobile),
  },
  {
    prefix: "lg",
    slug: "ubisoft",
    loadIcon: () => import("@icongo/si/esm/SIUbisoft").then((module) => module.SIUbisoft),
  },
  {
    prefix: "custom",
    slug: "neteaseuu",
    loadIcon: loadCustomSvgIcon("neteaseuu", "SXNeteaseUu"),
  },
  {
    prefix: "lg",
    slug: "unsplash",
    loadIcon: () => import("@icongo/si/esm/SIUnsplash").then((module) => module.SIUnsplash),
  },
  {
    prefix: "lg",
    slug: "crunchyroll",
    loadIcon: () => import("@icongo/si/esm/SICrunchyroll").then((module) => module.SICrunchyroll),
  },
  {
    prefix: "custom",
    slug: "dlsite",
    loadIcon: loadCustomSvgIcon("dlsite", "SXDlsite"),
  },
  {
    prefix: "lg",
    slug: "niconico",
    loadIcon: () => import("@icongo/si/esm/SINiconico").then((module) => module.SINiconico),
  },
  {
    prefix: "custom",
    slug: "bilibili",
    loadIcon: loadCustomSvgIcon("bilibili", "SXBilibili"),
  },
  {
    prefix: "custom",
    slug: "iqiyi",
    loadIcon: loadCustomSvgIcon("iqiyi", "SXIqiyi"),
  },
  {
    prefix: "lg",
    slug: "kuaishou",
    loadIcon: () => import("@icongo/si/esm/SIKuaishou").then((module) => module.SIKuaishou),
  },
  {
    prefix: "custom",
    slug: "neteasecloudmusic",
    loadIcon: loadCustomSvgIcon("neteasecloudmusic", "SXNeteaseCloudMusic"),
  },
  {
    prefix: "lg",
    slug: "youtube",
    loadIcon: () => import("@icongo/lg/esm/LGYoutubeIcon").then((module) => module.LGYoutubeIcon),
  },
  {
    prefix: "lg",
    slug: "spotify",
    loadIcon: () => import("@icongo/lg/esm/LGSpotifyIcon").then((module) => module.LGSpotifyIcon),
  },
  {
    prefix: "lg",
    slug: "netflix",
    loadIcon: () => import("@icongo/vl/esm/VLNetflix").then((module) => module.VLNetflix),
  },
  {
    prefix: "lg",
    slug: "amazonprimevideo",
    loadIcon: () => import("@icongo/vl/esm/VLCprime").then((module) => module.VLCprime),
  },
  {
    prefix: "lg",
    slug: "appletvplus",
    loadIcon: () => import("@icongo/lg/esm/LGApple").then((module) => module.LGApple),
  },
  {
    prefix: "lg",
    slug: "applemusic",
    loadIcon: () => import("@icongo/lg/esm/LGApple").then((module) => module.LGApple),
  },
  {
    prefix: "custom",
    slug: "youtubemusic",
    loadIcon: loadCustomSvgIcon("youtubemusic", "SXYoutubeMusic"),
  },
  {
    prefix: "custom",
    slug: "qqmusic",
    loadIcon: loadCustomSvgIcon("qqmusic", "SXQqMusic"),
  },
  {
    prefix: "custom",
    slug: "qq",
    loadIcon: loadCustomSvgIcon("qq", "SXQq"),
  },
  {
    prefix: "custom",
    slug: "kugoumusic",
    loadIcon: loadCustomSvgIcon("kugoumusic", "SXKugouMusic"),
  },
  {
    prefix: "custom",
    slug: "ximalaya",
    loadIcon: loadCustomSvgIcon("ximalaya", "SXXimalaya"),
  },
  {
    prefix: "lg",
    slug: "disneyplus",
    loadIcon: () => import("@icongo/vl/esm/VLDisney").then((module) => module.VLDisney),
  },
  {
    prefix: "lg",
    slug: "twitch",
    loadIcon: () => import("@icongo/lg/esm/LGTwitch").then((module) => module.LGTwitch),
  },
  {
    prefix: "lg",
    slug: "vimeo",
    loadIcon: () => import("@icongo/lg/esm/LGVimeoIcon").then((module) => module.LGVimeoIcon),
  },
  {
    prefix: "lg",
    slug: "soundcloud",
    loadIcon: () => import("@icongo/lg/esm/LGSoundcloud").then((module) => module.LGSoundcloud),
  },
  {
    prefix: "lg",
    slug: "plex",
    loadIcon: () => import("@icongo/vl/esm/VLPlextv").then((module) => module.VLPlextv),
  },
  {
    prefix: "custom",
    slug: "infuse",
    loadIcon: loadCustomSvgIcon("infuse", "SXInfuse"),
  },
  {
    prefix: "custom",
    slug: "emby",
    loadIcon: loadCustomSvgIcon("emby", "SXEmby"),
  },
  {
    prefix: "custom",
    slug: "jellyfin",
    loadIcon: loadCustomSvgIcon("jellyfin", "SXJellyfin"),
  },
  {
    prefix: "lg",
    slug: "vlc",
    loadIcon: () => import("@icongo/vl/esm/VLVideolanVlc").then((module) => module.VLVideolanVlc),
  },
  {
    prefix: "custom",
    slug: "tencentvideo",
    loadIcon: loadCustomSvgIcon("tencentvideo", "SXTencentVideo"),
  },
  {
    prefix: "lg",
    slug: "weibo",
    loadIcon: () => import("@icongo/vl/esm/VLWeibo").then((module) => module.VLWeibo),
  },
  {
    prefix: "lg",
    slug: "apple",
    loadIcon: () => import("@icongo/lg/esm/LGAppleAppStore").then((module) => module.LGAppleAppStore),
  },
  {
    prefix: "lg",
    slug: "tiktok",
    loadIcon: () => import("@icongo/lg/esm/LGTiktokIcon").then((module) => module.LGTiktokIcon),
  },
  {
    prefix: "lg",
    slug: "discord",
    loadIcon: () => import("@icongo/lg/esm/LGDiscordIcon").then((module) => module.LGDiscordIcon),
  },
  {
    prefix: "lg",
    slug: "line",
    loadIcon: () => import("@icongo/vl/esm/VLLine").then((module) => module.VLLine),
  },
  {
    prefix: "lg",
    slug: "telegram",
    loadIcon: () => import("@icongo/lg/esm/LGTelegram").then((module) => module.LGTelegram),
  },
  {
    prefix: "lg",
    slug: "facebook",
    loadIcon: () => import("@icongo/lg/esm/LGFacebook").then((module) => module.LGFacebook),
  },
  {
    prefix: "lg",
    slug: "instagram",
    loadIcon: () => import("@icongo/lg/esm/LGInstagramIcon").then((module) => module.LGInstagramIcon),
  },
  {
    prefix: "lg",
    slug: "bitcoin",
    loadIcon: () => import("@icongo/lg/esm/LGBitcoin").then((module) => module.LGBitcoin),
  },
  {
    prefix: "custom",
    slug: "okx",
    loadIcon: loadCustomSvgIcon("okx", "SXOkx"),
  },
  {
    prefix: "custom",
    slug: "binance",
    loadIcon: loadCustomSvgIcon("binance", "SXBinance"),
  },
  {
    prefix: "lg",
    slug: "ethereum",
    loadIcon: () => import("@icongo/lg/esm/LGEthereum").then((module) => module.LGEthereum),
  },
  {
    prefix: "lg",
    slug: "monero",
    loadIcon: () => import("@icongo/lg/esm/LGMonero").then((module) => module.LGMonero),
  },
  {
    prefix: "lg",
    slug: "kraken",
    loadIcon: () => import("@icongo/lg/esm/LGKraken").then((module) => module.LGKraken),
  },
  {
    prefix: "custom",
    slug: "game",
    loadIcon: loadCustomSvgIcon("game", "SXGame"),
  },
  {
    prefix: "bl",
    slug: "bankofamerica",
    loadIcon: () => import("@icongo/bl/esm/BLBankofamericaRect").then((module) => module.BLBankofamericaRect),
  },
  {
    prefix: "bl",
    slug: "hsbc",
    loadIcon: () => import("@icongo/vl/esm/VLHsbc").then((module) => module.VLHsbc),
  },
  {
    prefix: "bl",
    slug: "n26",
    loadIcon: () => import("@icongo/vl/esm/VLN26").then((module) => module.VLN26),
  },
  {
    prefix: "custom",
    slug: "zabank",
    loadIcon: loadCustomSvgIcon("zabank", "SXZaBank"),
  },
  {
    prefix: "bl",
    slug: "bankofchina",
    loadIcon: () => import("@icongo/bl/esm/BLBocRect").then((module) => module.BLBocRect),
  },
  {
    prefix: "bl",
    slug: "icbc",
    loadIcon: () => import("@icongo/bl/esm/BLIcbcRect").then((module) => module.BLIcbcRect),
  },
  {
    prefix: "bl",
    slug: "ccb",
    loadIcon: () => import("@icongo/bl/esm/BLCcbRect").then((module) => module.BLCcbRect),
  },
  {
    prefix: "bl",
    slug: "abc",
    loadIcon: () => import("@icongo/bl/esm/BLAbchinaRect").then((module) => module.BLAbchinaRect),
  },
  {
    prefix: "bl",
    slug: "cmbchina",
    loadIcon: () => import("@icongo/bl/esm/BLCmbchinaRect").then((module) => module.BLCmbchinaRect),
  },
  {
    prefix: "bl",
    slug: "cmbc",
    loadIcon: () => import("@icongo/bl/esm/BLCmbcRect").then((module) => module.BLCmbcRect),
  },
  {
    prefix: "bl",
    slug: "citicbank",
    loadIcon: () => import("@icongo/bl/esm/BLCiticbankRect").then((module) => module.BLCiticbankRect),
  },
  {
    prefix: "bl",
    slug: "bankcomm",
    loadIcon: () => import("@icongo/bl/esm/BLBankcommRect").then((module) => module.BLBankcommRect),
  },
  {
    prefix: "bl",
    slug: "cebbank",
    loadIcon: () => import("@icongo/bl/esm/BLCebbankRect").then((module) => module.BLCebbankRect),
  },
  {
    prefix: "bl",
    slug: "cib",
    loadIcon: () => import("@icongo/bl/esm/BLCibRect").then((module) => module.BLCibRect),
  },
]
