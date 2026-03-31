!define APP_NAME "Lighthouse"
!define APP_VERSION "1.0.0-win"
!define EXE_NAME "lighthouse.exe"
!define INSTALL_DIR "$LOCALAPPDATA\Lighthouse"

Name "${APP_NAME} ${APP_VERSION}"
OutFile "LighthouseSetup.exe"
InstallDir "${INSTALL_DIR}"
RequestExecutionLevel user
SetCompressor lzma

!include "MUI2.nsh"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section "Install"
    SetOutPath "${INSTALL_DIR}"

    ; copy the executable
    File "cli\lighthouse.exe"

    ; add to user PATH
    ReadRegStr $0 HKCU "Environment" "Path"
    StrCmp $0 "" 0 +2
        StrCpy $0 "${INSTALL_DIR}"
    StrCpy $0 "$0;${INSTALL_DIR}"
    WriteRegStr HKCU "Environment" "Path" "$0"
    SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

    ; create uninstaller
    WriteUninstaller "${INSTALL_DIR}\uninstall.exe"

    ; add to Add/Remove Programs
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lighthouse" "DisplayName" "${APP_NAME}"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lighthouse" "UninstallString" "${INSTALL_DIR}\uninstall.exe"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lighthouse" "DisplayVersion" "${APP_VERSION}"

    ; run lighthouse up after install
    Exec '"$INSTDIR\${EXE_NAME}" up'
SectionEnd

Section "Uninstall"
    ; stop lighthouse
    ExecWait '"$INSTDIR\${EXE_NAME}" down'

    ; remove from PATH via PowerShell
    ExecWait 'powershell -Command "$p = [Environment]::GetEnvironmentVariable(\"Path\", \"User\"); $p = ($p.Split(\";\") | Where-Object { $_ -ne \"$INSTDIR\" }) -join \";\"; [Environment]::SetEnvironmentVariable(\"Path\", $p, \"User\")"'

    ; remove files
    Delete "${INSTALL_DIR}\${EXE_NAME}"
    Delete "${INSTALL_DIR}\uninstall.exe"
    RMDir "${INSTALL_DIR}"

    ; remove registry entries
    DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Lighthouse"
SectionEnd