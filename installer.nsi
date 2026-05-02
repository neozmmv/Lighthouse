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

Function PathContainsEntry
    ; input: $0 = PATH, $2 = entry
    ; output: $3 = 1 if found, 0 otherwise
    Push $1
    Push $4
    Push $5
    Push $6
    Push $7

    StrCpy $3 "0"
    StrCpy $6 "$0"

    ${DoWhile} $6 != ""
        StrLen $7 "$6"
        StrCpy $4 0
        StrCpy $5 ""

        ${Do}
            ${If} $4 >= $7
                StrCpy $5 "$6"
                StrCpy $6 ""
                ${Break}
            ${EndIf}

            StrCpy $1 "$6" 1 $4

            ${If} $1 == ";"
                StrCpy $5 "$6" $4
                IntOp $4 $4 + 1
                StrCpy $6 "$6" "" $4
                ${Break}
            ${EndIf}

            IntOp $4 $4 + 1
        ${Loop}

        ${If} $5 == "$2"
            StrCpy $3 "1"
            ${Break}
        ${EndIf}
    ${Loop}

    Pop $7
    Pop $6
    Pop $5
    Pop $4
    Pop $1
FunctionEnd

Function un.RemovePathEntry
    ; input: $0 = PATH, $2 = entry to remove
    ; output: $0 = updated PATH
    Push $1
    Push $3
    Push $4
    Push $5
    Push $6
    Push $7

    StrCpy $5 ""
    StrCpy $6 "$0"

    ${DoWhile} $6 != ""
        StrLen $7 "$6"
        StrCpy $4 0
        StrCpy $3 ""

        ${Do}
            ${If} $4 >= $7
                StrCpy $3 "$6"
                StrCpy $6 ""
                ${Break}
            ${EndIf}

            StrCpy $1 "$6" 1 $4

            ${If} $1 == ";"
                StrCpy $3 "$6" $4
                IntOp $4 $4 + 1
                StrCpy $6 "$6" "" $4
                ${Break}
            ${EndIf}

            IntOp $4 $4 + 1
        ${Loop}

        ${If} $3 != ""
        ${AndIf} $3 != "$2"
            ${If} $5 == ""
                StrCpy $5 "$3"
            ${Else}
                StrCpy $5 "$5;$3"
            ${EndIf}
        ${EndIf}
    ${Loop}

    StrCpy $0 "$5"

    Pop $7
    Pop $6
    Pop $5
    Pop $4
    Pop $3
    Pop $1
FunctionEnd

Section "Install"
    SetOutPath "$INSTDIR"

    File "cli\lighthouse.exe"

    ClearErrors
    ReadRegStr $0 HKCU "Environment" "Path"

    ${If} ${Errors}
        MessageBox MB_ICONSTOP "Could not read PATH. Installation will be stopped to avoid overwriting PATH."
        Abort
    ${EndIf}

    ClearErrors
    ReadRegStr $4 HKCU "Environment" "Path_Backup_Before_Lighthouse"

    ${If} ${Errors}
        ClearErrors
        WriteRegExpandStr HKCU "Environment" "Path_Backup_Before_Lighthouse" "$0"
    ${EndIf}

    StrCpy $2 "$INSTDIR"
    Call PathContainsEntry

    ${If} $3 != "1"
        ${If} $0 == ""
            StrCpy $0 "$INSTDIR"
        ${Else}
            StrCpy $0 "$0;$INSTDIR"
        ${EndIf}

        WriteRegExpandStr HKCU "Environment" "Path" "$0"
        SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
    ${EndIf}

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

    ClearErrors
    ReadRegStr $0 HKCU "Environment" "Path"

    ${IfNot} ${Errors}
        StrCpy $2 "$INSTDIR"
        Call un.RemovePathEntry

        WriteRegExpandStr HKCU "Environment" "Path" "$0"
        SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
    ${EndIf}

    Delete "$INSTDIR\${EXE_NAME}"
    Delete "$INSTDIR\uninstall.exe"
    RMDir "$INSTDIR"

    DeleteRegKey HKCU "${UNINSTALL_KEY}"
SectionEnd
