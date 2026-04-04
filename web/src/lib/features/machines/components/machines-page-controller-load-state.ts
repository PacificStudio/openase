import type { MachinesPageControllerLoadState } from './machines-page-controller-load'
import type { MachinesPageControllerOpsState } from './machines-page-controller-types'

type MachinesPageControllerLoadStateInput = {
  controllerState: MachinesPageControllerOpsState
  getLoading: () => boolean
  setLoading: (value: boolean) => void
  getRefreshing: () => boolean
  setRefreshing: (value: boolean) => void
  getActiveOrgId: () => string
  getListRequestVersion: () => number
  setListRequestVersion: (value: number) => void
  getQueuedReload: () => boolean
  setQueuedReload: (value: boolean) => void
  getReloadInFlight: () => boolean
  setReloadInFlight: (value: boolean) => void
}

export function createMachinesPageControllerLoadState(
  input: MachinesPageControllerLoadStateInput,
): MachinesPageControllerLoadState {
  return {
    ...input.controllerState,
    get loading() {
      return input.getLoading()
    },
    set loading(value) {
      input.setLoading(value)
    },
    get refreshing() {
      return input.getRefreshing()
    },
    set refreshing(value) {
      input.setRefreshing(value)
    },
    get activeOrgId() {
      return input.getActiveOrgId()
    },
    get listRequestVersion() {
      return input.getListRequestVersion()
    },
    set listRequestVersion(value) {
      input.setListRequestVersion(value)
    },
    get queuedReload() {
      return input.getQueuedReload()
    },
    set queuedReload(value) {
      input.setQueuedReload(value)
    },
    get reloadInFlight() {
      return input.getReloadInFlight()
    },
    set reloadInFlight(value) {
      input.setReloadInFlight(value)
    },
  }
}
