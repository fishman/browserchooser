package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/atotto/clipboard"
	"github.com/jeandeaual/go-locale"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/skip2/go-qrcode"
	"golang.org/x/text/language"
)

//go:embed messages
var messagesFS embed.FS

func main() {
	app.SetMetadata(fyne.AppMetadata{
		ID:         appID,
		Migrations: map[string]bool{"fyneDo": true},
	})

	url := ""
	for _, a := range os.Args[1:] {
		switch a {
		case "--set-default":
			if err := setDefault(); err != nil {
				log.Fatalf("browserchooser: set default: %v", err)
			}
			return
		case "-h", "--help":
			usage()
			return
		default:
			if url == "" {
				url = a
			}
		}
	}

	a := app.NewWithID(appID)

	if url != "" {
		if b, ok := matchRule(url); ok {
			if err := b.open(url); err != nil {
				log.Printf("browserchooser: open %s: %v", b.name, err)
			} else {
				recordOpen(url, b.id)
			}
			return
		}
	}

	w := a.NewWindow(winTitle)
	u := newUI(a, w, url)
	u.applyTheme()
	w.SetContent(u.content())
	w.Resize(fyne.NewSize(winW, windowHeight()))
	w.SetFixedSize(true)
	w.CenterOnScreen()
	u.refresh()
	u.focusEntry()
	w.ShowAndRun()
}

type appUI struct {
	a        fyne.App
	w        fyne.Window
	url      string
	browsers []browser
	stats    map[string]useStat
	entry    *numEntry
	list     *widget.List
	rows     []row
	loc      *i18n.Localizer
	dark     bool
	showQR   bool
	selTimer *time.Timer
	hostTop  string
}

func newUI(a fyne.App, w fyne.Window, url string) *appUI {
	u := &appUI{
		a: a, w: w, url: url, loc: newLocalizer(),
		browsers: detectBrowsers(),
		dark:     true, // resolved to the effective variant in applyTheme
	}
	if len(u.browsers) == 0 {
		u.browsers = []browser{fallbackBrowser()}
	}
	u.stats = loadStats(u.browsers)
	if h := hostOf(url); h != "" {
		u.hostTop = dominantHostID(loadState().Hosts[h])
	}
	w.SetTitle(u.l(msgTitle))

	u.entry = &numEntry{}
	u.entry.Scroll = fyne.ScrollNone // clip text, no scrollbars
	u.entry.SetPlaceHolder(u.l(msgPlaceholder))
	if u.hostTop != "" {
		if b := findBrowser(u.browsers, u.hostTop); b != nil {
			u.entry.SetPlaceHolder(fmt.Sprintf(u.l(msgHostTop), b.name, hostOf(url)))
		}
	}
	u.entry.OnChanged = func(string) { u.refresh() }
	u.entry.OnSubmitted = func(string) {
		if len(u.rows) > 0 {
			u.activate(0)
		} else if u.copyVisible() {
			u.activate(u.copyIndex())
		}
	}
	u.entry.onSelect = func(i int) {
		if i == 4 {
			if u.url != "" {
				u.copyAndQuit()
			}
			return
		}
		u.activate(i)
	}
	u.entry.onEsc = func() { u.a.Quit() }
	u.entry.onCtrlC = func() { u.copyAndQuit() }

	u.list = widget.NewList(
		func() int { return u.listLen() },
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromResource(nil)
			icon.FillMode = canvas.ImageFillContain
			icon.SetMinSize(fyne.NewSize(24, 24))
			return container.NewBorder(nil, nil, container.NewPadded(icon), container.NewCenter(u.newChip()), widget.NewLabel(""))
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			i := int(id)
			border := o.(*fyne.Container)
			img := border.Objects[1].(*fyne.Container).Objects[0].(*canvas.Image)
			img.Resource = u.rowIcon(i)
			img.Hidden = img.Resource == nil
			img.Refresh() // re-raster from the new resource so the icon tracks its row
			border.Objects[0].(*widget.Label).SetText(u.rowLabel(i))
			chip := border.Objects[2].(*fyne.Container).Objects[0].(*fyne.Container)
			chip.Hidden = false
			u.updateChip(chip, u.chipNumber(i))
		},
	)
	u.list.OnSelected = func(id widget.ListItemID) {
		u.activate(int(id))
		u.list.UnselectAll()
	}
	return u
}

func (u *appUI) content() fyne.CanvasObject {
	var bottom fyne.CanvasObject
	if u.url != "" {
		btn := newQRButton(u.l(msgShowQR), u.toggleQR, u.dark)
		bottom = btn
		if u.showQR {
			if qr := u.qr(); qr != nil {
				bottom = container.NewVBox(qr, btn)
			}
		}
	}
	bar := container.NewBorder(u.entry, bottom, nil, nil, u.list)
	min := canvas.NewRectangle(color.Transparent)
	min.SetMinSize(fyne.NewSize(winW, 0))
	return container.NewStack(min, bar)
}

// toggleQR flips the QR overlay on and rebuilds the window so the code shows at
// the bottom, above the always-sticky Show QR button.
func (u *appUI) toggleQR() {
	u.showQR = !u.showQR
	u.w.SetContent(u.content())
	u.focusEntry()
}

func (u *appUI) qr() fyne.CanvasObject {
	png, err := qrcode.Encode(u.url, qrcode.Medium, qrSize)
	if err != nil {
		return nil
	}
	img := canvas.NewImageFromReader(bytes.NewReader(png), "qr.png")
	img.FillMode = canvas.ImageFillOriginal
	return img
}

// qrButton is a compact tappable bar (slot 7, ~2/3 of a browser row) that toggles
// the QR overlay. It is drawn with canvas primitives so its height stays short,
// unlike widget.Button which has a fixed minimum height.
type qrButton struct {
	widget.BaseWidget
	text  string
	onTap func()
	dark  bool
}

func newQRButton(text string, onTap func(), dark bool) *qrButton {
	b := &qrButton{text: text, onTap: onTap, dark: dark}
	b.ExtendBaseWidget(b)
	return b
}

func (b *qrButton) MinSize() fyne.Size { return fyne.NewSize(80, qrH) }

func (b *qrButton) Tapped(_ *fyne.PointEvent) {
	if b.onTap != nil {
		b.onTap()
	}
}

func (b *qrButton) TappedSecondary(_ *fyne.PointEvent) {}

func (b *qrButton) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = 5
	label := canvas.NewText(b.text, color.White)
	label.Alignment = fyne.TextAlignCenter
	label.TextSize = 12
	r := &qrRenderer{b: b, bg: bg, label: label, obj: container.NewStack(bg, container.NewCenter(label))}
	r.applyColors()
	return r
}

type qrRenderer struct {
	b     *qrButton
	bg    *canvas.Rectangle
	label *canvas.Text
	obj   fyne.CanvasObject
}

func (r *qrRenderer) applyColors() {
	if r.b.dark {
		r.bg.FillColor = nordPolar3
		r.bg.StrokeColor = nordPolar4
		r.label.Color = nordSnow1
	} else {
		r.bg.FillColor = nordSnow1
		r.bg.StrokeColor = nordSnow2
		r.label.Color = nordPolar1
	}
}

func (r *qrRenderer) Layout(s fyne.Size) { r.obj.Resize(s) }
func (r *qrRenderer) MinSize() fyne.Size { return r.b.MinSize() }
func (r *qrRenderer) Refresh()           { r.applyColors(); canvas.Refresh(r.obj) }
func (r *qrRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.obj}
}
func (r *qrRenderer) Destroy() {}

func (u *appUI) rowLabel(i int) string {
	if u.copyVisible() && i == u.copyIndex() {
		return u.l(msgCopyLink)
	}
	return u.rows[i].label
}

func (u *appUI) rowIcon(i int) fyne.Resource {
	if u.copyVisible() && i == u.copyIndex() {
		return nil
	}
	return u.rows[i].b.icon
}

// chipNumber is the hotkey badge pinned to the right of a row; the copy row is
// always key 5, browsers are 1..4 by list position.
func (u *appUI) chipNumber(i int) string {
	if u.copyVisible() && i == u.copyIndex() {
		return "5"
	}
	return string(rune('1' + i))
}

// newChip builds a rounded hotkey badge whose colors are refreshed on theme
// toggle via updateChip.
func (u *appUI) newChip() fyne.CanvasObject {
	bg := canvas.NewRectangle(u.chipBg())
	bg.CornerRadius = 5
	bg.SetMinSize(fyne.NewSize(22, 22))
	border := canvas.NewRectangle(color.Transparent)
	border.CornerRadius = 5
	border.StrokeWidth = 1
	text := canvas.NewText("", u.chipText())
	text.TextSize = 11
	text.TextStyle.Monospace = true
	return container.NewStack(bg, border, container.NewCenter(text))
}

func (u *appUI) updateChip(o fyne.CanvasObject, num string) {
	s := o.(*fyne.Container)
	s.Objects[0].(*canvas.Rectangle).FillColor = u.chipBg()
	s.Objects[1].(*canvas.Rectangle).StrokeColor = u.chipBorder()
	t := s.Objects[2].(*fyne.Container).Objects[0].(*canvas.Text)
	t.Text = num
	t.Color = u.chipText()
	o.Refresh()
}

func (u *appUI) chipBg() color.Color {
	if u.dark {
		return nordPolar3
	}
	return nordSnow1
}

func (u *appUI) chipBorder() color.Color {
	if u.dark {
		return nordPolar4
	}
	return nordSnow2
}

func (u *appUI) chipText() color.Color {
	if u.dark {
		return nordSnow1
	}
	return nordPolar4
}

// refresh re-ranks rows and debounces the highlight so rapid keystrokes don't
// make the selection jump; the window size stays fixed.
func (u *appUI) refresh() {
	u.rows = rankRows(u.browsers, u.stats, u.entry.Text, u.hostTop)
	if u.selTimer != nil {
		u.selTimer.Stop()
	}
	u.selTimer = time.AfterFunc(selDebounce, func() {
		fyne.Do(func() {
			u.list.Refresh() // re-render rows so icons/labels match the new sort
			if len(u.rows) > 0 {
				u.list.Highlight(0)
			}
		})
	})
}

func (u *appUI) copyVisible() bool {
	return u.url != "" && u.entry.Text == ""
}

func (u *appUI) copyIndex() int { return len(u.rows) }

func (u *appUI) listLen() int {
	if u.copyVisible() {
		return len(u.rows) + 1
	}
	return len(u.rows)
}

func (u *appUI) activate(i int) {
	if i == u.copyIndex() && u.copyVisible() {
		u.copyAndQuit()
		return
	}
	if i < 0 || i >= len(u.rows) {
		return
	}
	u.launch(u.rows[i].b)
	u.a.Quit()
}

// focusEntry returns keyboard focus to the entry on the next event-loop tick.
// Calling Canvas().Focus immediately after SetContent is dropped because the
// new widget is not yet attached; deferring keeps Esc and the hotkeys alive.
func (u *appUI) focusEntry() {
	time.AfterFunc(0, func() {
		fyne.Do(func() { u.w.Canvas().Focus(u.entry) })
	})
}

// applyTheme resolves the effective variant from config: "light"/"dark" force
// it, "auto" (default) follows the system's ThemeVariant. u.dark mirrors it so
// the chip and QR colors follow the same source.
func (u *appUI) applyTheme() {
	v := theme.VariantDark
	switch loadSettings().Theme {
	case "light":
		v = theme.VariantLight
	case "dark":
		v = theme.VariantDark
	default: // auto follows the system
		v = u.a.Settings().ThemeVariant()
	}
	u.dark = v == theme.VariantDark
	u.a.Settings().SetTheme(&nordTheme{variant: v})
}

func (u *appUI) copyAndQuit() {
	if err := clipboard.WriteAll(u.url); err != nil {
		log.Printf("browserchooser: clipboard: %v", err)
		return
	}
	u.a.Quit()
}

func (u *appUI) launch(b *browser) {
	if err := b.open(u.url); err != nil {
		log.Printf("browserchooser: open %s: %v", b.name, err)
		return
	}
	recordOpen(u.url, b.id)
}

func (u *appUI) l(id string) string {
	if u.loc == nil {
		return id
	}
	s, err := u.loc.Localize(&i18n.LocalizeConfig{MessageID: id})
	if err != nil {
		return id
	}
	return s
}

// numEntry intercepts the digit and escape keys before they reach the text
// entry, so 1-5 select a row and typing still filters. Ctrl+C is caught via
// TypedShortcut because fyne KeyEvents carry no modifier state.
type numEntry struct {
	widget.Entry
	onSelect func(int)
	onEsc    func()
	onCtrlC  func()
}

// FocusLost pins keyboard focus back to the entry. Fyne only delivers key
// events to the focused object, so clicking the list or QR button would
// otherwise strand focus and stop Esc from quitting. In this modal picker the
// entry is the only focus owner that matters.
func (e *numEntry) FocusLost() {
	fyne.Do(func() {
		if c := fyne.CurrentApp().Driver().CanvasForObject(e); c != nil {
			c.Focus(e)
		}
	})
}

func (e *numEntry) TypedRune(r rune) {
	if r >= '1' && r <= '9' {
		if e.onSelect != nil {
			e.onSelect(int(r - '1'))
		}
		return
	}
	e.Entry.TypedRune(r)
}

func (e *numEntry) TypedKey(key *fyne.KeyEvent) {
	if key.Name == fyne.KeyEscape {
		if e.onEsc != nil {
			e.onEsc()
		}
		return
	}
	e.Entry.TypedKey(key)
}

func (e *numEntry) TypedShortcut(s fyne.Shortcut) {
	if _, ok := s.(*fyne.ShortcutCopy); ok {
		if e.onCtrlC != nil {
			e.onCtrlC()
		}
		return
	}
	e.Entry.TypedShortcut(s)
}

func usage() {
	fmt.Println("Usage: browserchooser [--set-default] [url]")
	fmt.Println("  --set-default  register as the default browser (Linux)")
	fmt.Println("  url            open the picker for this link, or copy it")
}

func newLocalizer() *i18n.Localizer {
	lang := language.English
	if l, err := locale.GetLanguage(); err == nil && l != "" {
		if tag, err := language.Parse(l); err == nil {
			lang = tag
		}
	}
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	if _, err := bundle.LoadMessageFileFS(messagesFS, "messages/en.json"); err != nil {
		log.Printf("browserchooser: messages: %v", err)
	}
	return i18n.NewLocalizer(bundle, lang.String())
}

func loadStats(browsers []browser) map[string]useStat {
	s := loadState()
	out := make(map[string]useStat, len(browsers))
	for _, b := range browsers {
		out[b.id] = s.Counts[b.id]
	}
	return out
}
