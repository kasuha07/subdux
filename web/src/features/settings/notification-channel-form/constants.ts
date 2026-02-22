import type { ChannelType } from "./types"

export const WEBHOOK_HEADERS_PARSE_ERROR = "WEBHOOK_HEADERS_PARSE_ERROR"
export const PUSHOVER_SOUND_DEVICE_DEFAULT = "__device_default__"

export const CHANNEL_TYPE_OPTIONS: ChannelType[] = [
  "smtp",
  "resend",
  "telegram",
  "webhook",
  "pushdeer",
  "pushplus",
  "pushover",
  "gotify",
  "ntfy",
  "bark",
  "serverchan",
  "feishu",
  "wecom",
  "dingtalk",
  "napcat",
]

export const PUSHOVER_SOUND_OPTIONS: Array<{ value: string; i18nKey: string }> = [
  { value: PUSHOVER_SOUND_DEVICE_DEFAULT, i18nKey: "pushoverSoundOptionDeviceDefault" },
  { value: "pushover", i18nKey: "pushoverSoundOptionPushover" },
  { value: "vibrate", i18nKey: "pushoverSoundOptionVibrate" },
  { value: "none", i18nKey: "pushoverSoundOptionNone" },
  { value: "bike", i18nKey: "pushoverSoundOptionBike" },
  { value: "bugle", i18nKey: "pushoverSoundOptionBugle" },
  { value: "cashregister", i18nKey: "pushoverSoundOptionCashregister" },
  { value: "classical", i18nKey: "pushoverSoundOptionClassical" },
  { value: "cosmic", i18nKey: "pushoverSoundOptionCosmic" },
  { value: "falling", i18nKey: "pushoverSoundOptionFalling" },
  { value: "gamelan", i18nKey: "pushoverSoundOptionGamelan" },
  { value: "incoming", i18nKey: "pushoverSoundOptionIncoming" },
  { value: "intermission", i18nKey: "pushoverSoundOptionIntermission" },
  { value: "magic", i18nKey: "pushoverSoundOptionMagic" },
  { value: "mechanical", i18nKey: "pushoverSoundOptionMechanical" },
  { value: "pianobar", i18nKey: "pushoverSoundOptionPianobar" },
  { value: "siren", i18nKey: "pushoverSoundOptionSiren" },
  { value: "spacealarm", i18nKey: "pushoverSoundOptionSpacealarm" },
  { value: "tugboat", i18nKey: "pushoverSoundOptionTugboat" },
  { value: "alien", i18nKey: "pushoverSoundOptionAlien" },
  { value: "climb", i18nKey: "pushoverSoundOptionClimb" },
  { value: "persistent", i18nKey: "pushoverSoundOptionPersistent" },
  { value: "echo", i18nKey: "pushoverSoundOptionEcho" },
  { value: "updown", i18nKey: "pushoverSoundOptionUpdown" },
]
