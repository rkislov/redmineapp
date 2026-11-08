// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/rkislov/redmineapp/internal/redmine"
	"github.com/rkislov/redmineapp/internal/settings"
	"github.com/zalando/go-keyring"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const service = "org.redmine-go.desktop"

type UI struct {
	navButtons                                                     map[string]*widget.Button
	app                                                            fyne.App
	win                                                            fyne.Window
	cfg                                                            settings.Settings
	client                                                         *redmine.Client
	me                                                             redmine.User
	issues                                                         []redmine.Issue
	projects, statuses, priorities, trackers, activities           []redmine.Ref
	mode, status, search                                           string
	project, offset, total, selected, generation, detailGeneration int
	list                                                           *widget.List
	detail                                                         *fyne.Container
	heading, footer, account                                       *widget.Label
	prev, next                                                     *widget.Button
	appearanceSelect                                               *appearanceToggle
	projectsLoading                                                bool
	projectsError                                                  error
	activeProjects                                                 *projectPicker
}

func New(a fyne.App) *UI {
	s, e := settings.Load()
	a.Settings().SetTheme(appTheme{Mode: appearanceMode(s.Appearance), Dark: s.Appearance == "dark"})
	u := &UI{app: a, win: a.NewWindow("redmineapp"), cfg: s, mode: "Мои", status: "open"}
	u.win.Resize(fyne.NewSize(1240, 820))
	u.login()
	if e != nil {
		dialog.ShowError(fmt.Errorf("чтение настроек: %w", e), u.win)
	}
	return u
}
func (u *UI) Run() { u.win.ShowAndRun() }
func (u *UI) login() {
	name := widget.NewEntry()
	name.SetPlaceHolder("Рабочий Redmine")
	server := widget.NewEntry()
	server.SetPlaceHolder("https://redmine.example.org")
	key := widget.NewPasswordEntry()
	key.SetPlaceHolder("Моя учётная запись → API-ключ")
	username := widget.NewEntry()
	username.SetPlaceHolder("Логин Redmine")
	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Пароль Redmine")
	keyForm := widget.NewForm(widget.NewFormItem("API-ключ", key))
	basicForm := widget.NewForm(widget.NewFormItem("Логин", username), widget.NewFormItem("Пароль", password))
	basicForm.Hide()
	auth := widget.NewSelect([]string{"API-ключ", "Логин и пароль"}, func(mode string) {
		if mode == "Логин и пароль" {
			keyForm.Hide()
			basicForm.Show()
		} else {
			basicForm.Hide()
			keyForm.Show()
		}
	})
	auth.SetSelected("API-ключ")
	remember := widget.NewCheck("Сохранить в системном хранилище", nil)
	remember.SetChecked(true)
	names := []string{}
	for _, p := range u.cfg.Profiles {
		suffix := p.URL
		if p.Username != "" {
			suffix = p.Username + " · " + suffix
		}
		names = append(names, p.Name+" · "+suffix)
	}
	profiles := widget.NewSelect(names, nil)
	profiles.OnChanged = func(n string) {
		for i, label := range names {
			if n != label {
				continue
			}
			p := u.cfg.Profiles[i]
			name.SetText(p.Name)
			server.SetText(p.URL)
			username.SetText(p.Username)
			key.SetText("")
			password.SetText("")
			if p.Auth == "basic" {
				auth.SetSelected("Логин и пароль")
			} else {
				auth.SetSelected("API-ключ")
			}
			go func() {
				secret, e := keyring.Get(service, p.CredentialID())
				if e != nil {
					return
				}
				fyne.Do(func() {
					if server.Text != p.URL {
						return
					}
					if p.Auth == "basic" {
						if auth.Selected == "Логин и пароль" && username.Text == p.Username && password.Text == "" {
							password.SetText(secret)
						}
					} else if auth.Selected == "API-ключ" && key.Text == "" {
						key.SetText(secret)
					}
				})
			}()
			break
		}
	}
	profiles.PlaceHolder = "Сохранённые подключения"
	if len(names) > 0 {
		idx := u.cfg.Active
		if idx < 0 || idx >= len(names) {
			idx = 0
		}
		profiles.SetSelected(names[idx])
	}
	message := label("REST API должен быть включён в Redmine. Для защиты данных используйте HTTPS.")
	var connect *widget.Button
	connect = widget.NewButton("Подключиться", func() {
		var c *redmine.Client
		var e error
		basic := auth.Selected == "Логин и пароль"
		if basic {
			c, e = redmine.NewBasic(server.Text, username.Text, password.Text)
		} else {
			c, e = redmine.New(server.Text, key.Text)
		}
		if e != nil {
			dialog.ShowError(e, u.win)
			return
		}
		p := settings.Profile{Name: strings.TrimSpace(name.Text), URL: c.Base.String(), Auth: "api_key"}
		secret := c.Key
		if basic {
			p.Auth = "basic"
			p.Username = c.Username
			secret = c.Password
		}
		if p.Name == "" {
			p.Name = c.Base.Host
		}
		saveSecret := remember.Checked
		connect.Disable()
		message.SetText("Проверяем подключение…")
		go func() {
			me, e := c.Me(context.Background())
			var warning error
			if e == nil && saveSecret {
				warning = keyring.Set(service, p.CredentialID(), secret)
			}
			fyne.Do(func() {
				connect.Enable()
				if e != nil {
					message.SetText(e.Error())
					return
				}
				u.client = c
				u.me = me
				found := false
				for i, old := range u.cfg.Profiles {
					if old.CredentialID() == p.CredentialID() {
						u.cfg.Profiles[i] = p
						u.cfg.Active = i
						found = true
						break
					}
				}
				if !found {
					u.cfg.Profiles = append(u.cfg.Profiles, p)
					u.cfg.Active = len(u.cfg.Profiles) - 1
				}
				saveErr := u.cfg.Save()
				u.shell()
				u.loadMetadata()
				u.refresh()
				if warning != nil {
					dialog.ShowInformation("Вход без сохранения секрета", "Системное хранилище недоступно. Данные входа останутся только в памяти: "+warning.Error(), u.win)
				}
				if saveErr != nil {
					dialog.ShowError(saveErr, u.win)
				}
			})
		}()
	})
	connect.Importance = widget.HighImportance
	form := container.NewVBox(widget.NewLabelWithStyle("redmineapp", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}), label("Задачи, проекты и команда — в одном окне"), profiles, widget.NewForm(widget.NewFormItem("Название", name), widget.NewFormItem("Сервер", server), widget.NewFormItem("Способ входа", auth)), keyForm, basicForm, remember, message, connect, widget.NewSeparator(), u.appearanceControl())
	box := container.NewGridWrap(fyne.NewSize(590, 650), container.NewVScroll(newPanel(form)))
	u.win.SetContent(container.NewCenter(box))
}

func (u *UI) appearanceControl() fyne.CanvasObject {
	toggle := newAppearanceToggle(u.cfg.Appearance, func(mode string) {
		u.cfg.Appearance = mode
		u.app.Settings().SetTheme(appTheme{Mode: mode, Dark: mode == "dark"})
		if e := u.cfg.Save(); e != nil {
			dialog.ShowError(e, u.win)
		}
	})
	u.appearanceSelect = toggle
	return container.NewVBox(widget.NewLabel("Оформление"), toggle)
}

func (u *UI) shell() {
	u.account = widget.NewLabel(u.me.Firstname + " " + u.me.Lastname)
	nav := container.NewVBox(widget.NewLabelWithStyle("redmineapp", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), widget.NewSeparator(), u.account)
	navButtons := map[string]*widget.Button{}
	u.navButtons = navButtons
	for _, n := range []struct {
		name string
		icon fyne.Resource
	}{{"Мои", theme.AccountIcon()}, {"Слежу", theme.VisibilityIcon()}, {"Избранное", starIcon}, {"Проекты", theme.FolderIcon()}, {"Поиск", theme.SearchIcon()}} {
		n := n
		button := widget.NewButtonWithIcon(n.name, n.icon, func() {
			if n.name == "Проекты" {
				u.chooseProject()
				return
			}
			for name, button := range navButtons {
				button.Importance = widget.LowImportance
				if name == n.name {
					button.Importance = widget.HighImportance
				}
				button.Refresh()
			}
			u.mode = n.name
			u.project = 0
			u.offset = 0
			u.refresh()
		})
		button.Importance = widget.LowImportance
		if n.name == u.mode {
			button.Importance = widget.HighImportance
		}
		navButtons[n.name] = button
		nav.Add(button)
	}
	navBottom := container.NewVBox(u.appearanceControl(), widget.NewButtonWithIcon("Новая задача", theme.ContentAddIcon(), u.createIssue), widget.NewButtonWithIcon("Подключения", theme.SettingsIcon(), func() { u.generation++; u.detailGeneration++; u.client = nil; u.login() }))
	sidebar := newPanel(container.NewBorder(nav, navBottom, nil, nil, layout.NewSpacer()))
	u.heading = widget.NewLabelWithStyle(u.mode, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	u.heading.Truncation = fyne.TextTruncateEllipsis
	filterButtons := map[string]*widget.Button{}
	filter := container.NewGridWithColumns(3)
	for _, f := range []struct{ name, value string }{{"Открытые", "open"}, {"Закрытые", "closed"}, {"Все", "*"}} {
		f := f
		b := widget.NewButton(f.name, func() {
			u.status = f.value
			u.offset = 0
			for value, button := range filterButtons {
				button.Importance = widget.LowImportance
				if value == u.status {
					button.Importance = widget.HighImportance
				}
				button.Refresh()
			}
			u.refresh()
		})
		b.Importance = widget.LowImportance
		if f.value == u.status {
			b.Importance = widget.HighImportance
		}
		filterButtons[f.value] = b
		filter.Add(b)
	}
	search := widget.NewEntry()
	search.SetPlaceHolder("Тема задачи или #номер")
	search.OnSubmitted = func(s string) { u.search = strings.TrimSpace(s); u.offset = 0; u.refresh() }
	u.list = widget.NewList(func() int { return len(u.issues) }, func() fyne.CanvasObject {
		statusRow := container.NewBorder(nil, nil, newBadge(), nil, widget.NewLabel("#123"))
		title := widget.NewLabelWithStyle("Задача", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		title.Truncation = fyne.TextTruncateEllipsis
		meta := widget.NewLabel("Проект · Исполнитель")
		meta.Truncation = fyne.TextTruncateEllipsis
		p := newPanel(container.NewVBox(statusRow, title, meta, newProgress()))
		p.interactive = true
		return p
	}, func(i widget.ListItemID, o fyne.CanvasObject) {
		v := u.issues[i]
		p := o.(*floatingPanel)
		p.selected = v.ID == u.selected
		b := p.content.(*fyne.Container)
		statusRow := b.Objects[0].(*fyne.Container)
		statusRow.Objects[1].(*badge).Set(v.Status.Name)
		statusRow.Objects[0].(*widget.Label).SetText(v.Tracker.Name + " #" + strconv.Itoa(v.ID))
		b.Objects[1].(*widget.Label).SetText(v.Subject)
		b.Objects[2].(*widget.Label).SetText(v.Project.Name + " · " + v.AssignedTo.Name)
		b.Objects[3].(*miniProgress).SetValue(float64(v.Done) / 100)
		p.Refresh()
	})
	u.list.HideSeparators = true
	u.list.OnSelected = func(i widget.ListItemID) { u.openIssue(u.issues[i].ID) }
	u.footer = widget.NewLabel("")
	u.prev = widget.NewButton("←", func() {
		if u.offset >= 50 {
			u.offset -= 50
			u.refresh()
		}
	})
	u.next = widget.NewButton("→", func() { u.offset += 50; u.refresh() })
	mid := container.NewBorder(container.NewVBox(container.NewBorder(nil, nil, u.heading, widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), u.refresh)), filter, search), container.NewBorder(nil, nil, u.prev, u.next, u.footer), nil, nil, u.list)
	u.detail = container.NewStack(container.NewCenter(widget.NewLabel("Выберите задачу")))
	right := container.NewHSplit(container.NewPadded(mid), container.NewPadded(u.detail))
	right.Offset = .41
	root := container.NewHSplit(sidebar, right)
	root.Offset = .18
	u.win.SetContent(container.NewPadded(root))
}
func (u *UI) loadMetadata() {
	u.projectsLoading = true
	u.projectsError = nil
	u.projects = nil
	c := u.client
	go func() {
		type result struct {
			v []redmine.Ref
			e error
		}
		paths := [][2]string{{"projects.json", "projects"}, {"issue_statuses.json", "issue_statuses"}, {"enumerations/issue_priorities.json", "issue_priorities"}, {"trackers.json", "trackers"}, {"enumerations/time_entry_activities.json", "time_entry_activities"}}
		results := make([]result, len(paths))
		for i, p := range paths {
			results[i].v, results[i].e = c.Refs(context.Background(), p[0], p[1])
		}
		fyne.Do(func() {
			if u.client != c {
				return
			}
			u.projects = results[0].v
			u.projectsLoading = false
			u.projectsError = results[0].e
			if u.activeProjects != nil {
				p := u.activeProjects
				p.all = append([]redmine.Ref(nil), u.projects...)
				p.loading = false
				p.loadError = u.projectsError
				p.refilter()
			}
			u.statuses = results[1].v
			u.priorities = results[2].v
			u.trackers = results[3].v
			u.activities = results[4].v
			for _, r := range results {
				if r.e != nil {
					dialog.ShowInformation("Справочники", "Часть справочников недоступна: "+r.e.Error(), u.win)
					break
				}
			}
		})
	}()
}
func (u *UI) refresh() {
	if u.client == nil || u.list == nil {
		return
	}
	u.generation++
	g := u.generation
	c := u.client
	q := url.Values{"status_id": {u.status}, "sort": {"updated_on:desc"}, "limit": {"50"}, "offset": {strconv.Itoa(u.offset)}}
	if u.mode == "Мои" {
		q.Set("assigned_to_id", "me")
	}
	if u.mode == "Слежу" {
		q.Set("watcher_id", "me")
	}
	if u.project > 0 {
		q.Set("project_id", strconv.Itoa(u.project))
	}
	if u.search != "" {
		if id, e := strconv.Atoi(strings.TrimPrefix(u.search, "#")); e == nil && id > 0 {
			q.Set("issue_id", strconv.Itoa(id))
		} else {
			q.Set("subject", "~"+u.search)
		}
	}
	if u.mode == "Избранное" {
		ids := []string{}
		for k, v := range u.cfg.Favorites {
			prefix := c.Base.String() + "#"
			if v && strings.HasPrefix(k, prefix) {
				ids = append(ids, strings.TrimPrefix(k, prefix))
			}
		}
		if len(ids) == 0 {
			u.issues = nil
			u.total = 0
			u.list.Refresh()
			u.heading.SetText(u.mode)
			u.footer.SetText("Нет избранных задач")
			u.prev.Disable()
			u.next.Disable()
			return
		}
		q.Set("issue_id", strings.Join(ids, ","))
	}
	u.heading.SetText(u.mode)
	u.footer.SetText("Загрузка…")
	u.prev.Disable()
	u.next.Disable()
	go func() {
		p, e := c.Issues(context.Background(), q)
		fyne.Do(func() {
			if g != u.generation || u.client != c {
				return
			}
			if e != nil {
				u.footer.SetText("Ошибка загрузки")
				dialog.ShowError(e, u.win)
				return
			}
			u.issues = p.Issues
			u.total = p.Total
			u.list.UnselectAll()
			u.list.Refresh()
			u.footer.SetText(fmt.Sprintf("%d–%d из %d", min(u.offset+1, p.Total), u.offset+len(p.Issues), p.Total))
			if u.offset > 0 {
				u.prev.Enable()
			}
			if u.offset+50 < p.Total {
				u.next.Enable()
			}
		})
	}()
}
func label(s string) *widget.Label { w := widget.NewLabel(s); w.Wrapping = fyne.TextWrapWord; return w }
func (u *UI) openIssue(id int) {
	u.selected = id
	if u.list != nil {
		u.list.Refresh()
	}
	u.detailGeneration++
	g := u.detailGeneration
	c := u.client
	u.detail.Objects = []fyne.CanvasObject{container.NewCenter(widget.NewLabel("Загрузка задачи…"))}
	u.detail.Refresh()
	go func() {
		v, e := c.Issue(context.Background(), id)
		fyne.Do(func() {
			if u.client != c || g != u.detailGeneration {
				return
			}
			if e != nil {
				u.detail.Objects = []fyne.CanvasObject{label(e.Error())}
				u.detail.Refresh()
				return
			}
			u.showIssue(v)
		})
	}()
}
func (u *UI) showIssue(v redmine.Issue) {
	title := widget.NewRichText(&widget.TextSegment{Text: v.Subject, Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}, SizeName: theme.SizeNameHeadingText}})
	title.Wrapping = fyne.TextWrapWord
	favorite := widget.NewButtonWithIcon("", starIcon, nil)
	key := u.cfg.FavoriteKey(u.client.Base.String(), v.ID)
	setStar := func() {
		if u.cfg.Favorites[key] {
			favorite.SetText("В избранном")
		} else {
			favorite.SetText("В избранное")
		}
	}
	setStar()
	favorite.OnTapped = func() {
		u.cfg.Favorites[key] = !u.cfg.Favorites[key]
		if e := u.cfg.Save(); e != nil {
			dialog.ShowError(e, u.win)
		}
		setStar()
		if u.mode == "Избранное" {
			u.refresh()
		}
	}
	head := container.NewVBox(container.NewBorder(nil, nil, label(fmt.Sprintf("#%d · %s", v.ID, v.Project.Name)), favorite), container.NewHBox(func() *badge { b := newBadge(); b.Set(v.Status.Name); return b }(), label(v.Tracker.Name)), title, label("Создал(а) "+v.Author.Name))
	progress := newProgress()
	progress.SetValue(float64(v.Done) / 100)
	attrs := widget.NewForm(widget.NewFormItem("Статус", label(v.Status.Name)), widget.NewFormItem("Приоритет", label(v.Priority.Name)), widget.NewFormItem("Исполнитель", container.NewBorder(nil, nil, nil, widget.NewButton("Взять", func() { u.update(v.ID, map[string]any{"assigned_to_id": u.me.ID}) }), label(v.AssignedTo.Name))), widget.NewFormItem("Готовность", progress), widget.NewFormItem("Начало", label(v.Start)), widget.NewFormItem("Срок", label(v.Due)), widget.NewFormItem("Затрачено", label(fmt.Sprintf("%.2f ч", v.Spent))), widget.NewFormItem("Обновлено", label(formatDate(v.Updated))))
	body := container.NewVBox(head, newPanel(attrs), widget.NewLabelWithStyle("Описание", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), newPanel(label(v.Description)), widget.NewLabelWithStyle("Комментарии", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	for i, j := range v.Journals {
		if strings.TrimSpace(j.Notes) == "" {
			continue
		}
		body.Add(newPanel(container.NewVBox(widget.NewLabelWithStyle(j.User.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), label(formatDate(j.Created)+fmt.Sprintf(" · #%d", i+1)), label(j.Notes))))
	}
	comment := widget.NewButtonWithIcon("Комментарий", theme.MailComposeIcon(), func() { u.comment(v.ID) })
	comment.Importance = widget.HighImportance
	actions := container.NewHBox(comment, widget.NewButton("Изменить", func() { u.editIssue(v) }), widget.NewButtonWithIcon("Время", theme.HistoryIcon(), func() { u.logTime(v.ID) }), widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() { u.openIssue(v.ID) }))
	u.detail.Objects = []fyne.CanvasObject{container.NewBorder(nil, newPanel(actions), nil, nil, container.NewVScroll(body))}
	u.detail.Refresh()
}
func formatDate(s string) string {
	if t, e := time.Parse(time.RFC3339, s); e == nil {
		return t.Local().Format("02.01.2006 15:04")
	}
	return s
}
func (u *UI) mutation(work func() error, done func()) {
	c := u.client
	go func() {
		e := work()
		fyne.Do(func() {
			if u.client != c {
				return
			}
			if e != nil {
				dialog.ShowError(e, u.win)
				return
			}
			done()
		})
	}()
}
func (u *UI) update(id int, fields map[string]any) {
	c := u.client
	u.mutation(func() error { return c.Update(context.Background(), id, fields) }, func() {
		if u.selected == id {
			u.openIssue(id)
		}
		u.refresh()
	})
}
func (u *UI) comment(id int) {
	entry := widget.NewMultiLineEntry()
	entry.SetMinRowsVisible(6)
	d := dialog.NewForm("Комментарий", "Отправить", "Отмена", []*widget.FormItem{widget.NewFormItem("Текст", entry)}, func(ok bool) {
		if ok && strings.TrimSpace(entry.Text) != "" {
			u.update(id, map[string]any{"notes": entry.Text})
		}
	}, u.win)
	d.Resize(fyne.NewSize(560, 340))
	d.Show()
}
func selectRef(refs []redmine.Ref, id int) (*widget.Select, func() int) {
	opts := []string{}
	values := map[string]int{}
	chosen := ""
	for _, r := range refs {
		n := fmt.Sprintf("%s (#%d)", r.Name, r.ID)
		opts = append(opts, n)
		values[n] = r.ID
		if r.ID == id {
			chosen = n
		}
	}
	s := widget.NewSelect(opts, nil)
	s.SetSelected(chosen)
	return s, func() int { return values[s.Selected] }
}
func (u *UI) editIssue(v redmine.Issue) {
	subject := widget.NewEntry()
	subject.SetText(v.Subject)
	desc := widget.NewMultiLineEntry()
	desc.SetMinRowsVisible(4)
	desc.SetText(v.Description)
	status, statusID := selectRef(u.statuses, v.Status.ID)
	priority, priorityID := selectRef(u.priorities, v.Priority.ID)
	ratio := widget.NewSelect([]string{"0", "10", "20", "30", "40", "50", "60", "70", "80", "90", "100"}, nil)
	ratio.SetSelected(strconv.Itoa(v.Done))
	due := widget.NewEntry()
	due.SetText(v.Due)
	d := dialog.NewForm("Изменить задачу", "Сохранить", "Отмена", []*widget.FormItem{widget.NewFormItem("Тема", subject), widget.NewFormItem("Описание", desc), widget.NewFormItem("Статус", status), widget.NewFormItem("Приоритет", priority), widget.NewFormItem("Готовность, %", ratio), widget.NewFormItem("Срок YYYY-MM-DD", due)}, func(ok bool) {
		if !ok {
			return
		}
		if strings.TrimSpace(subject.Text) == "" {
			dialog.ShowError(fmt.Errorf("тема не может быть пустой"), u.win)
			return
		}
		if due.Text != "" {
			if _, e := time.Parse("2006-01-02", due.Text); e != nil {
				dialog.ShowError(fmt.Errorf("срок: используйте YYYY-MM-DD"), u.win)
				return
			}
		}
		n, _ := strconv.Atoi(ratio.Selected)
		f := map[string]any{"subject": subject.Text, "description": desc.Text, "done_ratio": n, "due_date": due.Text}
		if statusID() > 0 {
			f["status_id"] = statusID()
		}
		if priorityID() > 0 {
			f["priority_id"] = priorityID()
		}
		u.update(v.ID, f)
	}, u.win)
	d.Resize(fyne.NewSize(620, 530))
	d.Show()
}
func (u *UI) createIssue() {
	if u.client == nil {
		return
	}
	projectID := u.project
	projectName := widget.NewLabel("Проект не выбран")
	projectName.Truncation = fyne.TextTruncateEllipsis
	for _, v := range u.projects {
		if v.ID == projectID {
			projectName.SetText(v.Name)
			break
		}
	}
	choose := widget.NewButton("Выбрать…", func() { u.showProjectPicker(func(v redmine.Ref) { projectID = v.ID; projectName.SetText(v.Name) }) })
	p := container.NewBorder(nil, nil, nil, choose, projectName)
	t, tid := selectRef(u.trackers, 0)
	subject := widget.NewEntry()
	desc := widget.NewMultiLineEntry()
	desc.SetMinRowsVisible(5)
	d := dialog.NewForm("Новая задача", "Создать", "Отмена", []*widget.FormItem{widget.NewFormItem("Проект", p), widget.NewFormItem("Трекер", t), widget.NewFormItem("Тема", subject), widget.NewFormItem("Описание", desc)}, func(ok bool) {
		if !ok {
			return
		}
		if projectID == 0 || strings.TrimSpace(subject.Text) == "" {
			dialog.ShowError(fmt.Errorf("выберите проект и введите тему"), u.win)
			return
		}
		f := map[string]any{"project_id": projectID, "subject": subject.Text, "description": desc.Text}
		if tid() > 0 {
			f["tracker_id"] = tid()
		}
		c := u.client
		var created redmine.Issue
		u.mutation(func() error { var e error; created, e = c.Create(context.Background(), f); return e }, func() { u.refresh(); u.openIssue(created.ID) })
	}, u.win)
	d.Resize(fyne.NewSize(600, 450))
	d.Show()
}
func (u *UI) logTime(id int) {
	hours := widget.NewEntry()
	hours.SetPlaceHolder("1.5")
	date := widget.NewEntry()
	date.SetText(time.Now().Format("2006-01-02"))
	activity, aid := selectRef(u.activities, 0)
	notes := widget.NewEntry()
	d := dialog.NewForm("Учёт времени", "Записать", "Отмена", []*widget.FormItem{widget.NewFormItem("Часы", hours), widget.NewFormItem("Дата", date), widget.NewFormItem("Деятельность", activity), widget.NewFormItem("Комментарий", notes)}, func(ok bool) {
		if !ok {
			return
		}
		h, e := strconv.ParseFloat(strings.ReplaceAll(hours.Text, ",", "."), 64)
		if e != nil || h <= 0 || h > 24 {
			dialog.ShowError(fmt.Errorf("укажите часы: больше 0 и не больше 24"), u.win)
			return
		}
		if _, e = time.Parse("2006-01-02", date.Text); e != nil {
			dialog.ShowError(fmt.Errorf("дата: используйте YYYY-MM-DD"), u.win)
			return
		}
		f := map[string]any{"issue_id": id, "hours": h, "spent_on": date.Text, "comments": notes.Text}
		if aid() > 0 {
			f["activity_id"] = aid()
		}
		c := u.client
		u.mutation(func() error { return c.LogTime(context.Background(), f) }, func() {
			if u.selected == id {
				u.openIssue(id)
			}
		})
	}, u.win)
	d.Resize(fyne.NewSize(520, 340))
	d.Show()
}

// Demo opens an isolated in-memory connection without persisting credentials.
func (u *UI) Demo() {
	u.client = redmine.NewDemo()
	u.me = redmine.User{ID: 1, Firstname: "Сергей", Lastname: "Иванов"}
	u.mode = "Слежу"
	u.win.SetTitle("redmineapp — демонстрация")
	u.shell()
	u.loadMetadata()
	u.refresh()
	u.openIssue(246614)
}
