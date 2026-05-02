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
!include "StrFunc.nsh"

${StrStr}
${StrRep}
${UnStrRep}

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section "Install"
    SetOutPath "$INSTDIR"

    ; copy the executable
    File "cli\lighthouse.exe"

    ; read current user PATH
    ClearErrors
    ReadRegStr $0 HKCU "Environment" "Path"

    ${If} ${Errors}
        MessageBox MB_ICONSTOP "Nao foi possivel ler o PATH do usuario. A instalacao sera interrompida para evitar sobrescrever o PATH."
        Abort
    ${EndIf}

    ; create a backup only once
    ClearErrors
    ReadRegStr $4 HKCU "Environment" "Path_Backup_Before_Lighthouse"

    ${If} ${Errors}
        ClearErrors
        WriteRegExpandStr HKCU "Environment" "Path_Backup_Before_Lighthouse" "$0"
    ${EndIf}

    ; check if install dir already exists as a PATH segment
    StrCpy $1 ";$0;"
    StrCpy $2 ";$INSTDIR;"
    ${StrStr} $3 "$1" "$2"

    ${If} $3 == ""
        ${If} $0 == ""
            StrCpy $0 "$INSTDIR"
        ${Else}
            StrCpy $0 "$0;$INSTDIR"
        ${EndIf}

        WriteRegExpandStr HKCU "Environment" "Path" "$0"
        SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
    ${EndIf}

    ; create uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"

    ; add to Add/Remove Programs
    WriteRegStr HKCU "${UNINSTALL_KEY}" "DisplayName" "${APP_NAME}"
    WriteRegStr HKCU "${UNINSTALL_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegStr HKCU "${UNINSTALL_KEY}" "DisplayVersion" "${APP_VERSION}"
    WriteRegStr HKCU "${UNINSTALL_KEY}" "Publisher" "${APP_NAME}"
    WriteRegStr HKCU "${UNINSTALL_KEY}" "InstallLocation" "$INSTDIR"

    ; run lighthouse up after install, if desired
    ; Exec '"$INSTDIR\${EXE_NAME}" up'
SectionEnd

Section "Uninstall"
    ; stop lighthouse, ignore failures
    ExecWait '"$INSTDIR\${EXE_NAME}" down'

    ; read current user PATH
    ClearErrors
    ReadRegStr $0 HKCU "Environment" "Path"

    ${IfNot} ${Errors}
        ; remove only the exact Lighthouse install directory segment
        StrCpy $1 ";$0;"
        StrCpy $2 ";$INSTDIR;"
        ${UnStrRep} $1 "$1" "$2" ";"

        ; if result is only ";", PATH becomes empty
        ${If} $1 == ";"
            StrCpy $0 ""
        ${Else}
            ; trim first and last semicolon
            StrLen $3 "$1"
            IntOp $3 $3 - 2
            StrCpy $0 "$1" $3 1
        ${EndIf}

        WriteRegExpandStr HKCU "Environment" "Path" "$0"
        SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
    ${EndIf}

    Delete "$INSTDIR\${EXE_NAME}"
    Delete "$INSTDIR\uninstall.exe"
    RMDir "$INSTDIR"

    DeleteRegKey HKCU "${UNINSTALL_KEY}"
SectionEnd

