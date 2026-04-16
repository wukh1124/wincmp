@echo off
set "ROOT=%~dp0.."
"%ROOT%\bin\mailpit\mailpit-1.29.6\mailpit.exe" -s 127.0.0.1:1025 -l 127.0.0.1:8025 --smtp-auth-accept-any --smtp-auth-allow-insecure -d "%ROOT%\data\mailpit"