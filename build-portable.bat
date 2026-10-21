@echo off
title PuffSQL - Portable Build Launcher
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0build-portable.ps1"
if errorlevel 1 pause
