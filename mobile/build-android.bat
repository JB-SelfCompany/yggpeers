@echo off
echo Building yggpeers Android library...

REM Check for Android SDK
if not defined ANDROID_HOME (
    set ANDROID_HOME=your\path\to\sdk
)

if not exist "%ANDROID_HOME%" (
    echo ERROR: Android SDK not found at %ANDROID_HOME%
    echo Please set ANDROID_HOME environment variable
    exit /b 1
)

echo Using Android SDK: %ANDROID_HOME%

REM Check for gomobile
where gomobile >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: gomobile not found in PATH
    echo Installing gomobile...
    go install golang.org/x/mobile/cmd/gomobile@latest
    if %ERRORLEVEL% NEQ 0 (
        echo Failed to install gomobile
        exit /b 1
    )
    echo Initializing gomobile...
    gomobile init
)

echo.
echo Building AAR for all architectures (ARM, ARM64, x86, x86_64)...
gomobile bind -target=android -androidapi 23 -ldflags="-checklinkname=0" -o yggpeers.aar github.com/jbselfcompany/yggpeers/mobile

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ========================================
    echo Build successful!
    echo ========================================
    echo Output files:
    echo   - yggpeers.aar
    echo   - yggpeers-sources.jar
    echo.
    echo You can now import yggpeers.aar into your Android project.
    echo.
) else (
    echo.
    echo ========================================
    echo Build FAILED!
    echo ========================================
    exit /b 1
)
