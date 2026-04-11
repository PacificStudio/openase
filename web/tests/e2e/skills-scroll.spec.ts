import { measureNavigation } from './perf'
import { expect, test } from './fixtures'

test('skills page can scroll through long skill lists', async ({
  page,
  projectPath,
  request,
}, testInfo) => {
  for (let index = 0; index < 40; index += 1) {
    const skillName = `scroll-skill-${index.toString().padStart(2, '0')}`
    const response = await request.post('/api/v1/projects/project-e2e/skills', {
      data: {
        name: skillName,
        description: `Scroll regression fixture ${index}`,
        content: `# ${skillName}\n\nGenerated for scroll regression coverage.`,
        is_enabled: true,
      },
    })
    expect(response.ok()).toBeTruthy()
  }

  await measureNavigation({
    page,
    scenario: 'skills_scroll_page_ready',
    budgetMs: 800,
    ready: page.getByRole('heading', { name: 'Skills' }),
    testInfo,
    action: async () => {
      await page.goto(projectPath('skills'))
    },
  })

  const scrollContainer = page.getByTestId('route-scroll-container')
  const bottomSkill = page.getByText('deploy-openase').first()

  await expect
    .poll(async () => {
      return scrollContainer.evaluate((element) => element.scrollHeight > element.clientHeight)
    })
    .toBe(true)

  await bottomSkill.scrollIntoViewIfNeeded()

  await expect(bottomSkill).toBeInViewport()
  await expect
    .poll(async () => {
      return scrollContainer.evaluate((element) => element.scrollTop)
    })
    .toBeGreaterThan(0)
})
