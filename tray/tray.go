package tray

/*
#cgo CFLAGS: -Wno-deprecated-declarations
#cgo pkg-config: gtk+-3.0

#include <gtk/gtk.h>
#include "tray.h"
*/
import "C"
import "unsafe"

var (
	activateChan chan struct{}
	quitChan     chan struct{}
)

//export activate
func activate() {
	if activateChan != nil {
		activateChan <- struct{}{}
	}
}

//export quit
func quit() {
	if quitChan != nil {
		quitChan <- struct{}{}
	}
}

// GtkInit is a wrapper around gtk_init.
func GtkInit() {
	C.gtk_init(nil, nil)
}

// GtkMainIterationDo is a wrapper around gtk_main_iteration_do.
func GtkMainIterationDo(blocking bool) bool {
	return gobool(C.gtk_main_iteration_do(gbool(blocking)))
}

func NewTray(statusIconPath string, activate func(), quit func()) {
	// quitItem, err := gtk.MenuItemNewWithLabel("Quit")
	// if err != nil {
	//     return nil, err
	// }

	// menu, err := gtk.MenuNew()
	// if err != nil {
	//     return nil, err
	// }

	// menu.Append(quitItem)
	statusIconPathC := C.CString(statusIconPath)
	defer C.free(unsafe.Pointer(statusIconPathC))

	statusIcon := C.gtk_status_icon_new_from_file((*C.gchar)(statusIconPathC))

	activateSignalC := C.CString("activate")
	defer C.free(unsafe.Pointer(activateSignalC))

	C._g_signal_connect(C.gpointer(statusIcon), activateSignalC, (C.GCallback)(activate), nil)
}

func gbool(b bool) C.gboolean {
	if b {
		return C.gboolean(1)
	}
	return C.gboolean(0)
}

func gobool(b C.gboolean) bool {
	return b != C.FALSE
}
