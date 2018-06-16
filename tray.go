package main

import (
	"github.com/gotk3/gotk3/gtk"
)

type Tray struct {
	statusIcon *gtk.StatusIcon
}

func NewTray(iconName string, activate func()) (*Tray, error) {
	statusIcon, err := gtk.StatusIconNewFromIconName(iconName)
	if err != nil {
		return nil, err
	}
	statusIcon.Connect("activate", activate)

	tray := Tray{
		statusIcon: statusIcon,
	}

	return &tray, nil
}
