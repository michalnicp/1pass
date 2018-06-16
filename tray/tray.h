#pragma once

#include <glib.h>
#include <glib-object.h>

extern void activate();
extern void quit();

void create_tray();

static void _g_signal_connect(gpointer instance, const gchar *detailed_signal, GCallback c_handler, gpointer data) {
    g_signal_connect(instance, detailed_signal, c_handler, data);
}
