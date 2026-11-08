[Setup]
AppId=org.redmine-go.desktop
AppName=redmineapp
AppVersion=0.1.0
DefaultDirName={autopf}\redmineapp
DefaultGroupName=redmineapp
OutputDir=..\dist
OutputBaseFilename=redmineapp-windows-amd64-setup
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
[Files]
Source: "..\dist\redmineapp.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\NOTICE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\THIRD_PARTY_NOTICES.txt"; DestDir: "{app}"; Flags: ignoreversion
[Icons]
Name: "{group}\redmineapp"; Filename: "{app}\redmineapp.exe"
Name: "{autodesktop}\redmineapp"; Filename: "{app}\redmineapp.exe"; Tasks: desktopicon
[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"
