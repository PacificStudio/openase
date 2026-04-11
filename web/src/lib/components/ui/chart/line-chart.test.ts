import { render } from '@testing-library/svelte'
import { tick } from 'svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import LineChart from './line-chart.svelte'

const { MockChart, chartInstances, registerSpy } = vi.hoisted(() => {
  const chartInstances: Array<{ destroy: ReturnType<typeof vi.fn>; config: unknown }> = []
  const registerSpy = vi.fn()

  class MockChart {
    static register = registerSpy
    destroy = vi.fn()

    constructor(_canvas: HTMLCanvasElement, config: unknown) {
      chartInstances.push({ destroy: this.destroy, config })
    }
  }

  return { MockChart, chartInstances, registerSpy }
})

vi.mock('chart.js', () => {
  class MockController {}

  return {
    Chart: MockChart,
    LineController: MockController,
    LineElement: MockController,
    PointElement: MockController,
    LinearScale: MockController,
    CategoryScale: MockController,
    Filler: MockController,
    Tooltip: MockController,
  }
})

describe('LineChart', () => {
  afterEach(() => {
    chartInstances.length = 0
    registerSpy.mockClear()
  })

  it('mounts and updates without entering a reactive rebuild loop', async () => {
    const view = render(LineChart, {
      props: {
        labels: ['Mon', 'Tue'],
        datasets: [{ data: [12, 24], borderColor: '#0f0' }],
      },
    })

    await tick()

    expect(registerSpy).toHaveBeenCalledOnce()
    expect(chartInstances).toHaveLength(1)

    await view.rerender({
      labels: ['Wed', 'Thu'],
      datasets: [{ data: [18, 36], borderColor: '#0f0' }],
    })
    await tick()

    expect(chartInstances).toHaveLength(2)
    expect(chartInstances[0]?.destroy).toHaveBeenCalledTimes(1)
    expect(chartInstances[1]?.destroy).toHaveBeenCalledTimes(0)
  })
})
