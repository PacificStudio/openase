const net = require('node:net')

async function findAvailablePort(host = '127.0.0.1') {
  return new Promise((resolve, reject) => {
    const server = net.createServer()

    server.unref()
    server.once('error', reject)
    server.listen({ host, port: 0 }, () => {
      const address = server.address()
      const port = address && typeof address === 'object' ? address.port : 0
      server.close((closeError) => {
        if (closeError) {
          reject(closeError)
          return
        }
        resolve(port)
      })
    })
  })
}

module.exports = {
  findAvailablePort,
}
