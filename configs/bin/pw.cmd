@ECHO off
REM pw — hang-safe playwright-cli runner (see pw.cjs). Usage: pw [--timeout N] <args...>
node "%~dp0pw.cjs" %*
