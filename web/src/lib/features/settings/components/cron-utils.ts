export type ScheduleMode = 'seconds' | 'minutes' | 'hours' | 'daily' | 'monthly'

export type ScheduleConfig = {
  mode: ScheduleMode
  interval: number
  atMinute: number
  atHour: number
  atDay: number
}

export const scheduleModeOptions: { value: ScheduleMode; label: string }[] = [
  { value: 'seconds', label: 'second(s)' },
  { value: 'minutes', label: 'minute(s)' },
  { value: 'hours', label: 'hour(s)' },
  { value: 'daily', label: 'day(s)' },
  { value: 'monthly', label: 'month(s)' },
]

export function defaultScheduleConfig(): ScheduleConfig {
  return {
    mode: 'daily',
    interval: 1,
    atMinute: 0,
    atHour: 2,
    atDay: 1,
  }
}

export function getScheduleIntervalMax(mode: ScheduleMode) {
  switch (mode) {
    case 'seconds':
    case 'minutes':
      return 59
    case 'hours':
      return 23
    case 'daily':
      return 365
    case 'monthly':
      return 12
  }
}

export function clampScheduleNumber(val: string, min: number, max: number, fallback: number) {
  const parsed = Number.parseInt(val, 10)
  if (!Number.isFinite(parsed)) {
    return fallback
  }
  return Math.max(min, Math.min(max, parsed))
}

export function buildCronExpression(config: ScheduleConfig): string {
  const { mode, interval, atMinute, atHour, atDay } = config

  switch (mode) {
    case 'seconds':
      return interval === 1 ? '* * * * * *' : `*/${interval} * * * * *`
    case 'minutes':
      return interval === 1 ? '* * * * *' : `*/${interval} * * * *`
    case 'hours': {
      const h = interval === 1 ? '*' : `*/${interval}`
      return `${atMinute} ${h} * * *`
    }
    case 'daily': {
      const d = interval === 1 ? '*' : `*/${interval}`
      return `${atMinute} ${atHour} ${d} * *`
    }
    case 'monthly': {
      const m = interval === 1 ? '*' : `*/${interval}`
      return `${atMinute} ${atHour} ${atDay} ${m} *`
    }
  }
}

export function parseCronToConfig(cron: string): ScheduleConfig | null {
  const parts = cron.trim().split(/\s+/)

  if (parts.length === 6) {
    const [sec, min, hour, dom, mon] = parts
    if ((isStep(sec) || sec === '*') && min === '*' && hour === '*' && dom === '*' && mon === '*') {
      return {
        mode: 'seconds',
        interval: sec === '*' ? 1 : stepValue(sec),
        atMinute: 0,
        atHour: 0,
        atDay: 1,
      }
    }
    return null
  }

  if (parts.length !== 5) return null

  const [min, hour, dom, mon, dow] = parts

  // Every N minutes: */N * * * * or * * * * *
  if ((isStep(min) || min === '*') && hour === '*' && dom === '*' && mon === '*' && dow === '*') {
    return {
      mode: 'minutes',
      interval: min === '*' ? 1 : stepValue(min),
      atMinute: 0,
      atHour: 0,
      atDay: 1,
    }
  }

  // Every N hours at minute M: M */N * * * or M * * * *
  if (
    isNumeric(min) &&
    (isStep(hour) || hour === '*') &&
    dom === '*' &&
    mon === '*' &&
    dow === '*'
  ) {
    return {
      mode: 'hours',
      interval: hour === '*' ? 1 : stepValue(hour),
      atMinute: parseInt(min, 10),
      atHour: 0,
      atDay: 1,
    }
  }

  // Every N days at HH:MM: M H */N * * or M H * * *
  if (
    isNumeric(min) &&
    isNumeric(hour) &&
    (isStep(dom) || dom === '*') &&
    mon === '*' &&
    dow === '*'
  ) {
    return {
      mode: 'daily',
      interval: dom === '*' ? 1 : stepValue(dom),
      atMinute: parseInt(min, 10),
      atHour: parseInt(hour, 10),
      atDay: 1,
    }
  }

  // Every N months on day D at HH:MM: M H D */N * or M H D * *
  if (
    isNumeric(min) &&
    isNumeric(hour) &&
    isNumeric(dom) &&
    (isStep(mon) || mon === '*') &&
    dow === '*'
  ) {
    return {
      mode: 'monthly',
      interval: mon === '*' ? 1 : stepValue(mon),
      atMinute: parseInt(min, 10),
      atHour: parseInt(hour, 10),
      atDay: parseInt(dom, 10),
    }
  }

  return null
}

function isStep(field: string): boolean {
  return /^\*\/\d+$/.test(field)
}

function isNumeric(field: string): boolean {
  return /^\d+$/.test(field)
}

function stepValue(field: string): number {
  if (field === '*') return 1
  const match = field.match(/^\*\/(\d+)$/)
  return match ? parseInt(match[1], 10) : 1
}

export function getNextTriggerTimes(config: ScheduleConfig, count: number, from?: Date): Date[] {
  const now = from ?? new Date()
  const results: Date[] = []

  switch (config.mode) {
    case 'seconds': {
      const startSec = Math.floor(now.getTime() / 1000) + 1
      const rem = startSec % config.interval
      const first = rem === 0 ? startSec : startSec + (config.interval - rem)
      for (let i = 0; i < count; i++) {
        results.push(new Date((first + i * config.interval) * 1000))
      }
      break
    }

    case 'minutes': {
      const d = new Date(now)
      d.setSeconds(0, 0)
      d.setMinutes(d.getMinutes() + 1)
      if (config.interval > 1) {
        const totalMin = d.getHours() * 60 + d.getMinutes()
        const rem = totalMin % config.interval
        if (rem !== 0) d.setMinutes(d.getMinutes() + (config.interval - rem))
      }
      for (let i = 0; i < count; i++) {
        results.push(new Date(d))
        d.setMinutes(d.getMinutes() + config.interval)
      }
      break
    }

    case 'hours': {
      const d = new Date(now)
      d.setSeconds(0, 0)
      d.setMinutes(config.atMinute)
      if (d <= now) d.setHours(d.getHours() + 1)
      if (config.interval > 1) {
        const rem = d.getHours() % config.interval
        if (rem !== 0) d.setHours(d.getHours() + (config.interval - rem))
      }
      for (let i = 0; i < count; i++) {
        results.push(new Date(d))
        d.setHours(d.getHours() + config.interval)
      }
      break
    }

    case 'daily': {
      const d = new Date(now)
      d.setSeconds(0, 0)
      d.setMinutes(config.atMinute)
      d.setHours(config.atHour)
      if (d <= now) d.setDate(d.getDate() + 1)
      if (config.interval > 1) {
        const dayIdx = d.getDate() - 1
        const rem = dayIdx % config.interval
        if (rem !== 0) d.setDate(d.getDate() + (config.interval - rem))
      }
      for (let i = 0; i < count; i++) {
        results.push(new Date(d))
        d.setDate(d.getDate() + config.interval)
      }
      break
    }

    case 'monthly': {
      const d = new Date(now)
      d.setSeconds(0, 0)
      d.setMinutes(config.atMinute)
      d.setHours(config.atHour)
      d.setDate(Math.min(config.atDay, daysInMonth(d.getFullYear(), d.getMonth())))
      if (d <= now) d.setMonth(d.getMonth() + 1)
      if (config.interval > 1) {
        const rem = d.getMonth() % config.interval
        if (rem !== 0) d.setMonth(d.getMonth() + (config.interval - rem))
      }
      for (let i = 0; i < count; i++) {
        const maxDay = daysInMonth(d.getFullYear(), d.getMonth())
        const snapshot = new Date(d)
        snapshot.setDate(Math.min(config.atDay, maxDay))
        results.push(snapshot)
        d.setMonth(d.getMonth() + config.interval)
      }
      break
    }
  }

  return results
}

function daysInMonth(year: number, month: number): number {
  return new Date(year, month + 1, 0).getDate()
}

export function formatTriggerTime(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  const months = [
    'Jan',
    'Feb',
    'Mar',
    'Apr',
    'May',
    'Jun',
    'Jul',
    'Aug',
    'Sep',
    'Oct',
    'Nov',
    'Dec',
  ]
  return `${months[date.getMonth()]} ${date.getDate()}, ${date.getFullYear()}  ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}
