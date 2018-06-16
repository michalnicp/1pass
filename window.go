package main

import "github.com/go-gl/glfw/v3.2/glfw"

func centerWindow(window *glfw.Window) {
	monitor := glfw.GetPrimaryMonitor()
	mode := monitor.GetVideoMode()

	width, height := window.GetSize()
	x := (mode.Width - width) / 2
	y := (mode.Height - height) / 2

	window.SetPos(x, y)
}

func toggleWindow(window *glfw.Window) {
	if window.GetAttrib(glfw.Visible) == glfw.True {
		window.Hide()
	} else {
		window.Show()
	}
}
