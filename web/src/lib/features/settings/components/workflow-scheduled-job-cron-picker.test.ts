import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'

import { i18nStore } from '$lib/i18n/store.svelte'

import WorkflowScheduledJobCronPicker from './workflow-scheduled-job-cron-picker.svelte'

describe('WorkflowScheduledJobCronPicker', () => {
  afterEach(() => {
    cleanup()
    i18nStore.setLocale('en')
  })

  it('renders the selected schedule unit label instead of a leaked derived callback', () => {
    i18nStore.setLocale('zh')

    const { container } = render(WorkflowScheduledJobCronPicker)

    expect(container.textContent).toContain('每天')
    expect(container.textContent).not.toContain('=>')
    expect(container.textContent).not.toContain('m.t(')
  })
})
