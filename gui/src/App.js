"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
var electron_1 = require("electron");
var ipc_1 = __importDefault(require("./ipc"));
var path = require('path');
var ipcMain = require('electron').ipcMain;
var Main = /** @class */ (function () {
    function Main() {
        this.ipc = new ipc_1.default();
    }
    Main.prototype.createWindow = function () {
        // Create the browser window.
        var mainWindow = new electron_1.BrowserWindow({
            width: 800,
            height: 600,
            webPreferences: {
                nodeIntegration: true
            }
        });
        // and load the index.html of the app.
        mainWindow.loadFile(path.join(__dirname, 'statics/html/index.html'));
        // Open the DevTools.
        // mainWindow.webContents.openDevTools();
    };
    Main.prototype.quit = function () {
        electron_1.app.on('window-all-closed', function () {
            // On OS X it is common for applications and their menu bar
            // to stay active until the user quits explicitly with Cmd + Q
            if (process.platform !== 'darwin') {
                electron_1.app.quit();
            }
        });
    };
    Main.prototype.setUp = function () {
        electron_1.app.on('ready', this.createWindow);
        electron_1.app.on('window-all-closed', this.quit);
        electron_1.app.on('activate', this.quit);
    };
    return Main;
}());
exports.default = Main;
