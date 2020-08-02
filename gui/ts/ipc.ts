import * as zm from "zeromq"
// @ts-ignore
import * as myIP from "ip"
import {ipcMain, IpcMainEvent} from "electron";
import IpcMsg from "./ipcMsg";
import ipcMsg from "./ipcMsg";

export default class Ipc {
    socket: zm.Pair
    Queue: Array<object>
    replyEvent: object
    msgIdIndex: number

    constructor() {
        const execFile = require('child_process')
        this.socket = new zm.Pair();
        this.socket.connect("ipc:///home/jude/DawnTorrent/files/tmp/feeds/10/1")
        console.log(this.socket.context)
        console.log("this.socket.context")
        this.Queue = new Array<IpcMsg>()
        this.replyEvent = {}
        this.msgIdIndex = 0
        this.setIpcListener()
        this.execJob()
        console.log(myIP.address())
        console.log(zm)
    }

    addMsg(msg: Object, event: IpcMainEvent) {
        this.Queue.push(IpcMsg.getMessage(msg, event))
    }

    async sendMsg(msg: object, replyEvent: IpcMainEvent): Promise<void> {
        console.log(msg)
        // @ts-ignore
        msg["MsgIndex"] = this.msgIdIndex++
        console.log(JSON.stringify(msg))
        // @ts-ignore
        this.replyEvent[msg["MsgIndex"]] = replyEvent
         await this.socket.send(JSON.stringify(msg)).catch((reason) => {
             console.log(reason)
         })
    }


    async sendMsgFromQueue(){
        let isBusy: boolean = false

        if (!isBusy) {
            const msg: IpcMsg = <ipcMsg>this.Queue.pop()

            if (msg !== undefined) {
                isBusy = true
                await this.sendMsg(msg.msg, msg.replyEvent).then((answer) => {
                    // console.log("answer " ,answer[0].toString() )
                    // msg.replyEvent.reply("answer",answer[0].toString())
                    isBusy = false
                }).catch((reason) => {
                    isBusy = false
                    console.log("req failedddd!")
                })
                console.log("executing job.......")
            }
        }

    }

    execJob() {
    let isReceiverBusy = false
        setInterval(async () => {

           await this.sendMsgFromQueue()

            if (!isReceiverBusy){
                isReceiverBusy = true
                try {
                    await this.socket.receive().then((incomingMsg) => {
                        console.log("answer", incomingMsg[0].toString())
                        console.log(incomingMsg)
                        let msg = JSON.parse(incomingMsg[0].toString())
                        let msgIndex = msg["MsgIndex"]
                        // @ts-ignore
                        this.replyEvent[msgIndex].reply("answer", msg)
                        isReceiverBusy = false
                    })

                }catch (e) {
                    console.log(e)
                    isReceiverBusy = false

                }

            }

        }, 5)


    }

    setIpcListener() {
        ipcMain.on("addMsgFromToQueue", (event, args) => {
            console.log("received msg")
            this.addMsg(args, event)
        })
    }

}