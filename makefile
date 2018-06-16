build:
	go build -tags gtk_3_12

install:
	mkdir -p /usr/share/icons/hicolor/scalable/apps
	cp assets/1pass-lock.svg /usr/share/icons/hicolor/scalable/apps
	gtk-update-icon-cache -f -t /usr/share/icons/hicolor

# watch .go files for changes and restart.
run:
	@pt -g='\.(go|c|h)$$' | entr -rs '$(MAKE) build && ./1pass'

.PHONY: build install run
