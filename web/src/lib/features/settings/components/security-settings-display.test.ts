import { describe, expect, it } from 'vitest'

import { translate } from '$lib/i18n'

import {
  formatDisplayText,
  parseDeferredCapability,
  parseGitHubCredentialSource,
  parseGitHubProbeState,
  parseGitHubRepoAccess,
} from './security-settings-display'
import { configuredSecurity } from './security-settings.test-helpers'

describe('security display parsing', () => {
  it('maps deferred capability metadata to stable translated copy', () => {
    const display = parseDeferredCapability(configuredSecurity().deferred, 'github-device-flow')

    expect(display).not.toBeNull()
    expect(formatDisplayText(display!.title, translate.bind(null, 'zh'))).toBe('设备流程')
    expect(formatDisplayText(display!.summary, translate.bind(null, 'zh'))).toBe(
      '待 OAuth 应用接线完成后，可通过 GitHub 设备验证码流程授权 CLI。',
    )
  })

  it('parses GitHub probe status, source, and repo access at the display boundary', () => {
    const slot = configuredSecurity().github.effective

    expect(formatDisplayText(parseGitHubProbeState(slot), translate.bind(null, 'zh'))).toBe('可用')
    expect(
      formatDisplayText(parseGitHubCredentialSource(slot.source), translate.bind(null, 'zh')),
    ).toBe('从 gh 导入')
    expect(
      formatDisplayText(parseGitHubRepoAccess(slot.probe.repo_access), translate.bind(null, 'zh')),
    ).toBe('已授权')
  })
})
