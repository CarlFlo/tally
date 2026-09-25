@echo off
setlocal

cd /d "%~dp0.." || (
    echo Failed to enter repository root.
    exit /b 1
)

for /f "delims=" %%B in ('git branch --show-current 2^>nul') do set "BRANCH=%%B"
for /f "delims=" %%C in ('git rev-parse --short HEAD 2^>nul') do set "COMMIT=%%C"

if not defined COMMIT (
    echo This does not appear to be a Git repository.
    exit /b 1
)

if not defined BRANCH set "BRANCH=(detached HEAD)"

echo.
echo ========================================
echo  Tally Local Container Deployment
echo ========================================
echo.
echo Branch: %BRANCH%
echo Commit: %COMMIT%
echo.

where docker >nul 2>&1
if errorlevel 1 (
    echo Docker was not found. Start or install Docker Desktop and retry.
    exit /b 1
)

echo Validating Docker Compose configuration...
docker compose config -q
if errorlevel 1 (
    echo Docker Compose configuration validation failed.
    exit /b 1
)

set "BUILD_LOG=%TEMP%\tally-local-build-%RANDOM%.log"
set "DEPLOY_LOG=%TEMP%\tally-local-deploy-%RANDOM%.log"

echo.
echo Building the local tally:local image...
docker compose build >"%BUILD_LOG%" 2>&1
if errorlevel 1 goto :build_failed
del "%BUILD_LOG%" >nul 2>&1

echo Starting the local container...
docker compose up -d --no-build >"%DEPLOY_LOG%" 2>&1
if errorlevel 1 goto :deploy_failed
del "%DEPLOY_LOG%" >nul 2>&1

set "HEALTH_URL=http://127.0.0.1:8080/readyz"
set "HEALTHY=0"
echo Waiting for Tally readiness...

for /l %%A in (1,1,30) do (
    curl.exe -fsS --max-time 2 "%HEALTH_URL%" >nul 2>&1
    if not errorlevel 1 (
        set "HEALTHY=1"
        goto :healthy
    )
    powershell.exe -NoProfile -Command "Start-Sleep -Seconds 1" >nul 2>&1
)

:healthy
if "%HEALTHY%"=="0" goto :health_failed

echo.
echo ========================================
echo  Local deployment complete
echo ========================================
echo.
echo Branch: %BRANCH%
echo Commit: %COMMIT%
echo URL:    http://127.0.0.1:8080
echo Image:  tally:local
exit /b 0

:build_failed
echo.
echo Docker image build failed. Build log: %BUILD_LOG%
type "%BUILD_LOG%"
exit /b 1

:deploy_failed
echo.
echo Local Docker Compose deployment failed. Deploy log: %DEPLOY_LOG%
type "%DEPLOY_LOG%"
exit /b 1

:health_failed
echo.
echo Tally did not become ready at %HEALTH_URL%.
echo Recent container logs:
docker compose logs --tail=50
exit /b 1
