// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package main

import (
	_ "embed"
	"flag"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/rkislov/redmineapp/internal/ui"
)

//go:embed assets/icon.png
var icon []byte

func main() {
	demo := flag.Bool("demo", false, "Демонстрация без сервера и сетевых запросов")
	flag.Parse()
	a := app.NewWithID("org.redmine-go.desktop")
	a.SetIcon(fyne.NewStaticResource("redmine.png", icon))
	var u *ui.UI
	if *demo {
		u = ui.NewDemo(a)
	} else {
		u = ui.New(a)
	}
	u.Run()
}
