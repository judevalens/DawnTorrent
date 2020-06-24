import {IpcMainEvent} from "electron"

export default class IpcMsg{
    msg :   object
    replyEvent : IpcMainEvent
    constructor(msg : object,event : IpcMainEvent) {
        this.msg = msg
        this.replyEvent = event
    }

    public static getMessage (msg : object,replyEvent : IpcMainEvent) :IpcMsg{
        return new IpcMsg(msg, replyEvent)
    }
}