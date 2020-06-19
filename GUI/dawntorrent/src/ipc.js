module.exports = class DawnTorrentIpc {
    constructor() {
        const execFile = require('child_process')
        const child = execFile.execFile("C:\\Users\\jude\\go\\src\\DawnTorrent\\DawnTorrent.exe", (err, stdout, stdin) => {

        })

    }

    hi() {
        console.log("IPCCCCCCC")
    }
}