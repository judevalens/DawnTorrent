"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.App = exports.Command = void 0;
var electron_1 = require("electron");
var Command;
(function (Command) {
    Command[Command["AddTorrent"] = 0] = "AddTorrent";
    Command[Command["DeleteTorrent"] = 1] = "DeleteTorrent";
    Command[Command["PauseTorrent"] = 2] = "PauseTorrent";
    Command[Command["ResumeTorrent"] = 3] = "ResumeTorrent";
    Command[Command["CloseClient"] = 4] = "CloseClient";
    Command[Command["InitClient"] = 5] = "InitClient";
    Command[Command["updateUI"] = 6] = "updateUI";
})(Command = exports.Command || (exports.Command = {}));
var App = /** @class */ (function () {
    function App() {
        this.sendButton = document.getElementById("send");
        this.setIpcListener();
    }
    App.prototype.getMsgForTorrent = function (infoHash, command) {
        return {
            Commands: {
                Command: command,
                TorrentIPCData: [{
                        InfoHash: infoHash
                    }]
            },
        };
    };
    App.prototype.getMsgForClient = function (command) {
        return {
            Commands: [
                { command: command,
                    TorrentIPCData: {}
                }
            ],
        };
    };
    App.prototype.setIpcListener = function () {
        var _this = this;
        var _a;
        // @ts-ignore
        electron_1.ipcRenderer.on("answer", function (event, args) {
        });
        (_a = this.sendButton) === null || _a === void 0 ? void 0 : _a.addEventListener("click", function (event) {
            electron_1.ipcRenderer.send('addMsgFromToQueue', _this.getMsgForClient(Command.InitClient));
        });
    };
    return App;
}());
exports.App = App;
