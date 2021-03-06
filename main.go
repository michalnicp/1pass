package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/golang-ui/nuklear/nk"
	"github.com/michalnicp/1pass/op"
	"github.com/michalnicp/1pass/tray"
	"github.com/pkg/errors"
)

const (
	windowWidth  = 400
	windowHeight = 400

	maxVertexBuffer  = 512 * 1024
	maxElementBuffer = 128 * 1024
)

var (
	mu      sync.Mutex
	session *op.Session
)

func main() {
	runtime.LockOSThread()

	var code int
	defer func() { os.Exit(code) }()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Initialize glfw.
	if err := glfw.Init(); err != nil {
		log.Printf("initialize glfw: %v", err)
		code = 1
		return
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 2)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Floating, glfw.True)
	glfw.WindowHint(glfw.Decorated, glfw.False)
	glfw.WindowHint(glfw.Visible, glfw.False) // Show window after centering it.

	window, err := glfw.CreateWindow(windowWidth, windowHeight, "1pass", nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create window: %v", err)
		code = 1
		return
	}

	window.MakeContextCurrent()

	// Hide the window when it loses focus.
	window.SetFocusCallback(func(w *glfw.Window, focused bool) {
		if !focused {
			window.Hide()
		}
	})

	// Center and show the window.
	centerWindow(window)
	window.Show()

	// Initialize gl.
	if err := gl.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "initialize gl: %v", err)
		code = 1
		return
	}

	// Initialize nuklear.
	ctx := nk.NkPlatformInit(window, nk.PlatformInstallCallbacks)
	atlas := nk.NewFontAtlas()
	nk.NkFontStashBegin(&atlas)
	sansFont := nk.NkFontAtlasAddFromFile(atlas, "assets/FreeSans.ttf", 18, nil)
	nk.NkFontStashEnd()
	if sansFont != nil {
		nk.NkStyleSetFont(ctx, sansFont.Handle())
		font = sansFont
	}

	// Initialize system tray icon.
	tray.Init()

	// Read 1Password config and try to load existing session.
	session, err = op.NewSessionFromConfig()
	if err != nil {
		if cerr := errors.Cause(err); cerr != op.ErrInvalidOPConfig {
			log.Printf("create session: %v", err)
			code = 1
			return
		}
	}

	// Initialize ui state.
	state := NewUIState()

	// Main loop.
	for {

		// Process window events.
		glfw.PollEvents()

		// Run the gtk main loop without blocking.
		tray.Loop()

		// Draw the ui.
		UI(window, ctx, state)

		// Render.
		width, height := window.GetSize()
		gl.Viewport(0, 0, int32(width), int32(height))
		gl.Clear(gl.COLOR_BUFFER_BIT)
		gl.ClearColor(0, 0, 0, 255)
		nk.NkPlatformRender(nk.AntiAliasingOn, maxVertexBuffer, maxElementBuffer)
		window.SwapBuffers()
	}
}
