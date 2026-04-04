import { measureNavigation } from './perf'
import { expect, test } from './fixtures'

const TODO_STATUS_ID = 'status-todo'
const REVIEW_STATUS_ID = 'status-review'
const DONE_STATUS_ID = 'status-done'

test('board columns fill the available height and scroll internally', async ({
  page,
  projectPath,
  request,
}, testInfo) => {
  await page.setViewportSize({ width: 1440, height: 900 })

  const response = await request.post('/api/v1/__e2e__/seed-board', {
    data: {
      counts_by_status_id: {
        [TODO_STATUS_ID]: 18,
        [REVIEW_STATUS_ID]: 0,
        [DONE_STATUS_ID]: 1,
      },
    },
  })
  expect(response.ok()).toBeTruthy()

  const boardViewButton = page.getByRole('button', { name: 'Board view' })
  const hideEmptyButton = page.getByRole('button', { name: 'Hide empty' })

  await measureNavigation({
    page,
    scenario: 'tickets_board_layout_ready',
    budgetMs: 800,
    ready: boardViewButton,
    testInfo,
    action: async () => {
      await page.goto(projectPath('tickets'))
    },
  })

  if (!(await hideEmptyButton.isVisible())) {
    await boardViewButton.click()
  }

  await expect(hideEmptyButton).toBeVisible()
  await hideEmptyButton.click()

  const todoList = page.getByRole('list', { name: 'Todo tickets' })
  const reviewList = page.getByRole('list', { name: 'In Review tickets' })

  await expect(todoList).toBeVisible()
  await expect(reviewList).toBeVisible()
  await expect(page.getByText('ASE-101')).toBeVisible()

  const [todoMetrics, reviewMetrics] = await Promise.all([
    todoList.evaluate((node) => {
      const element = node as HTMLElement
      return {
        clientHeight: element.clientHeight,
        scrollHeight: element.scrollHeight,
      }
    }),
    reviewList.evaluate((node) => {
      const element = node as HTMLElement
      return {
        clientHeight: element.clientHeight,
        scrollHeight: element.scrollHeight,
      }
    }),
  ])

  expect(todoMetrics.clientHeight).toBeGreaterThan(350)
  expect(reviewMetrics.clientHeight).toBeGreaterThan(350)
  expect(Math.abs(todoMetrics.clientHeight - reviewMetrics.clientHeight)).toBeLessThan(24)
  expect(todoMetrics.scrollHeight).toBeGreaterThan(todoMetrics.clientHeight + 100)
  expect(reviewMetrics.scrollHeight).toBeLessThanOrEqual(reviewMetrics.clientHeight + 4)

  const scrolledTop = await todoList.evaluate((node) => {
    const element = node as HTMLElement
    element.scrollTop = element.scrollHeight
    return element.scrollTop
  })

  expect(scrolledTop).toBeGreaterThan(0)
})
