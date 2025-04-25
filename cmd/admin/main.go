//go:build windows

package main

import (
	"log"
	"path/filepath"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

var (
	teamNameEdit *walk.LineEdit
	sportCombo   *walk.ComboBox
	logoPreview  *walk.ImageView
	logoPath     string
)

func main() {
	var mw *walk.MainWindow
	var tabs *walk.TabWidget

	err := MainWindow{
		AssignTo: &mw,
		Title:    "Scoreboard Admin",
		MinSize:  Size{Width: 600, Height: 500},
		Layout:   VBox{MarginsZero: true},
		Children: []Widget{
			TabWidget{
				AssignTo:      &tabs,
				StretchFactor: 1,
				Pages: []TabPage{
					{
						Title:  "Templates",
						Layout: VBox{},
						Children: []Widget{
							Label{Text: "Hier kommen die Template-Einstellungen hin."},
						},
					},
					{
						Title:  "Teams",
						Layout: VBox{},
						Children: []Widget{
							GroupBox{
								Title:  "Teamdaten",
								Layout: Grid{Columns: 2},
								Children: []Widget{
									Label{Text: "Teamname:"},
									LineEdit{AssignTo: &teamNameEdit},

									Label{Text: "Sportart:"},
									ComboBox{
										AssignTo: &sportCombo,
										Model:    []string{"Football", "Basketball", "Handball"},
									},

									Label{Text: "Logo:"},
									PushButton{
										Text: "Logo wählen...",
										OnClicked: func() {
											chooseLogo()
										},
									},

									Label{Text: "Vorschau:"},
									ImageView{
										AssignTo:      &logoPreview,
										MinSize:       Size{Width: 150, Height: 100},
										StretchFactor: 1,
									},

									HSpacer{},
									PushButton{
										Text: "Team speichern",
										OnClicked: func() {
											saveTeam()
										},
									},
								},
							},

							VSpacer{},
						},
					},
					{
						Title:  "Videos",
						Layout: VBox{},
						Children: []Widget{
							Label{Text: "Videoverwaltung folgt hier."},
						},
					},
					{
						Title:  "Spiele",
						Layout: VBox{},
						Children: []Widget{
							Label{Text: "Spiele anlegen und verwalten."},
						},
					},
					{
						Title:  "Livespiel",
						Layout: VBox{},
						Children: []Widget{
							Label{Text: "Hier wird das aktuelle Spiel gesteuert."},
						},
					},
				},
			},
		},
	}.Create()

	if err != nil {
		log.Fatal(err)
	}

	mw.Run()
}

func chooseLogo() {
	dlg := new(walk.FileDialog)
	dlg.Filter = "Bilddateien (*.png;*.jpg)|*.png;*.jpg"

	if ok, _ := dlg.ShowOpen(nil); ok {
		logoPath = dlg.FilePath
		img, err := walk.NewImageFromFile(logoPath)
		if err == nil {
			logoPreview.SetImage(img)
		}
	}
}

func saveTeam() {
	teamName := teamNameEdit.Text()
	sport := sportCombo.Text()
	log.Printf("Speichern: %s (%s) Logo: %s", teamName, sport, filepath.Base(logoPath))

	walk.MsgBox(nil, "Team gespeichert", "Das Team wurde gespeichert (noch ohne DB).", walk.MsgBoxIconInformation)
}

func runDisplayWindow() error {
	var mw *walk.MainWindow

	err := MainWindow{
		AssignTo: &mw,
		Title:    "Anzeige",
		Bounds:   Rectangle{X: 0, Y: 0, Width: 288, Height: 96},
		Layout:   VBox{},
		Children: []Widget{
			Label{
				Text: "Anzeige aktiv",
			},
		},
	}.Create()
	if err != nil {
		return err
	}

	// Frameless:
	win.SetWindowLong(mw.Handle(), win.GWL_STYLE, 0)

	// Always on top:
	win.SetWindowPos(
		mw.Handle(),
		win.HWND_TOPMOST,
		0, 0, 0, 0,
		win.SWP_NOMOVE|win.SWP_NOSIZE,
	)

	mw.Show()

	return nil
}
