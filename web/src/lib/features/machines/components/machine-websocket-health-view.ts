import { i18nStore } from '$lib/i18n/store.svelte'
import type {
  MachineWebsocketHealth,
  MachineWebsocketHealthLayer,
  WebsocketHealthState,
} from '../types'
import { formatMachineRelativeTime } from '../machine-i18n'

export type WebsocketHealthLevelCard = {
  id: string
  label: string
  state: string
  value: string
  meta: string
}

export function buildWebsocketLevelCards(
  health: MachineWebsocketHealth,
): WebsocketHealthLevelCard[] {
  return [
    websocketLevelCard(
      'l2',
      i18nStore.t('machines.machineHealthPanel.levels.l2Link'),
      health.l2,
      i18nStore.t('machines.machineHealthPanel.dynamic.noWebsocketLinkObservation'),
    ),
    websocketLevelCard(
      'l3',
      i18nStore.t('machines.machineHealthPanel.levels.l3ControlPlaneNetwork'),
      health.l3,
      i18nStore.t('machines.machineHealthPanel.dynamic.noWebsocketNetworkObservation'),
    ),
    websocketLevelCard(
      'l4',
      i18nStore.t('machines.machineHealthPanel.levels.l4WebsocketTransport'),
      health.l4,
      i18nStore.t('machines.machineHealthPanel.dynamic.noWebsocketTransportObservation'),
    ),
    websocketLevelCard(
      'l5',
      i18nStore.t('machines.machineHealthPanel.levels.l5MachineAgent'),
      health.l5,
      i18nStore.t('machines.machineHealthPanel.dynamic.noWebsocketApplicationObservation'),
    ),
  ]
}

export function mapWebsocketLayerState(state: WebsocketHealthState | undefined): string {
  switch (state) {
    case 'healthy':
      return 'ok'
    case 'degraded':
      return 'warn'
    case 'failed':
      return 'error'
    default:
      return 'unknown'
  }
}

function websocketLevelCard(
  id: string,
  label: string,
  layer: MachineWebsocketHealthLayer | undefined,
  fallbackValue: string,
): WebsocketHealthLevelCard {
  return {
    id,
    label,
    state: mapWebsocketLayerState(layer?.state),
    value: layer?.reason ?? fallbackValue,
    meta: checkedAtLabel(layer?.observedAt),
  }
}

function checkedAtLabel(value: string | undefined): string {
  return value
    ? i18nStore.t('machines.machineHealthPanel.dynamic.checkedAt', {
        time: formatMachineRelativeTime(value),
      })
    : i18nStore.t('machines.machineHealthPanel.dynamic.notCheckedYet')
}
