import { ipcRenderer} from "electron"
import electron from 'electron'
const {remote} = electron;
const {dialog} = remote
export enum Command {
    AddTorrent = 0,
    DeleteTorrent = 1,
    PauseTorrent = 2,
    ResumeTorrent = 3,
    GetProgress = 4,
    CloseClient = -2,
    InitClient = -1 ,

}

enum AddMode{
    openMode = 0,
    resumeMode = 1
}

enum TorrentState{
    startedState   = 0,
stoppedState   = 1,
completedState = 2
}



export class App {

    sendButton: HTMLElement | null
    torrentContainer : HTMLElement | null
    addTorrentButton : HTMLElement | null
    startTorrentButton : HTMLElement | null
    pauseTorrentButton : HTMLElement | null
    torrents : object


    constructor() {
        this.sendButton = document.getElementById("send")
        this.torrentContainer  = document.getElementById("torrentContainer")
        this.addTorrentButton = document.getElementById("addTorrent")
        this.startTorrentButton = document.getElementById("playTorrent")
        this.pauseTorrentButton = document.getElementById("pauseTorrent")
        this.torrents = {}
        this.setIpcListener()
      ipcRenderer.send('addMsgFromToQueue', this.getMsgForClient(Command.InitClient))
        this.getProgress()

    }

    getMsgForTorrent( command: number, infoHashHex?: string,path?:string,addMode?:number): object {

        return {
            CommandType : command,
            Commands: [{
                Command: command,
                TorrentIPCData: {
                    InfoHashHex: infoHashHex,
                    Path : path,
                    AddMode : addMode
                }
            }]
        }
    }

    getMsgForClient(command: number): object {
        return {
            CommandType : command,
            Commands: []
        }
    }

    setIpcListener() {
        // @ts-ignore
        ipcRenderer.on("answer", (event, msg) => {
            console.log("incoming response")

            console.log(msg)
            this.msgRouter(msg)
        })

        this.sendButton?.addEventListener("click", (event) => {
            ipcRenderer.send('addMsgFromToQueue', this.getMsgForClient(Command.InitClient))
        })

        this.addTorrentButton?.addEventListener("click", () => {
           const fileObject : Promise<Object> = dialog.showOpenDialog({properties: ['openFile', 'multiSelections']})

           fileObject.then((files)=>{

               // @ts-ignore
               if (!files.canceled){
                   let filepath :string
                   // @ts-ignore
                   for (filepath of files.filePaths){
                       console.log(filepath)
                      let msg = this.getMsgForTorrent(Command.AddTorrent,undefined,filepath,AddMode.openMode)
                       console.log(msg)
                       console.log(JSON.stringify(msg))
                       ipcRenderer.send('addMsgFromToQueue', msg)
                   }
               }
           })
        })

        this.startTorrentButton?.addEventListener("click",()=>{
            let key : string
            console.log(this.torrents)
            for (key in this.torrents){
                // @ts-ignore
                // @ts-ignore
                const torrent : object = this.torrents[key]

                // @ts-ignore
                if (torrent.selected){
                    // @ts-ignore
                    const torrentInfoHash : string =   torrent.TorrentIPCData.InfoHashHex
                    ipcRenderer.send('addMsgFromToQueue', this.getMsgForTorrent(Command.ResumeTorrent,torrentInfoHash,undefined))

                }
            }
        })

        this.pauseTorrentButton?.addEventListener("click",()=>{
            let key : string
            for (key in this.torrents){
                // @ts-ignore
                const torrent : object = this.torrents[key]

                // @ts-ignore
                if (torrent.selected){
                    // @ts-ignore
                    const torrentInfoHash : string =   torrent.TorrentIPCData.InfoHashHex
                    ipcRenderer.send('addMsgFromToQueue', this.getMsgForTorrent(Command.PauseTorrent,torrentInfoHash,undefined))

                }
            }
        })

    }

    initClient(msg: object) {
        let command
        // @ts-ignore
        for (command of msg.Commands){
            // @ts-ignore
            this.addTorrent(command)
        }

        console.log(this.torrents)

    }

    addTorrent2(command: object) {
        // @ts-ignore

        this.torrents[command.TorrentIPCData.InfoHashHex] = command
        // @ts-ignore
        this.torrents[command.TorrentIPCData.InfoHashHex].selected = false
        let torrentBlock = document.createElement("div");
        // @ts-ignore
        torrentBlock.setAttribute("id",command.TorrentIPCData.InfoHashHex)
        torrentBlock.classList.add("row")
        let torrentName = document.createElement("span")
        torrentName.classList.add("col-md-2")
        let torrentCurrentLen = document.createElement("span")
        torrentCurrentLen.classList.add("col-md-2")

        let torrentLen = document.createElement("span")
        torrentLen.classList.add("col-md-2")

        let torrentStatus = document.createElement("span")
        torrentStatus.classList.add("col-md-2")

        let playPause = document.createElement("span")
        // @ts-ignore
        playPause.setAttribute("infoHash", command.TorrentIPCData.InfoHashHex)
        playPause.classList.add("playResume","col-md-2");
        // @ts-ignore
        playPause.setAttribute("torrentStatus", command.TorrentIPCData.State)

        playPause.addEventListener("click",  (event)=> {
            console.log(event)
            const torrentElement :EventTarget | null = event.target
            // @ts-ignore
            const torrentInfoHash : string =   torrentElement.getAttribute("infoHash")
            // @ts-ignore
            const torrentState : number = this.torrents[torrentInfoHash].TorrentIPCData.State
            let toggledPausePlay : number
            console.log("current STate")
            console.log(this.torrents)
            if (torrentState == TorrentState.startedState){
                toggledPausePlay = Command.PauseTorrent
            }else if (torrentState == TorrentState.stoppedState){
                toggledPausePlay = Command.ResumeTorrent
            }else{
                toggledPausePlay = -10
                console.log("SHITTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTT")
            }
            // @ts-ignore
            console.log(torrentElement.getAttribute("infoHash"))

            ipcRenderer.send('addMsgFromToQueue', this.getMsgForTorrent(toggledPausePlay,torrentInfoHash,undefined))

        })

        playPause.textContent = "play"
        // @ts-ignore
        torrentName.textContent = "Name :" + command.TorrentIPCData.Name
        // @ts-ignore
        torrentCurrentLen.textContent = "currentLen :" + command.TorrentIPCData.CurrentLen
        // @ts-ignore
        torrentLen.textContent = "currentLen :" + command.TorrentIPCData.Len
        // @ts-ignore
        torrentStatus.textContent = "Status :" + command.TorrentIPCData.Status
        torrentBlock.appendChild(torrentName)
        torrentBlock.appendChild(torrentCurrentLen)
        torrentBlock.appendChild(torrentLen)
        torrentBlock.appendChild(torrentStatus)
        torrentBlock.appendChild(playPause)

        // @ts-ignore


        this.torrentContainer?.appendChild(torrentBlock)

        torrentBlock.addEventListener("click",(event)=>{
            const torrentElement :EventTarget | null = event.target
            // @ts-ignore
            const torrentInfoHash : string =   torrentElement.getAttribute("infoHash")

            //this.torrents

        })
    }
    addTorrent(command: object) {
        let that : App= this
        // @ts-ignore

        this.torrents[command.TorrentIPCData.InfoHashHex] = command
        // @ts-ignore
        let infoHash : string = command.TorrentIPCData.InfoHashHex
        // @ts-ignore
        this.torrents[command.TorrentIPCData.InfoHashHex].selected = false
        // @ts-ignore
        let torrentBlock : HTMLElement  = document.getElementById("torrentBlockData").cloneNode(true);
        torrentBlock.classList.remove("hidden")
        torrentBlock.setAttribute("id",infoHash)

        let torrentName : Element = torrentBlock.getElementsByClassName("torrentName")[0]
        // @ts-ignore
        torrentName.textContent = command.TorrentIPCData.Name


        let torrentState : Element = torrentBlock.getElementsByClassName("torrentStatus")[0]

        // @ts-ignore
        if (command.TorrentIPCData.State == TorrentState.startedState){
            torrentState.textContent = "downloading"
        }else{
            torrentState.textContent = "paused"
        }

        // @ts-ignore

        this.torrentContainer?.appendChild(torrentBlock)

        torrentBlock.addEventListener("click",function(event){
            const torrentElement :EventTarget | null = event.relatedTarget
            // @ts-ignore
            const torrentInfoHash : string =   this.getAttribute("id")
            // @ts-ignore
            that.torrents[torrentInfoHash].selected = !that.torrents[torrentInfoHash].selected

            // @ts-ignore
            if (that.torrents[torrentInfoHash].selected){
                this.classList.add("selectedTorrent")
            }else{
                this.classList.remove("selectedTorrent")
            }
            //this.torrents

        })
    }

    getProgress(){
        let i : number = 0
        let y = 0
        console.log("helP!!!!!!!")
        setInterval(()=>{
            let key : string
            console.log("requesting update ......")
            for (key in this.torrents){
                // @ts-ignore
                // @ts-ignore
                const torrent : object = this.torrents[key]
                // @ts-ignore
                // @ts-ignore
                if (torrent.TorrentIPCData.State == TorrentState.startedState){
                    y = -1
                    // @ts-ignore
                    const torrentInfoHash : string =   torrent.TorrentIPCData.InfoHashHex
                    ipcRenderer.send('addMsgFromToQueue', this.getMsgForTorrent(Command.GetProgress,torrentInfoHash,undefined))
                    // @ts-ignore
                }
                if (y != 0){
                    // @ts-ignore
                    //torrent.selected = true

                    if (i % 2 == 0){
                  //  this.startTorrentButton?.click()

                    }else{
                      //this.pauseTorrentButton?.click()
                    }
                    i++
                    console.log(i)
                }

            }
        },200)
    }

    updateTorrent(command: Command) {
        // @ts-ignore
        const playResumeButton: HTMLElement | null = document.getElementById(command.TorrentIPCData.InfoHashHex
        ).getElementsByClassName("playResume")[0]
        // @ts-ignore
        const currentTorrent : Element = document.getElementById(command.TorrentIPCData.InfoHashHex)

        // @ts-ignore
        switch (command.Command) {
            case Command.PauseTorrent:

                // @ts-ignore
                let torrentState : Element = currentTorrent.getElementsByClassName("torrentStatus")[0]

                // @ts-ignore
                this.torrents[command.TorrentIPCData.InfoHashHex].TorrentIPCData.State = command.TorrentIPCData.State
                console.log("State changed to")
                // @ts-ignore
                console.log(this.torrents)
                torrentState.textContent ="paused"
                // @ts-ignore
                break;
            case Command.ResumeTorrent:

                // @ts-ignore
                let torrentState : Element = currentTorrent.getElementsByClassName("torrentStatus")[0]
                // @ts-ignore
                this.torrents[command.TorrentIPCData.InfoHashHex].TorrentIPCData.State = command.TorrentIPCData.State
                console.log("State changed to")
                // @ts-ignore
                console.log(this.torrents)
                // @ts-ignore
                torrentState.textContent ="downloading"
                break
            case Command.GetProgress:
                // @ts-ignore
                const currentLen :number = command.TorrentIPCData.CurrentLen
                // @ts-ignore
                const totalLen : number = this.torrents[command.TorrentIPCData.InfoHashHex].TorrentIPCData.Len

                // @ts-ignore
                const currentDownloadSpeed : number = command.TorrentIPCData.DownloadRate

                // @ts-ignore
                this.torrents[command.TorrentIPCData.InfoHashHex].TorrentIPCData.CurrentLen = currentLen
                console.log("progress update ........")
                let torrentProgressBar : Element =  currentTorrent.getElementsByClassName("torrentProgressBar")[0]
                let torrentProgressData : Element = torrentProgressBar.getElementsByClassName("torrentProgressData")[0]
                let downloadSpeed : Element = currentTorrent.getElementsByClassName("torrentDownloadSpeed")[0]
                // @ts-ignore
                const percent = (currentLen/totalLen)*100
                // @ts-ignore
                torrentProgressBar.setAttribute("aria-valuenow",percent)
                // @ts-ignore
                torrentProgressBar.style.width = percent+ "%"
                console.log(percent)

               const currentLenSt :string = this.bytesToSize(currentLen)
                const totalLenSt : string =  this.bytesToSize(totalLen)

                torrentProgressData.textContent = percent.toFixed(2) +"% ("+currentLenSt+"/"+totalLenSt+")"

                const downloadSpeedNumber : string = this.bytesToSize(currentDownloadSpeed)

                downloadSpeed.textContent = downloadSpeedNumber+"/s"

                break
        }
    }

    msgRouter(msg: object) {
        let command;
        // @ts-ignore
        if (msg.CommandType >= 0){
            // @ts-ignore
            for (command of msg.Commands) {
                // @ts-ignore
                switch (command.Command) {
                    case Command.InitClient:
                        break;
                    case Command.AddTorrent:
                        this.addTorrent(command)
                        break;
                    case Command.ResumeTorrent:
                        this.updateTorrent(command)
                        break;
                    case Command.PauseTorrent:
                        this.updateTorrent(command);
                        break
                    case Command.GetProgress :
                        this.updateTorrent(command);
                        break;
                }
            }

        }else{

            // @ts-ignore
            switch (msg.CommandType) {
                case Command.InitClient:
                    this.initClient(msg)
            }
        }

    }

     bytesToSize(bytes : number) {
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
        if (bytes == 0) return 'n/a';
        const i : number =Math.floor(Math.log(bytes) / Math.log(1024))
        if (i == 0) return bytes + ' ' + sizes[i];
        return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + sizes[i];
    }

}