const fs = require('node:fs')
const path = require('node:path')

function createFileLogger(logPath) {
  fs.mkdirSync(path.dirname(logPath), { recursive: true })

  return {
    log(level, message, metadata = {}) {
      const entry = {
        timestamp: new Date().toISOString(),
        level,
        message,
        ...metadata,
      }
      fs.appendFileSync(logPath, `${JSON.stringify(entry)}\n`, 'utf8')
    },
    info(message, metadata) {
      this.log('info', message, metadata)
    },
    error(message, metadata) {
      this.log('error', message, metadata)
    },
  }
}

module.exports = {
  createFileLogger,
}
