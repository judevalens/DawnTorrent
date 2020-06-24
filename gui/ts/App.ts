import {app, BrowserWindow } from 'electron';
import Ipc from "./ipc";
const path = require('path');

const {ipcMain} = require('electron')

export default class Main{
    ipc : Ipc | undefined
    constructor() {
     this.ipc = new Ipc()
    }

    createWindow(){
        // Create the browser window.
        const mainWindow = new BrowserWindow({
            width: 800,
            height: 600,
            webPreferences: {
                nodeIntegration: true
            }
        });


        // and load the index.html of the app.
        mainWindow.loadFile(path.join(__dirname, 'statics/html/index.html'))

        // Open the DevTools.
        // mainWindow.webContents.openDevTools();
    }

    quit (){
        app.on('window-all-closed', () => {
            // On OS X it is common for applications and their menu bar
            // to stay active until the user quits explicitly with Cmd + Q
            if (process.platform !== 'darwin') {
                app.quit();
            }
        });
    }

    setUp(){
        app.on('ready', this.createWindow);
        app.on('window-all-closed', this.quit)
        app.on('activate', this.quit)

    }

}