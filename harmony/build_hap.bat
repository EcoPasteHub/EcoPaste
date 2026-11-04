@echo off
setlocal enabledelayedexpansion

echo ===================================================
echo   EcoPaste HarmonyOS HAP Build Tool
echo ===================================================

set "DEVECO_DIR=E:\software\DevEco Studio"
if not exist "%DEVECO_DIR%" (
    echo [ERROR] DevEco Studio not found at %DEVECO_DIR%
    pause
    exit /b 1
)

set "JAVA_HOME=%DEVECO_DIR%\jbr"
set "NODE_HOME=%DEVECO_DIR%\tools\node"
set "DEVECO_SDK_HOME=%DEVECO_DIR%\sdk"
set "PATH=%DEVECO_DIR%\jbr\bin;%DEVECO_DIR%\tools\node;%PATH%"

set "SRC_DIR=%~dp0"
set "BUILD_DIR=D:\EcoPaste_Harmony_Project"

set "DO_BUMP=1"
for %%a in (%*) do (
    if /i "%%~a"=="--no-bump" set "DO_BUMP=0"
)

if "%DO_BUMP%"=="1" (
    echo [1/5] Auto-incrementing HarmonyOS version...
    call node "%SRC_DIR%bump_version.cjs" %*
) else (
    echo [1/5] Skipping version bump (--no-bump)
)

for /f "delims=" %%v in ('node "%SRC_DIR%bump_version.cjs" --get') do set "APP_VERSION=%%v"
for /f "delims=" %%c in ('node "%SRC_DIR%bump_version.cjs" --get-code') do set "APP_CODE=%%c"

echo Target Version: v%APP_VERSION% (versionCode: %APP_CODE%)

echo.
echo [2/5] Syncing source files to ASCII workspace: %BUILD_DIR% ...
if exist "%BUILD_DIR%\entry\build" rd /s /q "%BUILD_DIR%\entry\build"
robocopy "%SRC_DIR:~0,-1%" "%BUILD_DIR%" /E /XD .hvigor build /XF *.log build_hap.bat >nul

if not exist "%BUILD_DIR%\local.properties" (
    echo sdk.dir=E:/software/DevEco Studio/sdk > "%BUILD_DIR%\local.properties"
)

cd /d "%BUILD_DIR%"

echo.
echo [3/5] Building Debug HAP...
call "%DEVECO_DIR%\tools\hvigor\bin\hvigorw.bat" assembleApp --mode project -p product=default -p buildMode=debug --no-daemon
if errorlevel 1 (
    echo [ERROR] Debug build failed!
    cd /d "%SRC_DIR%"
    exit /b 1
)

if exist "%BUILD_DIR%\entry\build\default\outputs\default\app\entry-default.hap" (
    copy /y "%BUILD_DIR%\entry\build\default\outputs\default\app\entry-default.hap" "%SRC_DIR%EcoPaste-HarmonyOS-v%APP_VERSION%-debug.hap" >nul
    echo [SUCCESS] Debug HAP generated: %SRC_DIR%EcoPaste-HarmonyOS-v%APP_VERSION%-debug.hap
)

echo.
echo [4/5] Building Release HAP...
call "%DEVECO_DIR%\tools\hvigor\bin\hvigorw.bat" assembleApp --mode project -p product=default -p buildMode=release --no-daemon
if errorlevel 1 (
    echo [ERROR] Release build failed!
    cd /d "%SRC_DIR%"
    exit /b 1
)

if exist "%BUILD_DIR%\entry\build\default\outputs\default\app\entry-default.hap" (
    copy /y "%BUILD_DIR%\entry\build\default\outputs\default\app\entry-default.hap" "%SRC_DIR%EcoPaste-HarmonyOS-v%APP_VERSION%-release.hap" >nul
    copy /y "%BUILD_DIR%\entry\build\default\outputs\default\app\entry-default.hap" "%SRC_DIR%EcoPaste-HarmonyOS-v%APP_VERSION%.hap" >nul
    copy /y "%BUILD_DIR%\entry\build\default\outputs\default\app\entry-default.hap" "%SRC_DIR%EcoPaste-HarmonyOS-latest.hap" >nul
    echo [SUCCESS] Release HAP generated: %SRC_DIR%EcoPaste-HarmonyOS-v%APP_VERSION%-release.hap
    echo [SUCCESS] Default HAP generated: %SRC_DIR%EcoPaste-HarmonyOS-v%APP_VERSION%.hap
    echo [SUCCESS] Latest HAP link updated: %SRC_DIR%EcoPaste-HarmonyOS-latest.hap
)

if exist "%BUILD_DIR%\build\outputs\default\EcoPaste_Harmony_Project-default-unsigned.app" (
    copy /y "%BUILD_DIR%\build\outputs\default\EcoPaste_Harmony_Project-default-unsigned.app" "%SRC_DIR%EcoPaste-HarmonyOS-v%APP_VERSION%.app" >nul
    echo [SUCCESS] Release APP generated: %SRC_DIR%EcoPaste-HarmonyOS-v%APP_VERSION%.app
)

echo.
echo [5/5] Done! All generated files in %SRC_DIR%:
dir /b "%SRC_DIR%*.hap" "%SRC_DIR%*.app" 2>nul

cd /d "%SRC_DIR%"
