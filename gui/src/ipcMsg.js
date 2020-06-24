"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var IpcMsg = /** @class */ (function () {
    function IpcMsg(msg, event) {
        this.msg = msg;
        this.replyEvent = event;
    }
    IpcMsg.getMessage = function (msg, replyEvent) {
        return new IpcMsg(msg, replyEvent);
    };
    return IpcMsg;
}());
exports.default = IpcMsg;
