package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/golang-ui/nuklear/nk"
	"github.com/michalnicp/1pass/op"
)

const (
	signinText       = "Sign in to your 1Password account"
	bufSize    int32 = 256 * 1024
)

var font *nk.Font

type UIState struct {
	queueChan chan func()

	// Tab.
	id       int32
	activeID int32

	// Signin.
	signinOnce        sync.Once
	signinAddress     []byte
	signinAddressLen  int32
	email             []byte
	emailLen          int32
	secretKey         []byte
	secretKeyLen      int32
	masterPassword    []byte
	masterPasswordLen int32
	isSigningIn       bool

	// Search.
	searchOnce      sync.Once
	searchCancel    context.CancelFunc
	searchQuery     []byte
	searchQueryLen  int32
	items           []op.Item
	searchResults   []op.Item
	selectedItem    *op.Item
	isFetchingItems bool
	isFetchingItem  bool

	// Status.
	statusText string
}

func NewUIState() *UIState {
	state := UIState{
		queueChan: make(chan func(), 10),
		id:        -1,
		activeID:  -1,

		email:          make([]byte, bufSize),
		secretKey:      make([]byte, bufSize),
		masterPassword: make([]byte, bufSize),
		searchQuery:    make([]byte, bufSize),
	}

	if session != nil {
		state.signinAddress = []byte(session.SigninAddress)
		state.signinAddressLen = int32(len(session.SigninAddress))
		state.email = []byte(session.Email)
		state.emailLen = int32(len(session.Email))
		state.secretKey = []byte(session.SecretKey)
		state.secretKeyLen = int32(len(session.SecretKey))
	}

	return &state
}

// tab executes the function if the widget is focused using tab.
func (s *UIState) tab(f func()) {
	s.id++
	if s.activeID == s.id {
		f()
	}
}

// queue adds a function to the queue.
func (s *UIState) queue(f func()) {
	s.queueChan <- f
}

// processUpdates calls each function that has been queued for the main thread.
func (s *UIState) processQueue() {
	for {
		select {
		case f := <-s.queueChan:
			f()
		default:
			return
		}
	}
}

// UI creates a new frame and draws the ui.
func UI(window *glfw.Window, ctx *nk.Context, state *UIState) {
	defer func() {
		if i := recover(); i != nil {
			log.Printf("panic: %v", i)
		}
	}()

	state.processQueue()

	// Handle tab key.
	if window.GetKey(glfw.KeyTab) == glfw.Press {
		state.activeID++
		if state.activeID > state.id {
			state.activeID = 0
		}
	}

	// Reset tab id.
	state.id = -1

	// Handle escape key.
	if window.GetKey(glfw.KeyEscape) == glfw.Press {
		window.Hide()
		return
	}

	// Create a new frame and draw to it.
	nk.NkPlatformNewFrame()

	if session.Valid() {
		Search(window, ctx, state)
	} else {
		Signin(window, ctx, state)
	}
}

// Signin draws the signin view.
func Signin(window *glfw.Window, ctx *nk.Context, state *UIState) {
	submit := func() {
		state.isSigningIn = true

		signinAddress := string(state.signinAddress[:state.signinAddressLen])
		email := string(state.email[:state.emailLen])
		secretKey := string(state.secretKey[:state.secretKeyLen])
		masterPassword := string(state.masterPassword[:state.masterPasswordLen])

		go func() {
			defer state.queue(func() {
				state.isSigningIn = false
			})

			var err error
			session, err = op.Signin(signinAddress, email, secretKey, masterPassword)
			if err != nil {
				log.Printf("signin: %v", err)
				state.queue(func() {
					state.statusText = fmt.Sprintf("signin: %v", err)
				})
				return
			}
		}()
	}

	if window.GetKey(glfw.KeyEnter) == glfw.Press && !state.isSigningIn {
		submit()
	}

	width, height := window.GetSize()
	bounds := nk.NkRect(0, 0, float32(width), float32(height))
	if nk.NkBegin(ctx, "signin", bounds, nk.WindowNoScrollbar) > 0 {
		nk.NkLayoutRowDynamic(ctx, 0, 1)

		nk.NkLabel(ctx, "Sign in to your 1Password account", nk.TextLeft)

		nk.NkLabel(ctx, "Email", nk.TextLeft)
		state.tab(func() {
			nk.NkEditFocus(ctx, nk.EditField|nk.EditGotoEndOnActivate)
		})
		nk.NkEditString(
			ctx,
			nk.EditField,
			state.email,
			&state.emailLen,
			bufSize,
			nk.NkFilterDefault,
		)

		nk.NkLabel(ctx, "Secret Key", nk.TextLeft)
		state.tab(func() {
			nk.NkEditFocus(ctx, nk.EditField|nk.EditGotoEndOnActivate)
		})
		nk.NkEditString(
			ctx,
			nk.EditField,
			state.secretKey,
			&state.secretKeyLen,
			bufSize,
			nk.NkFilterDefault,
		)

		// Mask password with asterisks.
		nk.NkLabel(ctx, "Master Password", nk.TextLeft)
		oldLen := state.masterPasswordLen
		buf := make([]byte, bufSize)
		for i := 0; i < int(state.masterPasswordLen); i++ {
			buf[i] = '*'
		}
		state.tab(func() {
			nk.NkEditFocus(ctx, nk.EditField|nk.EditGotoEndOnActivate)
		})
		state.signinOnce.Do(func() {
			if session != nil &&
				session.Email != "" &&
				session.Token == "" {
				nk.NkEditFocus(ctx, nk.EditField|nk.EditGotoEndOnActivate)
			}
		})
		nk.NkEditString(
			ctx,
			nk.EditField,
			buf,
			&state.masterPasswordLen,
			bufSize,
			nk.NkFilterDefault,
		)
		if oldLen < state.masterPasswordLen {
			copy(state.masterPassword[oldLen:], buf[oldLen:state.masterPasswordLen])
		}

		// Padding.
		nk.NkLayoutRowStatic(ctx, 10, 0, 0)

		nk.NkLayoutRowDynamic(ctx, 30, 1)
		if nk.NkButtonLabel(ctx, "Sign In") > 0 {
			submit()
		}

		nk.NkEnd(ctx)
	}
}

// Search draws the search view.
func Search(window *glfw.Window, ctx *nk.Context, state *UIState) {
	state.searchOnce.Do(func() {
		state.isFetchingItems = true
		go func() {
			defer state.queue(func() {
				state.isFetchingItems = false
			})

			items, err := session.ListItems()
			if err != nil {
				log.Printf("list items: %v", err)
				return
			}

			log.Printf("list items returned %d items", len(items))
			state.queue(func() {
				state.items = items
				state.searchResults = state.items[:]
				state.statusText = fmt.Sprintf("%d results", len(items))
			})
		}()
	})

	width, height := window.GetSize()
	bounds := nk.NkRect(0, 0, float32(width), float32(height))
	if nk.NkBegin(ctx, "search", bounds, nk.WindowNoScrollbar) > 0 {
		region := nk.NkWindowGetContentRegion(ctx)

		nk.NkLayoutSpaceBegin(ctx, nk.Static, 0, 3)

		bounds := nk.NkLayoutWidgetBounds(ctx)

		// Copy the current search query to check if it changed.
		searchQuery := make([]byte, state.searchQueryLen)
		copy(searchQuery, state.searchQuery)

		nk.NkLayoutSpacePush(ctx, nk.NkRect(0, 0, bounds.W(), bounds.H()))

		state.tab(func() {
			nk.NkEditFocus(ctx, nk.EditField|nk.EditGotoEndOnActivate)
		})
		nk.NkEditString(
			ctx,
			nk.EditField,
			state.searchQuery,
			&state.searchQueryLen,
			bufSize,
			nk.NkFilterDefault,
		)

		if !bytes.Equal(searchQuery, state.searchQuery[:state.searchQueryLen]) {
			state.selectedItem = nil
			if state.isFetchingItems {

				// Cancel the previous search.
				state.searchCancel()
			}

			state.isFetchingItems = true
			ctx, cancel := context.WithCancel(context.Background())
			state.searchCancel = cancel

			query := string(state.searchQuery[:state.searchQueryLen])
			if query == "" {
				state.searchResults = state.items[:]
				state.isFetchingItems = false
			} else {
				go func() {
					defer state.queue(func() {
						select {
						case <-ctx.Done():
						default:
							state.isFetchingItems = false
						}
					})

					results, err := session.SearchItems(query)
					if err != nil {
						log.Printf("search items: %v", err)
						return
					}

					state.queue(func() {
						select {
						case <-ctx.Done():
						default:
							state.searchResults = results
							state.statusText = fmt.Sprintf("matched %d results", len(state.searchResults))
						}
					})
				}()
			}
		}

		// Search results item list.
		nk.NkLayoutSpacePush(ctx, nk.NkRect(0, bounds.H()+4, bounds.W(), region.H()-bounds.H()-20))

		state.tab(func() {
			if len(state.searchResults) == 0 {
				state.id--
				return
			}

			// TODO: Focus the results list and handle up down keys to navigate list.
		})

		nk.SetGroupPadding(ctx, nk.NkVec2(0, 0))
		if nk.NkGroupBegin(ctx, "items", nk.WindowScrollAutoHide) > 0 {
			for _, item := range state.searchResults {
				searchResultItem(window, ctx, state, item)
			}
			nk.NkGroupEnd(ctx)
		}

		nk.NkLayoutSpacePush(ctx, nk.NkRect(0, region.H()-28, bounds.W(), 28))

		StatusLine(window, ctx, state)

		nk.NkLayoutSpaceEnd(ctx)

		nk.NkEnd(ctx)
	}
}

func searchResultItem(window *glfw.Window, ctx *nk.Context, state *UIState, item op.Item) {
	if state.selectedItem != nil && state.selectedItem.UUID == item.UUID {

		// Show item details
		nk.NkLayoutRowDynamic(ctx, 95, 1)

		nk.SetGroupPadding(ctx, nk.NkVec2(0, 0))
		if nk.NkGroupBegin(ctx, "", 0) > 0 {
			nk.NkLayoutRowDynamic(ctx, 0, 1)
			nk.NkLabel(ctx, item.Overview.Title, nk.TextLeft)

			if state.selectedItem.Details != nil {

				// Display the username and password.
				var username string
				for _, field := range state.selectedItem.Details.Fields {
					if field.Designation == "username" {
						username = field.Value
						break
					}
				}
				var password string
				for _, field := range state.selectedItem.Details.Fields {
					if field.Designation == "password" {
						password = field.Value
						break
					}
				}

				nk.NkLayoutRowDynamic(ctx, 40, 2)
				if CopyButton(ctx, "username", username) > 0 {
					if err := writeClipboard(username); err != nil {
						log.Printf("copy username: %v", err)
					} else {
						log.Println("username copied")
					}
				}
				if CopyButton(ctx, "password", "********") > 0 {
					if err := writeClipboard(password); err != nil {
						log.Printf("copy password: %v", err)
					} else {
						log.Println("password copied")
					}
				}
			}

			nk.NkGroupEnd(ctx)
		}

		return
	}

	nk.NkLayoutRowDynamic(ctx, 0, 1)

	if nk.NkSelectLabel(ctx, item.Overview.Title, nk.TextLeft, 0) > 0 {
		state.isFetchingItem = true
		state.selectedItem = &item

		go func() {
			defer state.queue(func() {
				state.isFetchingItem = false
			})

			// Get item details.
			item, err := session.GetItem(item.UUID)
			if err != nil {
				log.Printf("get item: %v", err)
				return
			}

			state.queue(func() {
				state.selectedItem = item
			})
		}()
	}
}

// StatusLine draws the status line.
func StatusLine(window *glfw.Window, ctx *nk.Context, state *UIState) {
	nk.NkLabel(ctx, state.statusText, nk.TextLeft)
}

func CopyButton(ctx *nk.Context, text1, text2 string) int32 {
	out := nk.NkWindowGetCanvas(ctx)
	r := nk.NkWidgetBounds(ctx)

	ret := nk.NkButtonColor(ctx, nk.NkRgb(114, 105, 185))

	bounds := nk.NkRect(r.X()+4, r.Y(), r.W()-4, r.H()/2)
	nk.NkDrawText(out, bounds, text1, int32(len(text1)), font.Handle(), nk.NkRgb(188, 174, 118), nk.NkRgb(0, 0, 0))

	bounds = nk.NkRect(r.X()+4, r.Y()+r.H()/2, r.W()-4, r.H()/2)
	nk.NkDrawText(out, bounds, text2, int32(len(text2)), font.Handle(), nk.NkRgb(188, 174, 118), nk.NkRgb(0, 0, 0))

	return ret
}

/*
func CopyButton(ctx *nk.Context, text1, text2, copyText string) {
	out := nk.NkWindowGetCanvas(ctx)
	r := nk.NewRect()

	nk.NkWidget(r, ctx)

	// Draw button.
	// nk.NkFillRect(out, *r, 5, nk.NkRgb(114, 105, 185))
	nk.NkButtonColor(ctx, nk.NkRgb(114, 105, 185))

	bounds := nk.NkRect(r.X()+4, r.Y(), r.W()-4, r.H()/2)
	nk.NkDrawText(out, bounds, text1, int32(len(text1)), font.Handle(), nk.NkRgb(188, 174, 118), nk.NkRgb(0, 0, 0))

	bounds = nk.NkRect(r.X()+4, r.Y()+r.H()/2, r.W()-4, r.H()/2)
	nk.NkDrawText(out, bounds, text2, int32(len(text2)), font.Handle(), nk.NkRgb(188, 174, 118), nk.NkRgb(0, 0, 0))
}
*/
