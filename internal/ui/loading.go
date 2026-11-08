// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// loadingPanel keeps the animation and message centered in the available pane.
type loadingPanel struct {
	view     *fyne.Container
	activity *widget.Activity
	caption  *widget.Label
	running  bool
}

func newLoadingPanel(text string) *loadingPanel {
	p := &loadingPanel{activity: widget.NewActivity(), caption: label(text)}
	p.caption.Alignment = fyne.TextAlignCenter
	p.view = container.New(centeredStatusLayout{}, container.NewVBox(container.NewCenter(p.activity), p.caption))
	p.view.Hide()
	return p
}
func (p *loadingPanel) Start(text string) {
	p.caption.SetText(text)
	p.activity.Show()
	p.view.Show()
	p.activity.Start()
	p.running = true
}
func (p *loadingPanel) Stop() { p.activity.Stop(); p.running = false; p.view.Hide() }
func (p *loadingPanel) Message(text string) {
	p.activity.Stop()
	p.activity.Hide()
	p.running = false
	p.caption.SetText(text)
	p.view.Show()
}
func (u *UI) stopPaneRequests() {
	u.generation++
	u.detailGeneration++
	if u.listCancel != nil {
		u.listCancel()
	}
	if u.detailCancel != nil {
		u.detailCancel()
	}
	if u.metadataCancel != nil {
		u.metadataCancel()
	}
	if u.listLoading != nil {
		u.listLoading.Stop()
	}
	if u.detailLoading != nil {
		u.detailLoading.Stop()
	}
}
func (u *UI) stopWork() { u.cancelAuthentication(); u.stopPaneRequests() }
