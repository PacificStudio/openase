<script lang="ts">
  import {
    Chart,
    LineController,
    LineElement,
    PointElement,
    LinearScale,
    CategoryScale,
    Filler,
    Tooltip,
  } from 'chart.js'

  Chart.register(
    LineController,
    LineElement,
    PointElement,
    LinearScale,
    CategoryScale,
    Filler,
    Tooltip,
  )

  let {
    labels,
    datasets,
    tooltipCallback,
    yTickFormat,
    class: className = '',
  }: {
    labels: string[]
    datasets: Array<{
      label?: string
      data: number[]
      borderColor?: string
      backgroundColor?: string
      fill?: boolean | string
      tension?: number
      borderWidth?: number
      pointRadius?: number
      pointHoverRadius?: number
      pointBackgroundColor?: string
      pointBorderColor?: string
      pointBorderWidth?: number
    }>
    tooltipCallback?: (index: number) => string
    yTickFormat?: (value: number) => string
    class?: string
  } = $props()

  let canvasEl: HTMLCanvasElement | undefined = $state()
  let chart: Chart | undefined

  function getCSSColor(varName: string): string {
    if (!canvasEl) return ''
    return getComputedStyle(canvasEl).getPropertyValue(varName).trim()
  }

  function buildChart() {
    if (!canvasEl) return

    const borderColor = getCSSColor('--border')
    const mutedFg = getCSSColor('--muted-foreground')
    const fontFamily = getCSSColor('--font-sans') || 'Inter, sans-serif'

    chart = new Chart(canvasEl, {
      type: 'line',
      data: { labels, datasets },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        animation: { duration: 400, easing: 'easeOutQuart' },
        interaction: {
          mode: 'index',
          intersect: false,
        },
        layout: {
          padding: { top: 4, right: 4, bottom: 0, left: 0 },
        },
        scales: {
          x: {
            display: true,
            grid: { display: false },
            border: { display: false },
            ticks: {
              color: mutedFg,
              font: { family: fontFamily, size: 11 },
              maxRotation: 0,
              autoSkip: true,
              maxTicksLimit: 6,
            },
          },
          y: {
            display: true,
            grid: {
              color: borderColor,
              lineWidth: 0.8,
            },
            border: { display: false, dash: [4, 4] },
            ticks: {
              color: mutedFg,
              font: { family: fontFamily, size: 11 },
              maxTicksLimit: 5,
              callback: yTickFormat ? (value) => yTickFormat(Number(value)) : undefined,
            },
            beginAtZero: true,
          },
        },
        plugins: {
          tooltip: {
            enabled: true,
            backgroundColor: getCSSColor('--card'),
            titleColor: getCSSColor('--foreground'),
            bodyColor: getCSSColor('--muted-foreground'),
            borderColor: borderColor,
            borderWidth: 1,
            cornerRadius: 6,
            padding: { top: 8, bottom: 8, left: 10, right: 10 },
            titleFont: { family: fontFamily, size: 12, weight: 600 },
            bodyFont: { family: fontFamily, size: 11 },
            displayColors: false,
            callbacks: tooltipCallback
              ? {
                  title: () => '',
                  label: (ctx) => tooltipCallback(ctx.dataIndex),
                }
              : undefined,
          },
          legend: { display: false },
        },
      },
    })
  }

  $effect(() => {
    if (!canvasEl) return

    // Re-read reactive props so the chart rebuilds when callers replace arrays/functions.
    void labels
    void datasets
    void tooltipCallback
    void yTickFormat

    buildChart()

    return () => chart?.destroy()
  })
</script>

<div class={className}>
  <canvas bind:this={canvasEl}></canvas>
</div>
