export type ToastVariant = 'success' | 'error' | 'info' | 'warning'

export type Toast = {
  id: string
  message: string
  variant: ToastVariant
  duration: number
  createdAt: number
}

let nextId = 0
let toasts = $state<Toast[]>([])

function add(message: string, variant: ToastVariant = 'info', duration = 3000): string {
  const id = String(++nextId)
  toasts = [...toasts, { id, message, variant, duration, createdAt: Date.now() }]
  return id
}

function dismiss(id: string) {
  toasts = toasts.filter((t) => t.id !== id)
}

function clear() {
  toasts = []
}

export const toastStore = {
  get toasts() {
    return toasts
  },
  add,
  dismiss,
  clear,
  success: (message: string, duration?: number) => add(message, 'success', duration),
  error: (message: string, duration?: number) => add(message, 'error', duration ?? 5000),
  info: (message: string, duration?: number) => add(message, 'info', duration),
  warning: (message: string, duration?: number) => add(message, 'warning', duration ?? 4000),
}
