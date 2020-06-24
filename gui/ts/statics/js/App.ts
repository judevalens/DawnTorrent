import {ipcRenderer} from "electron"

export enum Command {
    AddTorrent = 0,
    DeleteTorrent,
    PauseTorrent,
    ResumeTorrent,
    CloseClient,
    InitClient,
    updateUI = 6,
}
export class App {

    sendButton: HTMLElement | null

    constructor() {
        this.sendButton = document.getElementById("send")
        this.setIpcListener()
    }


    getMsgForTorrent(infoHash: string, command: number): object {
        return {
            Commands: {
                Command: command,
                TorrentIPCData: [{
                    InfoHash: infoHash
                }]
            },


        }
    }

    getMsgForClient(command: number): object {
        return {
            Commands: [
                { command : command,
                TorrentIPCData: {}
            }],


        }
    }

    setIpcListener() {
        // @ts-ignore
        ipcRenderer.on("answer", (event, args) => {

        })

        this.sendButton?.addEventListener("click", (event) => {
            ipcRenderer.send('addMsgFromToQueue', this.getMsgForClient(Command.InitClient))
        })
    }


}