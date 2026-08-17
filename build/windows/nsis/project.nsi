Unicode true

####
## Please note: Template replacements don't work in this file. They are provided with default defines like
## mentioned underneath.
## If the keyword is not defined, "wails_tools.nsh" will populate them.
## If they are defined here, "wails_tools.nsh" will not touch them. This allows you to use this project.nsi manually
## from outside of Wails for debugging and development of the installer.
## 
## For development first make a wails nsis build to populate the "wails_tools.nsh":
## > wails build --target windows/amd64 --nsis
## Then you can call makensis on this file with specifying the path to your binary:
## For a AMD64 only installer:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app.exe
## For a ARM64 only installer:
## > makensis -DARG_WAILS_ARM64_BINARY=..\..\bin\app.exe
## For a installer with both architectures:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app-amd64.exe -DARG_WAILS_ARM64_BINARY=..\..\bin\app-arm64.exe
####
## The following information is taken from the wails_tools.nsh file, but they can be overwritten here.
####
## !define INFO_PROJECTNAME    "my-project" # Default "HeadscaleClient"
## !define INFO_COMPANYNAME    "My Company" # Default "HeadscaleClient Contributors"
## !define INFO_PRODUCTNAME    "My Product Name" # Default "HeadscaleClient"
## !define INFO_PRODUCTVERSION "1.0.0"     # Default "0.1.0"
## !define INFO_COPYRIGHT      "(c) Now, My Company" # Default "© 2026, My Company"
###
## !define PRODUCT_EXECUTABLE  "Application.exe"      # Default "${INFO_PROJECTNAME}.exe"
## !define UNINST_KEY_NAME     "UninstKeyInRegistry"  # Default "${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"
####
## !define REQUEST_EXECUTION_LEVEL "admin"            # Default "admin"  see also https://nsis.sourceforge.io/Docs/Chapter4.html
## !define WAILS_INSTALL_SCOPE     "user"             # Default "machine" - set to "user" for per-user install ($LOCALAPPDATA) without UAC prompt
####
## Include the wails tools
####
!include "wails_tools.nsh"

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"
!include "LogicLib.nsh"

!ifndef ARG_DAEMON_DIR
    !error "ARG_DAEMON_DIR must point to a verified managed-daemon payload"
!endif

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
# !define MUI_WELCOMEFINISHPAGE_BITMAP "resources\leftimage.bmp" #Include this to add a bitmap on the left side of the Welcome Page. Must be a size of 164x314
!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can take a look into the details of the installation steps
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.
!define MUI_LANGDLL_ALWAYSSHOW
!define MUI_LANGDLL_WINDOWTITLE "选择安装语言 / Select Setup Language"
!define MUI_LANGDLL_INFO "请选择安装语言。 / Please select a language."
!define MUI_LANGDLL_REGISTRY_ROOT "HKLM"
!define MUI_LANGDLL_REGISTRY_KEY "${UNINST_KEY}"
!define MUI_LANGDLL_REGISTRY_VALUENAME "InstallerLanguage"
!define INSTALL_VENDOR_DIRECTORY "BIMCC"
!define LEGACY_INSTALL_DIRECTORY "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"

!insertmacro MUI_PAGE_WELCOME # Welcome to the installer page.
# !insertmacro MUI_PAGE_LICENSE "resources\eula.txt" # Adds a EULA page to the installer
!insertmacro MUI_PAGE_DIRECTORY # In which folder install page.
!insertmacro MUI_PAGE_INSTFILES # Installing page.
!insertmacro MUI_PAGE_FINISH # Finished installation page.

!insertmacro MUI_UNPAGE_INSTFILES # Uninstalling page

!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

LangString WindowsVersionRequired ${LANG_SIMPCHINESE} "本产品仅支持 Windows 10（Server 2016）及更高版本。"
LangString WindowsVersionRequired ${LANG_ENGLISH} "This product is only supported on Windows 10 (Server 2016) and later."
LangString ArchitectureNotSupported ${LANG_SIMPCHINESE} "当前 Windows 架构不受支持。支持的架构：${ARCH}"
LangString ArchitectureNotSupported ${LANG_ENGLISH} "This product can't be installed on the current Windows architecture. Supports: ${ARCH}"
LangString ExistingInstallPrompt ${LANG_SIMPCHINESE} "检测到已安装的 HeadscaleClient。继续将更新或修复现有安装，并保留账号与网络配置。是否继续？"
LangString ExistingInstallPrompt ${LANG_ENGLISH} "HeadscaleClient is already installed. Continuing will update or repair the installation while preserving account and network configuration. Continue?"
LangString ServiceInstalling ${LANG_SIMPCHINESE} "正在安装 HeadscaleClient 托管网络服务"
LangString ServiceInstalling ${LANG_ENGLISH} "Installing the HeadscaleClient managed network service"
LangString ServiceRepairing ${LANG_SIMPCHINESE} "正在修复 HeadscaleClient 托管网络服务"
LangString ServiceRepairing ${LANG_ENGLISH} "Repairing the HeadscaleClient managed network service"
LangString ServiceMigrating ${LANG_SIMPCHINESE} "正在将 HeadscaleClient 托管网络服务迁移到新目录"
LangString ServiceMigrating ${LANG_ENGLISH} "Migrating the HeadscaleClient managed network service to the new directory"
LangString ServiceInstallFailed ${LANG_SIMPCHINESE} "无法安装网络服务（退出代码 $1）。"
LangString ServiceInstallFailed ${LANG_ENGLISH} "The network service could not be installed (exit code $1)."
LangString ServiceRepairFailed ${LANG_SIMPCHINESE} "无法修复网络服务（退出代码 $1）。请关闭手工运行的 tailscaled.exe 后重试。"
LangString ServiceRepairFailed ${LANG_ENGLISH} "The network service could not be repaired (exit code $1). Close any manually started tailscaled.exe process and try again."
LangString ServiceMigrationFailed ${LANG_SIMPCHINESE} "无法迁移旧版网络服务（退出代码 $1）。请先卸载旧版本后重试。"
LangString ServiceMigrationFailed ${LANG_ENGLISH} "The previous network service could not be migrated (exit code $1). Uninstall the previous version and try again."
LangString ServiceStartFailed ${LANG_SIMPCHINESE} "网络服务未能启动（退出代码 $1）。请关闭手工运行的 tailscaled.exe 后重试。"
LangString ServiceStartFailed ${LANG_ENGLISH} "The network service could not be started (exit code $1). Close any manually started tailscaled.exe process and try again."
LangString ServiceStarting ${LANG_SIMPCHINESE} "正在确认 Tailscale 网络服务状态"
LangString ServiceStarting ${LANG_ENGLISH} "Checking the Tailscale network service"
LangString ServiceStarted ${LANG_SIMPCHINESE} "Tailscale 网络服务已启动"
LangString ServiceStarted ${LANG_ENGLISH} "The Tailscale network service has started"
LangString ServiceAlreadyRunning ${LANG_SIMPCHINESE} "Tailscale 网络服务已在运行"
LangString ServiceAlreadyRunning ${LANG_ENGLISH} "The Tailscale network service is already running"
LangString ServiceReusing ${LANG_SIMPCHINESE} "正在复用现有的 Tailscale 网络服务"
LangString ServiceReusing ${LANG_ENGLISH} "Reusing the existing Tailscale network service"
LangString ServiceRemoving ${LANG_SIMPCHINESE} "正在移除 HeadscaleClient 托管网络服务"
LangString ServiceRemoving ${LANG_ENGLISH} "Removing the HeadscaleClient managed network service"
LangString ServiceLeavingExternal ${LANG_SIMPCHINESE} "保留外部 Tailscale 网络服务，不做更改"
LangString ServiceLeavingExternal ${LANG_ENGLISH} "Leaving the external Tailscale network service unchanged"

!define WAILS_WIN10_REQUIRED "$(WindowsVersionRequired)"
!define WAILS_ARCHITECTURE_NOT_SUPPORTED "$(ArchitectureNotSupported)"

## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
!if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
!else
    InstallDir "$PROGRAMFILES64\${INSTALL_VENDOR_DIRECTORY}\${INFO_PRODUCTNAME}"
!endif
ShowInstDetails show # This will always show the installation details.

Function .onInit
   !insertmacro MUI_LANGDLL_DISPLAY
   !insertmacro wails.checkArchitecture
   SetRegView 64
   ReadRegStr $0 HKLM "${UNINST_KEY}" "UninstallString"
   ${If} $0 != ""
       IfSilent continueExistingInstall 0
       MessageBox MB_ICONQUESTION|MB_OKCANCEL "$(ExistingInstallPrompt)" IDOK continueExistingInstall
       Abort
       continueExistingInstall:
   ${EndIf}
FunctionEnd

Function un.onInit
   !insertmacro MUI_UNGETLANGUAGE
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR
    
    !insertmacro wails.files

    ReadRegStr $0 HKLM "SYSTEM\CurrentControlSet\Services\Tailscale" "ImagePath"
    StrCpy $2 '"${LEGACY_INSTALL_DIRECTORY}\daemon\tailscaled.exe"'
    StrCpy $3 '${LEGACY_INSTALL_DIRECTORY}\daemon\tailscaled.exe'
    StrCpy $5 '"$INSTDIR\daemon\tailscaled.exe"'
    StrCpy $6 '$INSTDIR\daemon\tailscaled.exe'
    StrCpy $4 "0"
    ${If} $0 == ""
        StrCpy $4 "1"
    ${ElseIf} $0 == $2
    ${OrIf} $0 == $3
        DetailPrint "$(ServiceMigrating)"
        nsExec::ExecToLog '"${LEGACY_INSTALL_DIRECTORY}\daemon\tailscaled.exe" uninstall-system-daemon'
        Pop $1
        ${If} $1 != 0
            MessageBox MB_ICONSTOP "$(ServiceMigrationFailed)"
            Abort
        ${EndIf}
        IfFileExists "${LEGACY_INSTALL_DIRECTORY}\uninstall.exe" 0 +2
            ExecWait '"${LEGACY_INSTALL_DIRECTORY}\uninstall.exe" /S'
        StrCpy $4 "1"
    ${ElseIf} $0 == $5
    ${OrIf} $0 == $6
        DetailPrint "$(ServiceRepairing)"
        nsExec::ExecToLog '"$INSTDIR\daemon\tailscaled.exe" uninstall-system-daemon'
        Pop $1
        ${If} $1 != 0
            MessageBox MB_ICONSTOP "$(ServiceRepairFailed)"
            Abort
        ${EndIf}
        StrCpy $4 "1"
    ${Else}
        DetailPrint "$(ServiceReusing)"
    ${EndIf}

    SetOutPath "$INSTDIR\daemon"
    File "${ARG_DAEMON_DIR}\tailscaled.exe"
    File "${ARG_DAEMON_DIR}\tailscale.exe"
    File "${ARG_DAEMON_DIR}\wintun.dll"
    File "${ARG_DAEMON_DIR}\provenance.json"
    SetOutPath "$INSTDIR\daemon\licenses"
    File "${ARG_DAEMON_DIR}\licenses\TAILSCALE-LICENSE.txt"
    File "${ARG_DAEMON_DIR}\licenses\WINTUN-PREBUILT-LICENSE.txt"
    SetOutPath $INSTDIR

    ${If} $4 == "1"
        DetailPrint "$(ServiceInstalling)"
        nsExec::ExecToLog '"$INSTDIR\daemon\tailscaled.exe" install-system-daemon'
        Pop $1
        ${If} $1 != 0
            MessageBox MB_ICONSTOP "$(ServiceInstallFailed)"
            Abort
        ${EndIf}
    ${EndIf}

    DetailPrint "$(ServiceStarting)"
    nsExec::Exec '"$SYSDIR\sc.exe" start Tailscale'
    Pop $1
    ${If} $1 == 0
        DetailPrint "$(ServiceStarted)"
    ${ElseIf} $1 == 1056
        DetailPrint "$(ServiceAlreadyRunning)"
    ${Else}
        MessageBox MB_ICONSTOP "$(ServiceStartFailed)"
        Abort
    ${EndIf}

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols
    
    !insertmacro wails.writeUninstaller

    SetRegView 64
    ${If} $LANGUAGE == ${LANG_SIMPCHINESE}
        WriteRegStr HKLM "${UNINST_KEY}" "DefaultLanguage" "zh-CN"
    ${Else}
        WriteRegStr HKLM "${UNINST_KEY}" "DefaultLanguage" "en-US"
    ${EndIf}
SectionEnd

Section "uninstall" 
    !insertmacro wails.setShellContext

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # Remove the WebView2 DataPath

    ReadRegStr $0 HKLM "SYSTEM\CurrentControlSet\Services\Tailscale" "ImagePath"
    StrCpy $1 '"$INSTDIR\daemon\tailscaled.exe"'
    ${If} $0 == $1
        DetailPrint "$(ServiceRemoving)"
        nsExec::ExecToLog '"$INSTDIR\daemon\tailscaled.exe" uninstall-system-daemon'
        Pop $2
    ${ElseIf} $0 == "$INSTDIR\daemon\tailscaled.exe"
        DetailPrint "$(ServiceRemoving)"
        nsExec::ExecToLog '"$INSTDIR\daemon\tailscaled.exe" uninstall-system-daemon'
        Pop $2
    ${Else}
        DetailPrint "$(ServiceLeavingExternal)"
    ${EndIf}

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
SectionEnd
