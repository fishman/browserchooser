package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// promptView is a focused confirmation that opens a site in its remembered
// browser. Enter confirms, Esc declines; the two buttons are the mouse path.
// It is the window content while prompting so keys reach it, not the picker.
type promptView struct {
	widget.BaseWidget
	u    *appUI
	host string
	b    *browser
}

func newPromptView(u *appUI, host string, b *browser) *promptView {
	p := &promptView{u: u, host: host, b: b}
	p.ExtendBaseWidget(p)
	return p
}

func (p *promptView) MinSize() fyne.Size { return p.BaseWidget.MinSize() }

func (p *promptView) CreateRenderer() fyne.WidgetRenderer {
	msg := widget.NewLabel(fmt.Sprintf(p.u.l(msgHostPrompt), p.host, p.b.name))
	msg.Alignment = fyne.TextAlignCenter
	hint := widget.NewLabel(p.u.l(msgHostHint))
	hint.Alignment = fyne.TextAlignCenter
	hint.TextStyle = fyne.TextStyle{Italic: true}
	open := widget.NewButton(p.u.l(msgHostOpen), p.u.promptOpen)
	choose := widget.NewButton(p.u.l(msgHostChoose), p.u.promptDecline)
	row := container.NewHBox(open, choose)
	content := container.NewVBox(msg, hint, row)
	return widget.NewSimpleRenderer(content)
}

// FocusLost pins keyboard focus back here so Enter/Esc keep working after a
// button click, mirroring numEntry in the picker.
func (p *promptView) FocusLost() {
	fyne.Do(func() {
		if c := fyne.CurrentApp().Driver().CanvasForObject(p); c != nil {
			c.Focus(p)
		}
	})
}

func (p *promptView) FocusGained() {}
func (p *promptView) TypedRune(r rune) {
}

func (p *promptView) TypedKey(key *fyne.KeyEvent) {
	switch key.Name {
	case fyne.KeyEnter, fyne.KeyReturn:
		p.u.promptOpen()
	case fyne.KeyEscape:
		p.u.promptDecline()
	}
}
