"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
var electron_1 = require("electron");
var electron_2 = __importDefault(require("electron"));
var remote = electron_2.default.remote;
var dialog = remote.dialog;
var Command;
(function (Command) {
    Command[Command["AddTorrent"] = 0] = "AddTorrent";
    Command[Command["DeleteTorrent"] = 1] = "DeleteTorrent";
    Command[Command["PauseTorrent"] = 2] = "PauseTorrent";
    Command[Command["ResumeTorrent"] = 3] = "ResumeTorrent";
    Command[Command["GetProgress"] = 4] = "GetProgress";
    Command[Command["CloseClient"] = -2] = "CloseClient";
    Command[Command["InitClient"] = -1] = "InitClient";
})(Command = exports.Command || (exports.Command = {}));
var AddMode;
(function (AddMode) {
    AddMode[AddMode["openMode"] = 0] = "openMode";
    AddMode[AddMode["resumeMode"] = 1] = "resumeMode";
})(AddMode || (AddMode = {}));
var TorrentState;
(function (TorrentState) {
    TorrentState[TorrentState["startedState"] = 0] = "startedState";
    TorrentState[TorrentState["stoppedState"] = 1] = "stoppedState";
    TorrentState[TorrentState["completedState"] = 2] = "completedState";
})(TorrentState || (TorrentState = {}));
var App = /** @class */ (function () {
    function App() {
        this.sendButton = document.getElementById("send");
        this.torrentContainer = document.getElementById("torrentContainer");
        this.addTorrentButton = document.getElementById("addTorrent");
        this.startTorrentButton = document.getElementById("playTorrent");
        this.pauseTorrentButton = document.getElementById("pauseTorrent");
        this.torrents = {};
        this.setIpcListener();
        electron_1.ipcRenderer.send('addMsgFromToQueue', this.getMsgForClient(Command.InitClient));
        this.getProgress();
    }
    App.prototype.getMsgForTorrent = function (command, infoHash, path, addMode) {
        return {
            CommandType: command,
            Commands: [{
                    Command: command,
                    TorrentIPCData: {
                        InfoHash: infoHash,
                        Path: path,
                        AddMode: addMode
                    }
                }]
        };
    };
    App.prototype.getMsgForClient = function (command) {
        return {
            CommandType: command,
            Commands: []
        };
    };
    App.prototype.setIpcListener = function () {
        var _this = this;
        var _a, _b, _c, _d;
        // @ts-ignore
        electron_1.ipcRenderer.on("answer", function (event, args) {
            console.log("incoming response");
            var msg = JSON.parse(args);
            console.log(msg);
            _this.msgRouter(msg);
        });
        (_a = this.sendButton) === null || _a === void 0 ? void 0 : _a.addEventListener("click", function (event) {
            electron_1.ipcRenderer.send('addMsgFromToQueue', _this.getMsgForClient(Command.InitClient));
        });
        (_b = this.addTorrentButton) === null || _b === void 0 ? void 0 : _b.addEventListener("click", function () {
            var fileObject = dialog.showOpenDialog({ properties: ['openFile', 'multiSelections'] });
            fileObject.then(function (files) {
                // @ts-ignore
                if (!files.canceled) {
                    var filepath = void 0;
                    // @ts-ignore
                    for (var _i = 0, _a = files.filePaths; _i < _a.length; _i++) {
                        filepath = _a[_i];
                        console.log(filepath);
                        var msg = _this.getMsgForTorrent(Command.AddTorrent, undefined, filepath, AddMode.openMode);
                        console.log(msg);
                        console.log(JSON.stringify(msg));
                        electron_1.ipcRenderer.send('addMsgFromToQueue', msg);
                    }
                }
            });
        });
        (_c = this.startTorrentButton) === null || _c === void 0 ? void 0 : _c.addEventListener("click", function () {
            var key;
            console.log(_this.torrents);
            for (key in _this.torrents) {
                // @ts-ignore
                // @ts-ignore
                var torrent = _this.torrents[key];
                // @ts-ignore
                if (torrent.selected) {
                    // @ts-ignore
                    var torrentInfoHash = torrent.TorrentIPCData.InfoHash;
                    electron_1.ipcRenderer.send('addMsgFromToQueue', _this.getMsgForTorrent(Command.ResumeTorrent, torrentInfoHash, undefined));
                }
            }
        });
        (_d = this.pauseTorrentButton) === null || _d === void 0 ? void 0 : _d.addEventListener("click", function () {
            var key;
            for (key in _this.torrents) {
                // @ts-ignore
                var torrent = _this.torrents[key];
                // @ts-ignore
                if (torrent.selected) {
                    // @ts-ignore
                    var torrentInfoHash = torrent.TorrentIPCData.InfoHash;
                    electron_1.ipcRenderer.send('addMsgFromToQueue', _this.getMsgForTorrent(Command.PauseTorrent, torrentInfoHash, undefined));
                }
            }
        });
    };
    App.prototype.initClient = function (msg) {
        var command;
        // @ts-ignore
        for (var _i = 0, _a = msg.Commands; _i < _a.length; _i++) {
            command = _a[_i];
            // @ts-ignore
            this.addTorrent(command);
        }
        console.log(this.torrents);
    };
    App.prototype.addTorrent2 = function (command) {
        // @ts-ignore
        var _this = this;
        var _a;
        this.torrents[command.TorrentIPCData.InfoHash] = command;
        // @ts-ignore
        this.torrents[command.TorrentIPCData.InfoHash].selected = false;
        var torrentBlock = document.createElement("div");
        // @ts-ignore
        torrentBlock.setAttribute("id", command.TorrentIPCData.InfoHash);
        torrentBlock.classList.add("row");
        var torrentName = document.createElement("span");
        torrentName.classList.add("col-md-2");
        var torrentCurrentLen = document.createElement("span");
        torrentCurrentLen.classList.add("col-md-2");
        var torrentLen = document.createElement("span");
        torrentLen.classList.add("col-md-2");
        var torrentStatus = document.createElement("span");
        torrentStatus.classList.add("col-md-2");
        var playPause = document.createElement("span");
        // @ts-ignore
        playPause.setAttribute("infoHash", command.TorrentIPCData.InfoHash);
        playPause.classList.add("playResume", "col-md-2");
        // @ts-ignore
        playPause.setAttribute("torrentStatus", command.TorrentIPCData.State);
        playPause.addEventListener("click", function (event) {
            console.log(event);
            var torrentElement = event.target;
            // @ts-ignore
            var torrentInfoHash = torrentElement.getAttribute("infoHash");
            // @ts-ignore
            var torrentState = _this.torrents[torrentInfoHash].TorrentIPCData.State;
            var toggledPausePlay;
            console.log("current STate");
            console.log(_this.torrents);
            if (torrentState == TorrentState.startedState) {
                toggledPausePlay = Command.PauseTorrent;
            }
            else if (torrentState == TorrentState.stoppedState) {
                toggledPausePlay = Command.ResumeTorrent;
            }
            else {
                toggledPausePlay = -10;
                console.log("SHITTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTT");
            }
            // @ts-ignore
            console.log(torrentElement.getAttribute("infoHash"));
            electron_1.ipcRenderer.send('addMsgFromToQueue', _this.getMsgForTorrent(toggledPausePlay, torrentInfoHash, undefined));
        });
        playPause.textContent = "play";
        // @ts-ignore
        torrentName.textContent = "Name :" + command.TorrentIPCData.Name;
        // @ts-ignore
        torrentCurrentLen.textContent = "currentLen :" + command.TorrentIPCData.CurrentLen;
        // @ts-ignore
        torrentLen.textContent = "currentLen :" + command.TorrentIPCData.Len;
        // @ts-ignore
        torrentStatus.textContent = "Status :" + command.TorrentIPCData.Status;
        torrentBlock.appendChild(torrentName);
        torrentBlock.appendChild(torrentCurrentLen);
        torrentBlock.appendChild(torrentLen);
        torrentBlock.appendChild(torrentStatus);
        torrentBlock.appendChild(playPause);
        // @ts-ignore
        (_a = this.torrentContainer) === null || _a === void 0 ? void 0 : _a.appendChild(torrentBlock);
        torrentBlock.addEventListener("click", function (event) {
            var torrentElement = event.target;
            // @ts-ignore
            var torrentInfoHash = torrentElement.getAttribute("infoHash");
            //this.torrents
        });
    };
    App.prototype.addTorrent = function (command) {
        var _a;
        var that = this;
        // @ts-ignore
        this.torrents[command.TorrentIPCData.InfoHash] = command;
        // @ts-ignore
        var infoHash = command.TorrentIPCData.InfoHash;
        // @ts-ignore
        this.torrents[command.TorrentIPCData.InfoHash].selected = false;
        // @ts-ignore
        var torrentBlock = document.getElementById("torrentBlockData").cloneNode(true);
        torrentBlock.classList.remove("hidden");
        torrentBlock.setAttribute("id", infoHash);
        var torrentName = torrentBlock.getElementsByClassName("torrentName")[0];
        // @ts-ignore
        torrentName.textContent = command.TorrentIPCData.Name;
        var torrentState = torrentBlock.getElementsByClassName("torrentStatus")[0];
        // @ts-ignore
        if (command.TorrentIPCData.State == TorrentState.startedState) {
            torrentState.textContent = "downloading";
        }
        else {
            torrentState.textContent = "paused";
        }
        // @ts-ignore
        (_a = this.torrentContainer) === null || _a === void 0 ? void 0 : _a.appendChild(torrentBlock);
        torrentBlock.addEventListener("click", function (event) {
            var torrentElement = event.relatedTarget;
            // @ts-ignore
            var torrentInfoHash = this.getAttribute("id");
            // @ts-ignore
            that.torrents[torrentInfoHash].selected = !that.torrents[torrentInfoHash].selected;
            // @ts-ignore
            if (that.torrents[torrentInfoHash].selected) {
                this.classList.add("selectedTorrent");
            }
            else {
                this.classList.remove("selectedTorrent");
            }
            //this.torrents
        });
    };
    App.prototype.getProgress = function () {
        var _this = this;
        var i = 0;
        var y = 0;
        console.log("helP!!!!!!!");
        setInterval(function () {
            var key;
            console.log("requesting update ......");
            for (key in _this.torrents) {
                // @ts-ignore
                // @ts-ignore
                var torrent = _this.torrents[key];
                // @ts-ignore
                // @ts-ignore
                if (torrent.TorrentIPCData.State == TorrentState.startedState) {
                    y = -1;
                    // @ts-ignore
                    var torrentInfoHash = torrent.TorrentIPCData.InfoHash;
                    electron_1.ipcRenderer.send('addMsgFromToQueue', _this.getMsgForTorrent(Command.GetProgress, torrentInfoHash, undefined));
                    // @ts-ignore
                }
            }
        }, 200);
    };
    App.prototype.updateTorrent = function (command) {
        // @ts-ignore
        var playResumeButton = document.getElementById(command.TorrentIPCData.InfoHash).getElementsByClassName("playResume")[0];
        // @ts-ignore
        var currentTorrent = document.getElementById(command.TorrentIPCData.InfoHash);
        // @ts-ignore
        switch (command.Command) {
            case Command.PauseTorrent:
                // @ts-ignore
                var torrentState = currentTorrent.getElementsByClassName("torrentStatus")[0];
                // @ts-ignore
                this.torrents[command.TorrentIPCData.InfoHash].TorrentIPCData.State = command.TorrentIPCData.State;
                console.log("State changed to");
                // @ts-ignore
                console.log(this.torrents);
                torrentState.textContent = "paused";
                // @ts-ignore
                break;
            case Command.ResumeTorrent:
                // @ts-ignore
                var torrentState = currentTorrent.getElementsByClassName("torrentStatus")[0];
                // @ts-ignore
                this.torrents[command.TorrentIPCData.InfoHash].TorrentIPCData.State = command.TorrentIPCData.State;
                console.log("State changed to");
                // @ts-ignore
                console.log(this.torrents);
                // @ts-ignore
                torrentState.textContent = "downloading";
                break;
            case Command.GetProgress:
                // @ts-ignore
                var currentLen = command.TorrentIPCData.CurrentLen;
                // @ts-ignore
                var totalLen = this.torrents[command.TorrentIPCData.InfoHash].TorrentIPCData.Len;
                // @ts-ignore
                var currentDownloadSpeed = command.TorrentIPCData.DownloadRate;
                // @ts-ignore
                this.torrents[command.TorrentIPCData.InfoHash].TorrentIPCData.CurrentLen = currentLen;
                console.log("progress update ........");
                var torrentProgressBar = currentTorrent.getElementsByClassName("torrentProgressBar")[0];
                var torrentProgressData = torrentProgressBar.getElementsByClassName("torrentProgressData")[0];
                var downloadSpeed = currentTorrent.getElementsByClassName("torrentDownloadSpeed")[0];
                // @ts-ignore
                var percent = (currentLen / totalLen) * 100;
                // @ts-ignore
                torrentProgressBar.setAttribute("aria-valuenow", percent);
                // @ts-ignore
                torrentProgressBar.style.width = percent + "%";
                console.log(percent);
                var currentLenSt = this.bytesToSize(currentLen);
                var totalLenSt = this.bytesToSize(totalLen);
                torrentProgressData.textContent = percent.toFixed(2) + "% (" + currentLenSt + "/" + totalLenSt + ")";
                var downloadSpeedNumber = this.bytesToSize(currentDownloadSpeed);
                downloadSpeed.textContent = downloadSpeedNumber + "/s";
                break;
        }
    };
    App.prototype.msgRouter = function (msg) {
        var command;
        // @ts-ignore
        if (msg.CommandType >= 0) {
            // @ts-ignore
            for (var _i = 0, _a = msg.Commands; _i < _a.length; _i++) {
                command = _a[_i];
                // @ts-ignore
                switch (command.Command) {
                    case Command.InitClient:
                        break;
                    case Command.AddTorrent:
                        this.addTorrent(command);
                        break;
                    case Command.ResumeTorrent:
                        this.updateTorrent(command);
                        break;
                    case Command.PauseTorrent:
                        this.updateTorrent(command);
                        break;
                    case Command.GetProgress:
                        this.updateTorrent(command);
                        break;
                }
            }
        }
        else {
            // @ts-ignore
            switch (msg.CommandType) {
                case Command.InitClient:
                    this.initClient(msg);
            }
        }
    };
    App.prototype.bytesToSize = function (bytes) {
        var sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
        if (bytes == 0)
            return 'n/a';
        var i = Math.floor(Math.log(bytes) / Math.log(1024));
        if (i == 0)
            return bytes + ' ' + sizes[i];
        return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + sizes[i];
    };
    return App;
}());
exports.App = App;
