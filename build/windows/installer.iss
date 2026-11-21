#ifndef AppVersion
  #define AppVersion "0.0.0-dev"
#endif

#define AppName "Frost Flashcard"
#define AppExe "frost-flashcard.exe"
#define AppID "frostflashcard.app"

[Setup]
AppId={{44A7C97A-90AA-4A87-A4BC-F459B446E477}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher=Frost Flashcard
AppPublisherURL=https://github.com/nantapop-kj/frost-flashcard
DefaultDirName={autopf}\{#AppName}
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
OutputDir=..\bin
OutputBaseFilename=Frost-Flashcard-Setup-{#AppVersion}
SetupIconFile=icon.ico
UninstallDisplayIcon={app}\{#AppExe}
UninstallDisplayName={#AppName}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern dark
WizardBackImageFile=wizard-bg-100.png,wizard-bg-150.png,wizard-bg-200.png
WizardBackColor=#12121a
WizardImageFile=
WizardSmallImageFile=..\..\assets\icons\appicon.png
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
CloseApplications=yes

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Messages]
FinishedLabel=Setup has installed [name] on your computer.%n%n[name] runs in the system tray, not in a window. If you cannot see its icon, click ^ on the taskbar, then drag the icon onto the taskbar to keep it visible.

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "..\bin\{#AppExe}"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\{#AppName}"; Filename: "{app}\{#AppExe}"
Name: "{autodesktop}\{#AppName}"; Filename: "{app}\{#AppExe}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#AppExe}"; Description: "{cm:LaunchProgram,{#AppName}}"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "{sys}\taskkill.exe"; Parameters: "/F /IM {#AppExe}"; Flags: runhidden; RunOnceId: "CloseApp"

[Registry]
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueName: "frost-flashcard"; Flags: dontcreatekey uninsdeletevalue

[Code]
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  DataDir: String;
begin
  if CurUninstallStep = usPostUninstall then
  begin
    DataDir := ExpandConstant('{userappdata}\{#AppID}');
    if DirExists(DataDir) then
      if MsgBox('Also delete all your decks and words?' + #13#10 + #13#10 +
                'Choose "No" to keep them for the next time you install.',
                mbConfirmation, MB_YESNO or MB_DEFBUTTON2) = IDYES then
        DelTree(DataDir, True, True, True);
  end;
end;
