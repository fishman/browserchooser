package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"log"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/jeandeaual/go-locale"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/skip2/go-qrcode"
	"golang.org/x/text/language"
)

//go:embed messages
var messagesFS embed.FS

func main() {
	app.SetMetadata(fyne.AppMetadata{
		ID:         "dev.fishman.browserchooser",
		Migrations: map[string]bool{"fyneDo": true},
	})

	url := ""
	if len(os.Args) > 1 {
		url = os.Args[1]
	}

	a := app.NewWithID("dev.fishman.browserchooser")

	if url != "" {
		if b, ok := matchRule(url); ok {
			if err := b.open(url); err != nil {
				log.Printf("browserchooser: open %s: %v", b.name, err)
			} else {
				recordUse(a.Preferences(), b.id)
			}
			return
		}
	}

	w := a.NewWindow("Browser Chooser")
	u := newUI(a, w, url)
	w.SetContent(u.content())
	w.Resize(fyne.NewSize(520, 400))
	u.refresh()
	w.Canvas().Focus(u.entry)
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
}

func newUI(a fyne.App, w fyne.Window, url string) *appUI {
	u := &appUI{
		a: a, w: w, url: url, loc: newLocalizer(),
		browsers: detectBrowsers(),
	}
	if len(u.browsers) == 0 {
		u.browsers = []browser{fallbackBrowser()}
	}
	u.stats = loadStats(a.Preferences(), u.browsers)
	w.SetTitle(u.l("title"))

	u.entry = &numEntry{}
	u.entry.SetPlaceHolder(u.l("placeholder"))
	u.entry.OnChanged = func(string) { u.refresh() }
	u.entry.OnSubmitted = func(string) {
		if len(u.rows) > 0 {
			u.activate(0)
		}
	}
	u.entry.onSelect = u.activate
	u.entry.onEsc = func() { u.a.Quit() }
	u.entry.onCtrlC = func() {
		u.a.Clipboard().SetContent(u.url)
		u.a.Quit()
	}

	u.list = widget.NewList(
		func() int { return len(u.rows) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(u.rowLabel(int(id)))
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
		if qr := u.qr(); qr != nil {
			bottom = container.NewVBox(qr)
		}
	}
	return container.NewBorder(u.entry, bottom, nil, nil, u.list)
}

func (u *appUI) qr() fyne.CanvasObject {
	png, err := qrcode.Encode(u.url, qrcode.Medium, 128)
	if err != nil {
		return nil
	}
	img := canvas.NewImageFromReader(bytes.NewReader(png), "qr.png")
	img.FillMode = canvas.ImageFillOriginal
	return img
}

func (u *appUI) rowLabel(i int) string {
	if u.rows[i].copy {
		return u.rows[i].label
	}
	return numberLabel(i) + " " + u.rows[i].label
}

func numberLabel(i int) string {
	return string(rune('0'+i+1)) + "."
}

func (u *appUI) refresh() {
	u.rows = rankRows(u.browsers, u.stats, u.entry.Text, u.url, u.l("copyLink"))
	if len(u.rows) > 0 {
		u.list.Highlight(0)
	} else {
		u.list.Refresh()
	}
}

func (u *appUI) activate(i int) {
	if i < 0 || i >= len(u.rows) {
		return
	}
	r := u.rows[i]
	if r.copy {
		u.a.Clipboard().SetContent(u.url)
		u.a.Quit()
		return
	}
	u.launch(r.b)
	u.a.Quit()
}

func (u *appUI) launch(b *browser) {
	if err := b.open(u.url); err != nil {
		log.Printf("browserchooser: open %s: %v", b.name, err)
		return
	}
	recordUse(u.a.Preferences(), b.id)
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

func countKey(id string) string { return "count." + id }
func lastKey(id string) string  { return "last." + id }

func loadStats(p fyne.Preferences, browsers []browser) map[string]useStat {
	m := make(map[string]useStat, len(browsers))
	for _, b := range browsers {
		m[b.id] = useStat{
			Count: p.IntWithFallback(countKey(b.id), 0),
			Last:  int64(p.IntWithFallback(lastKey(b.id), 0)),
		}
	}
	return m
}

func recordUse(p fyne.Preferences, id string) {
	p.SetInt(countKey(id), p.IntWithFallback(countKey(id), 0)+1)
	p.SetInt(lastKey(id), int(time.Now().Unix()))
}
