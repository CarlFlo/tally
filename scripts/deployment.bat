@echo off
setlocal

cd /d "%~dp0.." || (
    echo Failed to enter repository root.
    pause
    exit /b 1
)

for /f "delims=" %%B in ('git branch --show-current 2^>nul') do set "BRANCH=%%B"
for /f "delims=" %%C in ('git rev-parse --short HEAD 2^>nul') do set "COMMIT=%%C"

if not defined COMMIT (
    echo This does not appear to be a Git repository.
    pause
    exit /b 1
)

if not defined BRANCH set "BRANCH=(detached HEAD)"

echo.
echo ========================================
echo  Tally Deployment
echo ========================================
echo.
echo Branch: %BRANCH%
echo Commit: %COMMIT%
echo.
echo Building the current checkout without changing Git state...
echo.

echo Validating Docker Compose configuration...
docker compose config -q
if errorlevel 1 (
    echo Docker Compose configuration validation failed.
    pause
    exit /b 1
)

echo.
echo Building Docker image...
echo.
echo The Dockerfile will:
echo  - install locked frontend dependencies with npm ci
echo  - build the React/Vite frontend
echo  - embed the generated frontend assets
echo  - download Go modules
echo  - run go test ./...
echo  - compile the production binary
echo  - assemble the runtime image
echo.

docker compose build
if errorlevel 1 (
    echo.
    echo Docker image build failed.
    pause
    exit /b 1
)

echo.
echo Deploying built image...
docker compose up -d --no-build
if errorlevel 1 (
    echo.
    echo Docker Compose deployment failed.
    pause
    exit /b 1
)

set "HEALTH_URL=http://127.0.0.1:8080/healthz"
set "APP_URL=http://127.0.0.1:8080"
set "HEALTHY=0"

echo.
echo Waiting for Tally to become healthy...

for /l %%A in (1,1,30) do (
    curl -fsS --max-time 2 "%HEALTH_URL%" >nul 2>&1

    if not errorlevel 1 (
        set "HEALTHY=1"
        goto :healthy
    )

    timeout /t 1 /nobreak >nul
)

:healthy

echo.
docker compose ps

if "%HEALTHY%"=="0" (
    echo.
    echo ========================================
    echo  Deployment failed health check
    echo ========================================
    echo.
    echo Tally did not become healthy at:
    echo %HEALTH_URL%
    echo.
    echo Recent container logs:
    echo.
    docker compose logs --tail=50
    echo.
    pause
    exit /b 1
)

echo.
echo ========================================
echo  Deployment complete
echo ========================================
echo.
echo Branch: %BRANCH%
echo Commit: %COMMIT%
echo URL:    %APP_URL%
echo.
pause