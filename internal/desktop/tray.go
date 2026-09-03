package desktop

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	appservice "github.com/headscaleclient/headscaleclient/internal/application"
	"github.com/headscaleclient/headscaleclient/internal/domain"
	wails "github.com/wailsapp/wails/v3/pkg/application"
)

const maxTrayDevices = 8

type trayProfile struct {
	ID         string
	Label      string
	Active     bool
	Switchable bool
}

type trayDevice struct {
	ID             string
	Label          string
	Group          string
	ExitNodeOption bool
}

type trayModel struct {
	Language       domain.Language
	StatusLabel    string
	Connected      bool
	CanToggle      bool
	CanConfigure   bool
	AccountLabel   string
	DeviceLabel    string
	Profiles       []trayProfile
	Devices        []trayDevice
	ExitNodes      []trayDevice
	ExitNodeID     string
	AcceptDNS      bool
	AcceptRoutes   bool
	AllowLANAccess bool
	ShieldsUp      bool
}

type trayRenderState struct {
	Model     trayModel
	Busy      bool
	LastError string
}

type trayLocalizer struct {
	language domain.Language
}

func newTrayLocalizer(language domain.Language) trayLocalizer {
	if !language.Valid() {
		language = domain.LanguageChinese
	}
	return trayLocalizer{language: language}
}

func (l trayLocalizer) text(chinese, english string) string {
	if l.language == domain.LanguageEnglish {
		return english
	}
	return chinese
}

func projectTray(snapshot appservice.AppSnapshot) trayModel {
	localizer := newTrayLocalizer(snapshot.Settings.Language)
	model := trayModel{
		Language:       localizer.language,
		StatusLabel:    localizer.text("正在读取状态", "Reading status"),
		AccountLabel:   localizer.text("未登录", "Not signed in"),
		DeviceLabel:    localizer.text("本机信息不可用", "Local device unavailable"),
		AcceptDNS:      snapshot.Preferences.AcceptDNS,
		AcceptRoutes:   snapshot.Preferences.AcceptRoutes,
		AllowLANAccess: snapshot.Preferences.AllowLANAccess,
		ShieldsUp:      snapshot.Preferences.ShieldsUp,
	}
	if snapshot.Preferences.ExitNodeID != nil {
		model.ExitNodeID = *snapshot.Preferences.ExitNodeID
	}

	switch {
	case snapshot.Runtime.Daemon == "" || snapshot.Runtime.Daemon == "unknown":
		model.StatusLabel = localizer.text("正在读取状态", "Reading status")
	case snapshot.Runtime.Daemon != "ready":
		model.StatusLabel = localizer.text("网络服务不可用", "Network service unavailable")
	case snapshot.Runtime.Session == "login-required" || snapshot.Runtime.Session == "none":
		model.StatusLabel = localizer.text("需要登录", "Sign-in required")
	case snapshot.Runtime.Session == "approval-required":
		model.StatusLabel = localizer.text("等待设备审批", "Awaiting device approval")
	case snapshot.Runtime.Connection == "starting":
		model.StatusLabel = localizer.text("正在连接", "Connecting")
		model.Connected = true
	case snapshot.Runtime.Connection == "stopping":
		model.StatusLabel = localizer.text("正在断开", "Disconnecting")
	case snapshot.Runtime.Connection == "degraded" && snapshot.Runtime.Control == "unreachable":
		model.StatusLabel = localizer.text("隧道已连接 · 控制服务器同步受限", "Tunnel connected · Control sync limited")
		model.Connected = true
	case snapshot.Runtime.Connection == "degraded":
		model.StatusLabel = localizer.text("隧道已连接 · 本地网络警告", "Tunnel connected · Local network warning")
		model.Connected = true
	case snapshot.Runtime.Connection == "running":
		model.StatusLabel = localizer.text("已连接", "Connected")
		model.Connected = true
	default:
		model.StatusLabel = localizer.text("已断开", "Disconnected")
	}
	model.CanToggle = snapshot.Runtime.Daemon == "ready" &&
		snapshot.Runtime.Session == "authenticated" &&
		snapshot.Runtime.Connection != "starting" &&
		snapshot.Runtime.Connection != "stopping"
	model.CanConfigure = snapshot.Runtime.Daemon == "ready" &&
		snapshot.Runtime.Session == "authenticated"

	activeProfileID := ""
	if snapshot.ActiveProfileID != nil {
		activeProfileID = *snapshot.ActiveProfileID
	}
	endpointNames := make(map[string]string, len(snapshot.Endpoints))
	for _, endpoint := range snapshot.Endpoints {
		endpointNames[endpoint.ID] = strings.TrimSpace(endpoint.Name)
	}
	for _, profile := range snapshot.Profiles {
		label := strings.TrimSpace(profile.Account)
		if label == "" {
			label = strings.TrimSpace(profile.DisplayName)
		}
		if label == "" {
			label = localizer.text("未命名账号", "Unnamed account")
		}
		if endpointName := endpointNames[profile.EndpointID]; endpointName != "" {
			label += " · " + endpointName
		}
		switch profile.State {
		case appservice.ProfileStateLoginRequired:
			label += " · " + localizer.text("需登录", "Sign-in required")
		case appservice.ProfileStateApprovalRequired:
			label += " · " + localizer.text("待审批", "Awaiting approval")
		}
		active := profile.ID == activeProfileID
		if active {
			model.AccountLabel = label
		}
		model.Profiles = append(model.Profiles, trayProfile{
			ID:         profile.ID,
			Label:      label,
			Active:     active,
			Switchable: strings.TrimSpace(profile.ID) != "",
		})
	}

	deviceName := strings.TrimSpace(snapshot.LocalDevice.Name)
	if deviceName != "" {
		model.DeviceLabel = deviceName
		if len(snapshot.LocalDevice.Addresses) > 0 {
			model.DeviceLabel += " · " + snapshot.LocalDevice.Addresses[0]
		}
	}
	for _, device := range snapshot.Devices {
		path := localizer.text("离线", "Offline")
		if device.Online {
			path = localizer.text("在线", "Online")
		}
		switch device.ConnectionType {
		case appservice.ConnectionTypeDirect:
			path = localizer.text("直连", "Direct")
		case appservice.ConnectionTypeRelay:
			path = localizer.text("中继", "Relay")
			if device.RelayRegion != "" {
				path += " " + device.RelayRegion
			}
		case appservice.ConnectionTypeOffline:
			path = localizer.text("离线", "Offline")
		case appservice.ConnectionTypeUnknown:
			path = localizer.text("路径未知", "Path unknown")
		}
		address := ""
		if len(device.Addresses) > 0 {
			address = " · " + device.Addresses[0]
		}
		trayDevice := trayDevice{
			ID:             device.ID,
			Label:          fmt.Sprintf("%s%s · %s", device.Name, address, path),
			Group:          strings.TrimSpace(device.Group),
			ExitNodeOption: device.ExitNodeOption,
		}
		if device.Online {
			model.Devices = append(model.Devices, trayDevice)
		}
		if device.ExitNodeOption {
			model.ExitNodes = append(model.ExitNodes, trayDevice)
		}
	}
	return model
}

type TrayControllerOptions struct {
	Icon       []byte
	ShowWindow func()
	Navigate   func(NavigationTarget)
	Quit       func()
}

type TrayController struct {
	app     *wails.App
	service *appservice.Service
	options TrayControllerOptions
	tray    *wails.SystemTray

	mu         sync.Mutex
	snapshot   appservice.AppSnapshot
	busy       bool
	lastError  string
	lastRender *trayRenderState
}

func NewTrayController(app *wails.App, service *appservice.Service, options TrayControllerOptions) *TrayController {
	controller := &TrayController{app: app, service: service, options: options}
	controller.tray = app.SystemTray.New()
	controller.tray.SetIcon(options.Icon)
	controller.tray.SetTooltip("HeadscaleClient · 正在读取状态")
	controller.tray.OnClick(controller.tray.OpenMenu)
	controller.renderLocked()
	return controller
}

func (c *TrayController) Refresh() {
	snapshot, err := c.service.GetSnapshot()
	if err != nil {
		c.setError(err.Error())
		return
	}
	c.ApplySnapshot(snapshot)
}

func (c *TrayController) HandleEvent(name string, payload any) {
	switch name {
	case appservice.EventSnapshotChanged:
		if event, ok := payload.(appservice.SnapshotChangedEvent); ok {
			c.ApplySnapshot(event.Snapshot)
		}
	case appservice.EventLoginFinished:
		if event, ok := payload.(appservice.LoginFinishedEvent); ok {
			c.ApplySnapshot(event.Snapshot)
		}
	case appservice.EventOperationFailed:
		if event, ok := payload.(appservice.OperationFailedEvent); ok {
			c.setError(event.Problem.Message)
		}
	}
}

func (c *TrayController) ApplySnapshot(snapshot appservice.AppSnapshot) {
	c.mu.Lock()
	c.snapshot = snapshot
	c.lastError = ""
	c.renderLocked()
	c.mu.Unlock()
}

func (c *TrayController) setError(message string) {
	c.mu.Lock()
	c.lastError = strings.TrimSpace(message)
	c.busy = false
	c.renderLocked()
	c.mu.Unlock()
}

func (c *TrayController) run(action func() (appservice.AppSnapshot, error)) {
	c.mu.Lock()
	if c.busy {
		c.mu.Unlock()
		return
	}
	c.busy = true
	c.lastError = ""
	c.renderLocked()
	c.mu.Unlock()

	snapshot, err := action()
	if err != nil {
		c.setError(err.Error())
		return
	}

	c.mu.Lock()
	c.busy = false
	c.snapshot = snapshot
	c.renderLocked()
	c.mu.Unlock()
}

func (c *TrayController) renderLocked() {
	model := projectTray(c.snapshot)
	renderState := trayRenderState{Model: model, Busy: c.busy, LastError: c.lastError}
	if c.lastRender != nil && reflect.DeepEqual(*c.lastRender, renderState) {
		return
	}
	localizer := newTrayLocalizer(model.Language)
	menu := c.app.NewMenu()
	menu.Add("HeadscaleClient").SetEnabled(false)
	status := model.StatusLabel
	if c.busy {
		status = localizer.text("正在应用更改", "Applying changes")
	}
	menu.Add(status).SetEnabled(false)
	if c.lastError != "" {
		menu.Add(localizer.text("操作失败 · ", "Operation failed · ") + truncateMenuLabel(c.lastError, 48)).SetEnabled(false)
	}
	menu.AddSeparator()

	connectItem := menu.AddCheckbox(localizer.text("启用连接", "Enable connection"), model.Connected).SetEnabled(model.CanToggle && !c.busy)
	connectItem.OnClick(func(*wails.Context) {
		c.run(func() (appservice.AppSnapshot, error) {
			return c.service.SetConnection(!model.Connected)
		})
	})

	accounts := menu.AddSubmenu(localizer.text("当前账号 · ", "Current account · ") + model.AccountLabel)
	if len(model.Profiles) == 0 {
		accounts.Add(localizer.text("未登录", "Not signed in")).SetEnabled(false)
	} else {
		for _, profile := range model.Profiles {
			profile := profile
			accounts.AddRadio(profile.Label, profile.Active).
				SetEnabled(profile.Switchable && !c.busy).
				OnClick(func(*wails.Context) {
					if profile.Active {
						return
					}
					c.run(func() (appservice.AppSnapshot, error) {
						return c.service.SwitchProfile(profile.ID)
					})
				})
		}
	}
	accounts.AddSeparator()
	accounts.Add(localizer.text("管理账号与控制服务器...", "Manage accounts and control servers...")).OnClick(func(*wails.Context) {
		c.openView(NavigateNetworks)
	})

	menu.Add(localizer.text("本机 · ", "This device · ") + model.DeviceLabel).SetEnabled(false)
	devices := menu.AddSubmenu(localizer.text("在线设备", "Online devices"))
	if len(model.Devices) == 0 {
		devices.Add(localizer.text("没有在线设备", "No online devices")).SetEnabled(false)
	} else {
		limit := len(model.Devices)
		if limit > maxTrayDevices {
			limit = maxTrayDevices
		}
		groupMenus := make(map[string]*wails.Menu)
		for _, device := range model.Devices[:limit] {
			group := device.Group
			if group == "" {
				group = localizer.text("未分组", "Ungrouped")
			}
			groupMenu, ok := groupMenus[group]
			if !ok {
				groupMenu = devices.AddSubmenu(group)
				groupMenus[group] = groupMenu
			}
			groupMenu.Add(device.Label).OnClick(func(*wails.Context) {
				c.openView(NavigateDevices)
			})
		}
		if len(model.Devices) > limit {
			devices.AddSeparator()
			devices.Add(fmt.Sprintf(localizer.text("查看全部 %d 台设备...", "View all %d devices..."), len(model.Devices))).OnClick(func(*wails.Context) {
				c.openView(NavigateDevices)
			})
		}
	}

	exitNodes := menu.AddSubmenu(localizer.text("出口节点", "Exit node"))
	exitNodes.AddRadio(localizer.text("不使用", "Do not use"), model.ExitNodeID == "").
		SetEnabled(model.CanConfigure && !c.busy).
		OnClick(func(*wails.Context) {
			if model.ExitNodeID == "" {
				return
			}
			c.run(func() (appservice.AppSnapshot, error) {
				return c.service.SetExitNode(nil)
			})
		})
	for _, device := range model.ExitNodes {
		device := device
		exitNodes.AddRadio(device.Label, model.ExitNodeID == device.ID).
			SetEnabled(model.CanConfigure && !c.busy).
			OnClick(func(*wails.Context) {
				if model.ExitNodeID == device.ID {
					return
				}
				c.run(func() (appservice.AppSnapshot, error) {
					deviceID := device.ID
					return c.service.SetExitNode(&deviceID)
				})
			})
	}

	preferences := menu.AddSubmenu(localizer.text("网络偏好", "Network preferences"))
	preferences.AddCheckbox(localizer.text("使用 MagicDNS", "Use MagicDNS"), model.AcceptDNS).
		SetEnabled(model.CanConfigure && !c.busy).
		OnClick(func(*wails.Context) {
			c.run(func() (appservice.AppSnapshot, error) {
				return c.service.SetPreference(appservice.PreferenceAcceptDNS, !model.AcceptDNS)
			})
		})
	preferences.AddCheckbox(localizer.text("接受子网路由", "Accept subnet routes"), model.AcceptRoutes).
		SetEnabled(model.CanConfigure && !c.busy).
		OnClick(func(*wails.Context) {
			c.run(func() (appservice.AppSnapshot, error) {
				return c.service.SetPreference(appservice.PreferenceAcceptRoutes, !model.AcceptRoutes)
			})
		})
	preferences.AddCheckbox(localizer.text("允许局域网访问", "Allow LAN access"), model.AllowLANAccess).
		SetEnabled(model.CanConfigure && model.ExitNodeID != "" && !c.busy).
		OnClick(func(*wails.Context) {
			c.run(func() (appservice.AppSnapshot, error) {
				return c.service.SetPreference(appservice.PreferenceAllowLANAccess, !model.AllowLANAccess)
			})
		})
	preferences.AddCheckbox(localizer.text("阻止传入连接", "Block incoming connections"), model.ShieldsUp).
		SetEnabled(model.CanConfigure && !c.busy).
		OnClick(func(*wails.Context) {
			c.run(func() (appservice.AppSnapshot, error) {
				return c.service.SetPreference(appservice.PreferenceShieldsUp, !model.ShieldsUp)
			})
		})

	menu.AddSeparator()
	menu.Add(localizer.text("打开详细设置", "Open detailed settings")).OnClick(func(*wails.Context) {
		c.openView(NavigateOverview)
	})
	menu.Add(localizer.text("退出", "Quit")).OnClick(func(*wails.Context) {
		if c.options.Quit != nil {
			c.options.Quit()
		}
	})

	c.tray.SetTooltip("HeadscaleClient · " + model.StatusLabel + " · " + model.AccountLabel)
	c.tray.SetMenu(menu)
	c.lastRender = &renderState
}

func (c *TrayController) openView(target NavigationTarget) {
	if c.options.ShowWindow != nil {
		c.options.ShowWindow()
	}
	if c.options.Navigate != nil {
		c.options.Navigate(target)
	}
}

func truncateMenuLabel(value string, maximum int) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	return string(runes[:maximum-1]) + "…"
}
