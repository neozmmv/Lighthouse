#define AppName "Lighthouse"
#define AppVersion "1.0.0-win"
#define ExeName "lighthouse.exe"
#define TrayExeName "lighthouse-tray.exe"

[Setup]
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppName}
DefaultDirName={localappdata}\Lighthouse
DisableDirPage=no
OutputBaseFilename=LighthouseSetup
OutputDir=.
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=lowest
UninstallDisplayName={#AppName}
ChangesEnvironment=yes

[Components]
Name: "startup"; Description: "Start Lighthouse Tray automatically with Windows"; Types: full compact custom

[Files]
Source: "cli\lighthouse.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "tray\src-tauri\target\release\lighthouse-tray.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{userprograms}\Lighthouse\Lighthouse Tray"; Filename: "{app}\{#TrayExeName}"
Name: "{userstartup}\Lighthouse Tray"; Filename: "{app}\{#TrayExeName}"; Components: startup

[Run]
Filename: "{app}\{#TrayExeName}"; Description: "Start Lighthouse Tray"; Flags: postinstall nowait skipifsilent

[UninstallRun]
Filename: "taskkill"; Parameters: "/F /IM {#TrayExeName}"; Flags: runhidden waituntilterminated

[Code]
procedure AddToPath();
var
  PS, TmpFile: String;
  ResultCode: Integer;
begin
  TmpFile := ExpandConstant('{tmp}\lighthouse-path-install.ps1');
  PS :=
    '$ErrorActionPreference = ''Stop''' + #13#10 +
    '$dir = $args[0]' + #13#10 +
    '$path = [Environment]::GetEnvironmentVariable(''Path'', ''User'')' + #13#10 +
    'if ($null -eq $path) { $path = '''' }' + #13#10 +
    '$items = @($path -split '';'' | Where-Object { $_ -ne '''' })' + #13#10 +
    '$exists = $false' + #13#10 +
    'foreach ($item in $items) {' + #13#10 +
    '  if ($item.TrimEnd([char]92) -ieq $dir.TrimEnd([char]92)) { $exists = $true; break }' + #13#10 +
    '}' + #13#10 +
    'if (-not $exists) {' + #13#10 +
    '  $backup = [Environment]::GetEnvironmentVariable(''Path_Backup_Before_Lighthouse'', ''User'')' + #13#10 +
    '  if ($null -eq $backup) {' + #13#10 +
    '    [Environment]::SetEnvironmentVariable(''Path_Backup_Before_Lighthouse'', $path, ''User'')' + #13#10 +
    '  }' + #13#10 +
    '  $items = @($items) + $dir' + #13#10 +
    '  [Environment]::SetEnvironmentVariable(''Path'', ($items -join '';''), ''User'')' + #13#10 +
    '}' + #13#10 +
    'exit 0';
  SaveStringToFile(TmpFile, PS, False);
  if not Exec('powershell.exe',
              '-NoProfile -ExecutionPolicy Bypass -File "' + TmpFile + '" "' +
              ExpandConstant('{app}') + '"',
              '', SW_HIDE, ewWaitUntilTerminated, ResultCode) or (ResultCode <> 0) then
  begin
    MsgBox('Could not update user PATH.', mbError, MB_OK);
    Abort;
  end;
end;

procedure RemoveFromPath();
var
  PS, TmpFile: String;
  ResultCode: Integer;
begin
  TmpFile := ExpandConstant('{tmp}\lighthouse-path-uninstall.ps1');
  PS :=
    '$ErrorActionPreference = ''Stop''' + #13#10 +
    '$dir = $args[0]' + #13#10 +
    '$path = [Environment]::GetEnvironmentVariable(''Path'', ''User'')' + #13#10 +
    'if ($null -ne $path) {' + #13#10 +
    '  $items = @($path -split '';'' | Where-Object { $_ -ne '''' -and $_.TrimEnd([char]92) -ine $dir.TrimEnd([char]92) })' + #13#10 +
    '  [Environment]::SetEnvironmentVariable(''Path'', ($items -join '';''), ''User'')' + #13#10 +
    '}' + #13#10 +
    'exit 0';
  SaveStringToFile(TmpFile, PS, False);
  Exec('powershell.exe',
       '-NoProfile -ExecutionPolicy Bypass -File "' + TmpFile + '" "' +
       ExpandConstant('{app}') + '"',
       '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
    AddToPath();
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  ResultCode: Integer;
begin
  if CurUninstallStep = usUninstall then
  begin
    Exec(ExpandConstant('{app}\lighthouse.exe'), 'down', '',
         SW_HIDE, ewWaitUntilTerminated, ResultCode);
    RemoveFromPath();
  end;
end;