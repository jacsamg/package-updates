'use strict';

const path = require('path');
const { copyFile } = require('fs');

const isWindows = process.platform === 'win32';

if (isWindows) {
  const originPath = path.join(__dirname, 'dist', 'windows.exe');
  const destinationPath = path.join(__dirname, 'bin', 'cmd.exe');

  copyFile(originPath, destinationPath, (err) => {
    if (err) throw err;
    console.log('CLI installed successfully');
  });
}
