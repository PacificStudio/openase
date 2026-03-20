function createStatusSync() {
  let version = $state(0)

  return {
    get version() {
      return version
    },
    touch() {
      version += 1
    },
  }
}

export const statusSync = createStatusSync()
