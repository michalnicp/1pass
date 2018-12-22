#pragma once

#include <gtk/gtk.h>

extern void activate(GtkWidget *widget, gpointer data);
extern void quit(GtkWidget *widget, gpointer data);

static void status_icon_popup_menu(GtkStatusIcon *status_icon, guint button, guint activation_time, GtkWidget *menu) {
    gtk_menu_popup(GTK_MENU(menu), NULL, NULL, gtk_status_icon_position_menu, status_icon, button, activation_time);
}

static void init() {
    gtk_init(0, NULL);

    // Create context menu.
    GtkWidget *menu = gtk_menu_new();

    GtkWidget *quit_item = gtk_menu_item_new_with_label("Quit");
    gtk_menu_shell_append(GTK_MENU_SHELL(menu), quit_item);
    g_signal_connect(quit_item, "activate", G_CALLBACK(quit), NULL);

    gtk_widget_show_all(menu);

    // Create system tray icon.
    GtkStatusIcon *status_icon = gtk_status_icon_new_from_icon_name("1pass");
    g_signal_connect(status_icon, "activate", G_CALLBACK(activate), NULL);
    g_signal_connect(status_icon, "popup-menu", G_CALLBACK(status_icon_popup_menu), menu);
}

static void loop() {
    gtk_main_iteration_do(FALSE);
}
