import * as zm from "zeromq"
import {Message} from "zeromq"
// @ts-ignore
import * as myIP from "ip"
import {ipcMain, IpcMainEvent} from "electron";
import IpcMsg from "./ipcMsg";
import ipcMsg from "./ipcMsg";

export default class Ipc {
    socket : zm.Request
    Queue : Array<object>
    constructor() {
        const execFile = require('child_process')
        this.socket = new zm.Request();
        this.socket.connect("tcp://127.0.0.1:5555")
        this.socket.receiveTimeout = 1000
        this.Queue = new Array<IpcMsg>()
        this.setIpcListener()
        this.execJob()
        console.log(myIP.address())
        console.log(zm)
    }

    addMsg(msg: Object,event:IpcMainEvent){
        this.Queue.push(IpcMsg.getMessage(msg,event))

    }

    async sendMsg(msg: object, replyEvent: IpcMainEvent) :Promise<Message[]>{
        console.log(msg)
        console.log(JSON.stringify(msg))
        await this.socket.send(JSON.stringify(msg))
        return await this.socket.receive()
    }

    execJob (){
     let isBusy : boolean = false
        setInterval(async ()=>{

            if (!isBusy){
                const msg : IpcMsg= <ipcMsg>this.Queue.pop()

                if (msg !== undefined){
                    isBusy = true
                    await this.sendMsg(msg.msg, msg.replyEvent).then((answer) =>{
                        console.log("answer " ,answer[0].toString() )
                        msg.replyEvent.reply("answer",answer[0].toString())
                        isBusy = false
                    }).catch((reason) => {
                        isBusy = false
                        console.log("req failedddd!")
                    })
                    console.log("executing job.......")
                }
            }

        },0.01)


    }

    setIpcListener(){
        ipcMain.on("addMsgFromToQueue",(event,args) =>{
            console.log("received msg")
            this.addMsg(args,event)
        })
    }

}