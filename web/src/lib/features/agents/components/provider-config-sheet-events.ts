export function draftFieldValue(event: Event) {
  const target = event.currentTarget as HTMLInputElement | HTMLTextAreaElement
  return target.value
}
