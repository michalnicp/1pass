package tray

/*
#cgo CFLAGS: -Wno-deprecated-declarations
#cgo pkg-config: gtk+-3.0

#include "tray.h"
#include <gtk/gtk.h>

*/
import "C"

var (
	Activate func()
	Quit     func()
)

//export activate
func activate(widget *C.GtkWidget, data C.gpointer) {
	Activate()
}

//export quit
func quit(widget *C.GtkWidget, data C.gpointer) {
	Quit()
}

func Init() { C.init() }

func Loop() { C.loop() }
