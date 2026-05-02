!define APP_NAME "Lighthouse"
!define APP_VERSION "1.0.0-win"
!define EXE_NAME "lighthouse.exe"
!define INSTALL_DIR "$LOCALAPPDATA\Lighthouse"
!define UNINSTALL_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lighthouse"

Name "${APP_NAME} ${APP_VERSION}"
OutFile "LighthouseSetup.exe"
InstallDir "${INSTALL_DIR}"
RequestExecutionLevel user
SetCompressor lzma

!include "MUI2.nsh"
!include "LogicLib.nsh"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Function WriteInstallPathScript
    InitPluginsDir

    FileOpen $8 "$PLUGINSDIR\lighthouse-path-install.ps1" w
    FileWrite $8 "$$ErrorActionPreference = 'Stop'$\r$\n"
    FileWrite $8 "$$dir = $$args[0]$\r$\n"
    FileWrite $8 "$$path = [Environment]::GetEnvironmentVariable('Path', 'User')$\r$\n"
    FileWrite $8 "if ($$null -eq $$path) { $$path = '' }$\r$\n"
    FileWrite $8 "$$items = @($$path -split ';' | Where-Object { $$_ -ne '' })$\r$\n"
    FileWrite $8 "$$exists = $$false$\r$\n"
    FileWrite $8 "foreach ($$item in $$items) {$\r$\n"
    FileWrite $8 "  if ($$item.TrimEnd('\') -ieq $$dir.TrimEnd('\')) { $$exists = $$true; break }$\r$\n"
    FileWrite $8 "}$\r$\n"
    FileWrite $8 "if (-not $$exists) {$\r$\n"
    FileWrite $8 "  $$backup = [Environment]::GetEnvironmentVariable('Path_Backup_Before_Lighthouse', 'User')$\r$\n"
    FileWrite $8 "  if ($$null -eq $$backup) {$\r$\n"
    FileWrite $8 "    [Environment]::SetEnvironmentVariable('Path_Backup_Before_Lighthouse', $$path, 'User')$\r$\n"
    FileWrite $8 "  }$\r$\n"
    FileWrite $8 "  $$items = @($$items) + $$dir$\r$\n"
    FileWrite $8 "  [Environment]::SetEnvironmentVariable('Path', ($$items -join ';'), 'User')$\r$\n"
    FileWrite $8 "}$\r$\n"
    FileWrite $8 "exit 0$\r$\n"
    FileClose $8
FunctionEnd

Function un.WriteUninstallPathScript
    InitPluginsDir

    FileOpen $8 "$PLUGINSDIR\lighthouse-path-uninstall.ps1" w
    FileWrite $8 "$$ErrorActionPreference = 'Stop'$\r$\n"
    FileWrite $8 "$$dir = $$args[0]$\r$\n"
    FileWrite $8 "$$path = [Environment]::GetEnvironmentVariable('Path', 'User')$\r$\n"
    FileWrite $8 "if ($$null -ne $$path) {$\r$\n"
    FileWrite $8 "  $$items = @($$path -split ';' | Where-Object { $$_ -ne '' -and $$_.TrimEnd('\') -ine $$dir.TrimEnd('\') })$\r$\n"
    FileWrite $8 "  [Environment]::SetEnvironmentVariable('Path', ($$items -join ';'), 'User')$\r$\n"
    FileWrite $8 "}$\r$\n"
    FileWrite $8 "exit 0$\r$\n"
    FileClose $8
FunctionEnd


Section "Install"
    SetOutPath "$INSTDIR"

    File "cli\lighthouse.exe"

    Call WriteInstallPathScript

    ExecWait 'powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$PLUGINSDIR\lighthouse-path-install.ps1" "$INSTDIR"' $9

    ${If} $9 != 0
        MessageBox MB_ICONSTOP "Could not update user PATH."
        Abort
    ${EndIf}

    SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

    WriteUninstaller "$INSTDIR\uninstall.exe"

    WriteRegStr HKCU "${UNINSTALL_KEY}" "DisplayName" "${APP_NAME}"
    WriteRegStr HKCU "${UNINSTALL_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegStr HKCU "${UNINSTALL_KEY}" "DisplayVersion" "${APP_VERSION}"
    WriteRegStr HKCU "${UNINSTALL_KEY}" "Publisher" "${APP_NAME}"
    WriteRegStr HKCU "${UNINSTALL_KEY}" "InstallLocation" "$INSTDIR"

    ; Exec '"$INSTDIR\${EXE_NAME}" up'
SectionEnd

Section "Uninstall"
    ExecWait '"$INSTDIR\${EXE_NAME}" down'

    Call un.WriteUninstallPathScript

    ExecWait 'powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$PLUGINSDIR\lighthouse-path-uninstall.ps1" "$INSTDIR"' $9

    SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

    Delete "$INSTDIR\${EXE_NAME}"
    Delete "$INSTDIR\uninstall.exe"
    RMDir "$INSTDIR"

    DeleteRegKey HKCU "${UNINSTALL_KEY}"
SectionEnd
