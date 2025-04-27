//go:build windows

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unsafe"

	"github.com/KernTom/scoreboard-manager/internal/database"
	"github.com/KernTom/scoreboard-manager/internal/models"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	"golang.org/x/image/draw"
)

const defaultFontFamily = "DS-Digital"
const defaultClockSize = 32
const defaultScoreSize = 36
const defaultPeriodSize = 24

var (
	iconNew    *walk.Bitmap
	iconEdit   *walk.Bitmap
	iconSave   *walk.Bitmap
	iconCancel *walk.Bitmap
	iconDelete *walk.Bitmap
)
var (
	binder       *walk.DataBinder
	teamNameEdit *walk.LineEdit
	sportCombo   *walk.ComboBox
	logoPreview  *walk.ImageView
	logoPath     string

	heimCombo *walk.ComboBox
	gastCombo *walk.ComboBox

	passwordEdit *walk.LineEdit
	widthEdit    *walk.NumberEdit
	heightEdit   *walk.NumberEdit
	xEdit        *walk.NumberEdit
	yEdit        *walk.NumberEdit

	sportSelect        *walk.ComboBox
	periodLabelEdit    *walk.LineEdit
	periodsCountEdit   *walk.NumberEdit
	periodDurationEdit *walk.NumberEdit

	gameclockModeCombo *walk.ComboBox
	showPeriodCB       *walk.CheckBox
	showGameclockCB    *walk.CheckBox
	showClockCB        *walk.CheckBox

	sports         = []string{"American Football", "Fußball"}
	gameclockModes = []string{"Aufwärts (MM:SS)", "Aufwärts (Fußball-Minuten)", "Abwärts (MM:SS)"}
)

var previewWindow *walk.MainWindow
var previewContent *walk.Composite
var previewOpen bool
var matchSportFilterCombo *walk.ComboBox
var clockColor walk.Color
var scoreColor walk.Color
var periodColor walk.Color
var backgroundColor walk.Color
var clockFontColor string
var scoreFontColor walk.Color
var periodFontColor walk.Color
var backgroundFontColor walk.Color
var clockColorPreview *walk.Composite
var scoreColorPreview *walk.Composite
var periodColorPreview *walk.Composite
var backgroundColorPreview *walk.Composite
var clockFontColorPreview *walk.Composite
var scoreFontColorPreview *walk.Composite
var periodFontColorPreview *walk.Composite
var backgroundFontColorPreview *walk.Composite
var extraTimeFontColor walk.Color
var extraTimeFontColorPreview *walk.Composite

type StringListModel struct {
	walk.ListModelBase
	Items []string
}

var (
	lblClock  *walk.Label
	lblScore  *walk.Label
	lblHome   *walk.Label
	lblGuest  *walk.Label
	lblPeriod *walk.Label
)

func (m *StringListModel) ItemCount() int {
	return len(m.Items)
}

func (m *StringListModel) Value(index int) interface{} {
	if index < 0 || index >= len(m.Items) {
		return ""
	}
	return m.Items[index] // <- wichtig: ein string, kein anderes Objekt!
}

var (
	sportsModel         = &StringListModel{Items: []string{"American Football", "Fußball"}}
	gameclockModesModel = &StringListModel{Items: []string{"Aufwärts (MM:SS)", "Aufwärts (Fußball-Minuten)", "Abwärts (MM:SS)"}}
)
var currentLogoData []byte
var templateActionButtons *walk.Composite
var templateSaveButtons *walk.Composite
var (
	teamListView *walk.TableView
	teamsModel   *TeamTableModel
)

var (
	teamTable        *walk.TableView
	teamModel        *TeamTableModel
	sportFilterCombo *walk.ComboBox
)

var (
	matchTable       *walk.TableView
	matchModel       *MatchTableModel
	matchFilterCombo *walk.ComboBox
)

type MatchTableModel struct {
	walk.TableModelBase
	Matches  []*models.Match
	Filtered []*models.Match
}

func (m *MatchTableModel) RowCount() int {
	return len(m.Filtered)
}

func (m *MatchTableModel) Value(row, col int) interface{} {
	team := m.Filtered[row]
	switch col {
	case 0:
		return team.ID // NEU: ID als erste Spalte
	case 1:
		return team.Team1.Name
	case 2:
		return team.Sportart
	case 3:
		return team.Team2.Name
	case 4:
		return team.GameTime.Format("2006-01-02 15:04:05")
	default:
		return ""
	}
}

func (m *MatchTableModel) Sort(col int, order walk.SortOrder) error {
	sort.SliceStable(m.Filtered, func(i, j int) bool {
		less := m.Filtered[i].ID < m.Filtered[j].ID
		if col == 1 {
			less = m.Filtered[i].Sportart < m.Filtered[j].Sportart
		}
		if order == walk.SortDescending {
			return !less
		}
		return less
	})
	return nil
}

func (m *MatchTableModel) ApplyFilter(sportart string) {
	if sportart == "" || sportart == "Alle" {
		m.Filtered = m.Matches
	} else {
		m.Filtered = []*models.Match{}
		for _, team := range m.Matches {
			if team.Sportart == sportart {
				m.Filtered = append(m.Filtered, team)
			}
		}
	}
	m.PublishRowsReset()
}

var (
	templateTable      *walk.TableView
	templateModel      *TemplateTableModel
	templateNameEdit   *walk.LineEdit
	templateSportCombo *walk.ComboBox
	templateForm       *walk.Composite
	currentTemplateID  int
)

type TeamTableModel struct {
	walk.TableModelBase
	Teams    []*models.Team
	Filtered []*models.Team
}

func (m *TeamTableModel) RowCount() int {
	return len(m.Filtered)
}

func (m *TeamTableModel) Value(row, col int) interface{} {
	team := m.Filtered[row]
	switch col {
	case 0:
		return team.ID // NEU: ID als erste Spalte
	case 1:
		return team.Name
	case 2:
		return team.Sportart
	default:
		return ""
	}
}

func (m *TeamTableModel) Sort(col int, order walk.SortOrder) error {
	sort.SliceStable(m.Filtered, func(i, j int) bool {
		less := m.Filtered[i].Name < m.Filtered[j].Name
		if col == 1 {
			less = m.Filtered[i].Sportart < m.Filtered[j].Sportart
		}
		if order == walk.SortDescending {
			return !less
		}
		return less
	})
	return nil
}

func (m *TeamTableModel) ApplyFilter(sportart string) {
	if sportart == "" || sportart == "Alle" {
		m.Filtered = m.Teams
	} else {
		m.Filtered = []*models.Team{}
		for _, team := range m.Teams {
			if team.Sportart == sportart {
				m.Filtered = append(m.Filtered, team)
			}
		}
	}
	m.PublishRowsReset()
}

type TemplateTableModel struct {
	walk.TableModelBase
	Templates []*models.TemplateSettings
}

func (m *TemplateTableModel) RowCount() int {
	return len(m.Templates)
}

func (m *TemplateTableModel) Value(row, col int) interface{} {
	template := m.Templates[row]
	switch col {
	case 0:
		return template.Name
	case 1:
		return template.Sportart
	default:
		return ""
	}
}

func (m *TemplateTableModel) Sort(col int, order walk.SortOrder) error {
	sort.SliceStable(m.Templates, func(i, j int) bool {
		less := m.Templates[i].Name < m.Templates[j].Name
		if col == 1 {
			less = m.Templates[i].Sportart < m.Templates[j].Sportart
		}
		if order == walk.SortDescending {
			return !less
		}
		return less
	})
	return nil
}

func loadIcons() error {

	executablePath, _ := os.Executable()
	iconPath := filepath.Join(filepath.Dir(executablePath), "icons", "new.bmp")

	var err error

	iconNew, err = walk.NewBitmapFromFile(iconPath)
	if err != nil {
		return err
	}
	iconPath = filepath.Join(filepath.Dir(executablePath), "icons", "edit.bmp")
	iconEdit, err = walk.NewBitmapFromFile(iconPath)
	if err != nil {
		return err
	}
	iconPath = filepath.Join(filepath.Dir(executablePath), "icons", "save.bmp")
	iconSave, err = walk.NewBitmapFromFile(iconPath)
	if err != nil {
		return err
	}
	iconPath = filepath.Join(filepath.Dir(executablePath), "icons", "trash.bmp")
	iconDelete, err = walk.NewBitmapFromFile(iconPath)
	if err != nil {
		return err
	}
	iconPath = filepath.Join(filepath.Dir(executablePath), "icons", "cancel.bmp")
	iconCancel, err = walk.NewBitmapFromFile(iconPath)
	if err != nil {
		return err
	}

	log.Printf("Icon New valid: %v", iconNew != nil)
	log.Printf("Icon Edit valid: %v", iconEdit != nil)
	log.Printf("Icon Save valid: %v", iconSave != nil)
	log.Printf("Icon Cancel valid: %v", iconCancel != nil)
	log.Printf("Icon Delete valid: %v", iconDelete != nil)

	return nil
}

var refreshButton *walk.PushButton

func updatePreviewButtonState() {
	if refreshButton != nil {
		refreshButton.SetEnabled(previewOpen)
	}
}
func main() {
	if err := database.InitDatabase(); err != nil {
		log.Fatal("Datenbank konnte nicht initialisiert werden:", err)
	}
	defer database.Close()

	if err := loadIcons(); err != nil {
		log.Fatal("Icons konnten nicht geladen werden:", err)
	}

	teamModel = &TeamTableModel{
		Teams:    []*models.Team{},
		Filtered: []*models.Team{},
	}

	matchModel = &MatchTableModel{
		Matches:  []*models.Match{},
		Filtered: []*models.Match{},
	}

	templateModel = &TemplateTableModel{
		Templates: []*models.TemplateSettings{},
	}

	sportNames, err := loadSports()
	if err != nil {
		log.Fatal("Sportarten konnten nicht geladen werden:", err)
	}

	sportsModel = &StringListModel{Items: sportNames}
	teamNames, err := loadTeams()
	if err != nil {
		log.Fatal("Sportarten konnten nicht geladen werden:", err)
	}

	teamListModel := &StringListModel{Items: teamNames}

	var mw *walk.MainWindow
	var tabs *walk.TabWidget
	err = MainWindow{
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
							Composite{
								AssignTo: &templateForm, // <- Damit wir sichtbar/unsichtbar steuern können
								Visible:  false,         // Start = versteckt
								Layout:   VBox{},
								Children: []Widget{
									GroupBox{
										Title:  "Template Verwaltung",
										Layout: Grid{Columns: 4},
										Children: []Widget{
											Label{Text: "Name:"},
											LineEdit{AssignTo: &templateNameEdit},
											Label{Text: "Passwort:"},
											LineEdit{AssignTo: &passwordEdit, PasswordMode: true, OnEditingFinished: checkPassword},
											Label{Text: "Breite:"},
											NumberEdit{AssignTo: &widthEdit, Enabled: false},
											Label{Text: "Höhe:"},
											NumberEdit{AssignTo: &heightEdit, Enabled: false},
											Label{Text: "X:"},
											NumberEdit{AssignTo: &xEdit, Enabled: false},
											Label{Text: "Y:"},
											NumberEdit{AssignTo: &yEdit, Enabled: false},
										},
									},
									GroupBox{
										Title:  "Sportart- und Periodeneinstellungen",
										Layout: Grid{Columns: 4},
										Children: []Widget{
											Label{Text: "Sportart:"},
											ComboBox{
												AssignTo: &sportSelect,
												Model:    sportsModel,
												Editable: false,
											},

											Label{Text: "Perioden-Label:"},
											LineEdit{
												AssignTo: &periodLabelEdit,
												Text:     "Halbzeit", // Default Wert
											},

											Label{Text: "Anzahl Perioden:"},
											NumberEdit{
												AssignTo: &periodsCountEdit,
												Value:    float64(2), // <-- HIER float64 setzen
												MinValue: float64(1),
												MaxValue: float64(8),
												Decimals: 0,
											},

											Label{Text: "Periodendauer (Minuten):"},
											NumberEdit{
												AssignTo: &periodDurationEdit,
												Value:    float64(45),
												MinValue: float64(1),
												MaxValue: float64(120),
												Decimals: 0,
											},
										},
									},
									GroupBox{
										Title:  "Farbeinstellungen",
										Layout: Grid{Columns: 6},
										Children: []Widget{
											Label{Text: "Uhrfarbe:"},
											Composite{
												Layout: HBox{},
												Children: []Widget{
													PushButton{
														Text: "Farbe wählen",
														OnClicked: func() {
															color, ok := pickColor(nil)
															if ok {
																hex := colorToHex(color)

																// Farbe ins Template schreiben
																clockFontColor = hex

																// Vorschau aktualisieren
																if clockColorPreview != nil {
																	brush, _ := walk.NewSolidColorBrush(color)
																	clockColorPreview.SetBackground(brush)
																}
															}
														},
													},
													Composite{
														AssignTo: &clockColorPreview,
														MinSize:  Size{Width: 20, Height: 20},
														MaxSize:  Size{Width: 20, Height: 20},
														Border:   true,
														Layout:   VBox{},
														Children: []Widget{},
													},
												},
											},
											Label{Text: "Scorefarbe:"},
											Composite{
												Layout: HBox{},
												Children: []Widget{
													PushButton{
														Text: "Farbe wählen",
														OnClicked: func() {
															color, ok := pickColor(nil)
															if ok {
																scoreFontColor = color
																if scoreColorPreview != nil {
																	brush, _ := walk.NewSolidColorBrush(color)
																	scoreColorPreview.SetBackground(brush)
																}
															}
														},
													},
													Composite{
														AssignTo: &scoreColorPreview,
														MinSize:  Size{Width: 20, Height: 20},
														MaxSize:  Size{Width: 20, Height: 20},
														Border:   true,
														Layout:   VBox{}, // <- FEHLT aktuell!
														Children: []Widget{},
													},
												},
											},

											Label{Text: "Periodenfarbe:"},
											Composite{
												Layout: HBox{},
												Children: []Widget{
													PushButton{
														Text: "Farbe wählen",
														OnClicked: func() {
															color, ok := pickColor(nil)
															if ok {
																periodFontColor = color
																if periodColorPreview != nil {
																	brush, _ := walk.NewSolidColorBrush(color)
																	periodColorPreview.SetBackground(brush)
																}
															}
														},
													},
													Composite{
														AssignTo: &periodColorPreview,
														MinSize:  Size{Width: 20, Height: 20},
														MaxSize:  Size{Width: 20, Height: 20},
														Border:   true,
														Layout:   VBox{},
														Children: []Widget{},
													},
												},
											},

											Label{Text: "Hintergrundfarbe:"},
											Composite{
												Layout: HBox{},
												Children: []Widget{
													PushButton{
														Text: "Farbe wählen",
														OnClicked: func() {
															color, ok := pickColor(nil)
															if ok {
																backgroundFontColor = color
																if backgroundColorPreview != nil {
																	brush, _ := walk.NewSolidColorBrush(color)
																	backgroundColorPreview.SetBackground(brush)
																}
															}
														},
													},
													Composite{
														AssignTo: &backgroundColorPreview,
														MinSize:  Size{Width: 20, Height: 20},
														MaxSize:  Size{Width: 20, Height: 20},
														Border:   true,
														Layout:   VBox{},
														Children: []Widget{},
													},
												},
											},

											Label{Text: "Overtime Farbe:"},
											Composite{
												Layout: HBox{},
												Children: []Widget{
													PushButton{
														Text: "Farbe wählen",
														OnClicked: func() {
															color, ok := pickColor(nil)
															if ok {
																extraTimeFontColor = color
																if extraTimeFontColorPreview != nil {
																	brush, _ := walk.NewSolidColorBrush(color)
																	extraTimeFontColorPreview.SetBackground(brush)
																}
															}
														},
													},
													Composite{
														AssignTo: &extraTimeFontColorPreview,
														MinSize:  Size{Width: 20, Height: 20},
														MaxSize:  Size{Width: 20, Height: 20},
														Border:   true,
														Layout:   VBox{},
														Children: []Widget{},
													},
												},
											},
										},
									},
									GroupBox{
										Title:  "Anzeigeoptionen",
										Layout: Grid{Columns: 2},
										Children: []Widget{
											Label{Text: "Gameclock-Modus:"},
											ComboBox{
												AssignTo: &gameclockModeCombo,
												Model:    []string{"Aufwärts (MM:SS)", "Aufwärts (Fußball-Minuten)", "Abwärts (MM:SS)"},
												Editable: false,
											},

											Label{Text: "Periodenanzeige:"},
											CheckBox{
												AssignTo: &showPeriodCB,
												Checked:  true,
											},
											Label{Text: "Gameclock-Anzeige:"},
											CheckBox{
												AssignTo: &showGameclockCB,
												Checked:  true,
												OnCheckedChanged: func() {
													// Nur wenn Gameclock aus, echte Uhr aktivierbar
													if showGameclockCB.Checked() {
														showClockCB.SetChecked(false)
														showClockCB.SetEnabled(false)
													} else {
														showClockCB.SetEnabled(true)
													}
												},
											},
											Label{Text: "Echte Uhrzeit statt Gameclock:"},
											CheckBox{
												AssignTo: &showClockCB,
												Enabled:  false, // Nur aktivierbar, wenn Gameclock deaktiviert
											},
										},
									},
								},
							},
							TableView{
								AssignTo: &templateTable,
								Columns: []TableViewColumn{
									{Title: "Name", Width: 150},
									{Title: "Sportart", Width: 100},
								},
								Model:            templateModel,
								CheckBoxes:       false,
								AlternatingRowBG: true,
								OnCurrentIndexChanged: func() {
									// Optional kannst du hier das Template vorausfüllen lassen
								},
							},
							Composite{
								AssignTo: &templateActionButtons,
								Layout:   HBox{},
								Children: []Widget{
									PushButton{
										Text:  "Neu",
										Image: iconNew,
										OnClicked: func() {
											resetTemplateForm()
											showTemplateForm(nil) // Formular öffnen
										},
									},
									PushButton{
										Text:  "Bearbeiten",
										Image: iconEdit,
										OnClicked: func() {
											editSelectedTemplate()
										},
									},
									PushButton{
										Text:  "Löschen",
										Image: iconDelete,
										OnClicked: func() {
											deleteSelectedTemplate()
										},
									},
								},
							},
							Composite{
								AssignTo: &templateSaveButtons,
								Visible:  false, // Start unsichtbar
								Layout:   HBox{},
								Children: []Widget{
									PushButton{
										Text:  "Speichern",
										Image: iconSave,
										OnClicked: func() {
											saveTemplate()
										},
									},
									PushButton{
										Text:  "Abbrechen",
										Image: iconCancel,
										OnClicked: func() {
											resetTemplateForm()
											hideTemplateForm() // Formular schließen
										},
									},
									PushButton{
										Text: "Vorschau",
										OnClicked: func() {
											if !previewOpen {
												openPreviewWindow()
											} else {
												closePreviewWindow()
											}
										},
									},
									PushButton{
										AssignTo: &refreshButton,
										Text:     "Refresh",
										Enabled:  false, // Start deaktiviert
										OnClicked: func() {
											refreshPreviewWindow()
										},
									},
								},
							},
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
										Model:    sportsModel,
										Editable: false,
									},

									Label{Text: "Logo:"},
									PushButton{
										Text:  "Logo wählen...",
										Image: iconEdit,
										OnClicked: func() {
											chooseLogo()
										},
									},

									Label{Text: "Vorschau:"},
									ImageView{
										AssignTo: &logoPreview,
										MinSize:  Size{Width: 70, Height: 70},
										MaxSize:  Size{Width: 70, Height: 70},
									},

									HSpacer{},
									PushButton{
										Text:  "Team speichern",
										Image: iconSave,
										OnClicked: func() {
											saveTeam()
										},
									},
								},
							},

							VSpacer{},
							Composite{
								Layout: HBox{},
								Children: []Widget{
									Label{Text: "Sportart filtern:"},
									ComboBox{
										AssignTo: &sportFilterCombo,
										Model:    []string{"Alle", "American Football", "Fußball"},
										OnCurrentIndexChanged: func() {
											if teamModel != nil {
												teamModel.ApplyFilter(sportFilterCombo.Text())
											}
										},
									},
									HSpacer{},
									PushButton{
										Text:  "Löschen",
										Image: iconDelete,
										OnClicked: func() {
											deleteSelectedTeam()
										},
									},
								},
							},
							TableView{
								AssignTo: &teamTable,
								Columns: []TableViewColumn{
									{Title: "ID", Width: 0, Hidden: true}, // Versteckt
									{Title: "Teamname", Width: 150},
									{Title: "Sportart", Width: 100},
								},
								Model:            teamModel,
								CheckBoxes:       false,
								AlternatingRowBG: true,
								OnCurrentIndexChanged: func() {
									if index := teamTable.CurrentIndex(); index >= 0 && index < len(teamModel.Filtered) {
										team := teamModel.Filtered[index]
										loadTeam(team)
									}
								},
							},
						},
					},
					{
						Title:  "Spiele",
						Layout: VBox{},
						Children: []Widget{
							GroupBox{
								Title:  "Spieldaten",
								Layout: Grid{Columns: 2},
								Children: []Widget{
									Label{Text: "Heim Team:"},
									ComboBox{
										AssignTo: &heimCombo,
										Model:    teamListModel,
										Editable: false,
									},
									Label{Text: "Sportart filtern:"},
									ComboBox{
										AssignTo: &matchSportFilterCombo,
										Model:    []string{"Alle", "American Football", "Fußball"},
										OnCurrentIndexChanged: func() {
											if matchModel != nil {
												matchModel.ApplyFilter(matchSportFilterCombo.Text())
											}
										},
									},

									Label{Text: "Gast Team:"},
									ComboBox{
										AssignTo: &gastCombo,
										Model:    teamListModel,
										Editable: false,
									},
									PushButton{
										Text:  "Spiel speichern",
										Image: iconSave,
										OnClicked: func() {
											saveMatch()
										},
									},
								},
							},

							VSpacer{},
							Composite{
								Layout: HBox{},
								Children: []Widget{
									HSpacer{},
									PushButton{
										Text:  "Löschen",
										Image: iconDelete,
										OnClicked: func() {
											deleteSelectedMatch()
										},
									},
								},
							},
							TableView{
								AssignTo: &matchTable,
								Columns: []TableViewColumn{
									{Title: "ID", Width: 0, Hidden: true}, // Versteckt
									{Title: "Heim Team", Width: 150},
									{Title: "Sportart", Width: 100},
									{Title: "Gast Team", Width: 150},
									{Title: "Datum Uhrzeit", Width: 150},
								},
								Model:            matchModel,
								CheckBoxes:       false,
								AlternatingRowBG: true,
								OnCurrentIndexChanged: func() {
									if index := matchTable.CurrentIndex(); index >= 0 && index < len(matchModel.Filtered) {
										match := matchModel.Filtered[index]
										loadMatch(match)
									}
								},
							},
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
	mw.Show()
	reloadTeams()
	reloadTemplates()

	sportSelect.SetCurrentIndex(0)
	sportCombo.SetCurrentIndex(0)
	gameclockModeCombo.SetCurrentIndex(0)
	mw.Run()
}

func checkPassword() {
	if passwordEdit.Text() == "Garfield" {
		widthEdit.SetEnabled(true)
		heightEdit.SetEnabled(true)
		xEdit.SetEnabled(true)
		yEdit.SetEnabled(true)
	}
}

func loadTeam(team *models.Team) {
	teamNameEdit.SetText(team.Name)
	sportCombo.SetText(team.Sportart)
	currentLogoData = team.LogoData
	setLogoFromData(team.LogoData)
}

func loadMatch(team *models.Match) {
	//teamNameEdit.SetText(team.Name)
	sportCombo.SetText(team.Sportart)
	//currentLogoData = team.LogoData
	//setLogoFromData(team.LogoData)
}

func deleteSelectedTeam() {
	index := teamTable.CurrentIndex()
	if index < 0 || index >= len(teamModel.Filtered) {
		walk.MsgBox(nil, "Hinweis", "Bitte ein Team auswählen.", walk.MsgBoxIconInformation)
		return
	}

	team := teamModel.Filtered[index]

	// Passwort je Sportart definieren
	var expectedPassword string
	switch team.Sportart {
	case "American Football":
		expectedPassword = "FinishStrong"
	case "Fußball":
		expectedPassword = "Alzstadion"
	default:
		walk.MsgBox(nil, "Fehler", "Unbekannte Sportart, kann nicht gelöscht werden.", walk.MsgBoxIconError)
		return
	}

	// Passwortabfrage
	input, ok := askPassword("Bitte Passwort eingeben:")
	if !ok {
		return // Abgebrochen
	}

	if trim(input) != expectedPassword {
		walk.MsgBox(nil, "Fehler", "Falsches Passwort ("+input+") für Sportart "+team.Sportart+". Team wird nicht gelöscht.", walk.MsgBoxIconError)
		return
	}

	// Wirklich löschen
	if err := database.DeleteTeam(team.ID); err != nil {
		walk.MsgBox(nil, "Fehler", "Fehler beim Löschen des Teams: "+err.Error(), walk.MsgBoxIconError)
		return
	}

	// Neu laden
	reloadTeams()

	walk.MsgBox(nil, "Erfolg", "Team wurde erfolgreich gelöscht.", walk.MsgBoxIconInformation)
}

func deleteSelectedMatch() {
	index := matchTable.CurrentIndex()
	if index < 0 || index >= len(matchModel.Filtered) {
		walk.MsgBox(nil, "Hinweis", "Bitte ein Team auswählen.", walk.MsgBoxIconInformation)
		return
	}

	team := matchModel.Filtered[index]

	// Passwort je Sportart definieren
	var expectedPassword string
	switch team.Sportart {
	case "American Football":
		expectedPassword = "FinishStrong"
	case "Fußball":
		expectedPassword = "Alzstadion"
	default:
		walk.MsgBox(nil, "Fehler", "Unbekannte Sportart, kann nicht gelöscht werden.", walk.MsgBoxIconError)
		return
	}

	// Passwortabfrage
	input, ok := askPassword("Bitte Passwort eingeben:")
	if !ok {
		return // Abgebrochen
	}

	if trim(input) != expectedPassword {
		walk.MsgBox(nil, "Fehler", "Falsches Passwort ("+input+") für Sportart "+team.Sportart+". Team wird nicht gelöscht.", walk.MsgBoxIconError)
		return
	}

	// Wirklich löschen
	if err := database.DeleteMatch(team.ID); err != nil {
		walk.MsgBox(nil, "Fehler", "Fehler beim Löschen des Spiels: "+err.Error(), walk.MsgBoxIconError)
		return
	}

	// Neu laden
	reloadMatches()

	walk.MsgBox(nil, "Erfolg", "Team wurde erfolgreich gelöscht.", walk.MsgBoxIconInformation)
}

func trim(s string) string {
	return strings.TrimSpace(s)
}

func reloadTeams() {
	teams, err := database.LoadTeams()
	if err != nil {
		walk.MsgBox(nil, "Fehler", "Konnte Teams nicht laden: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	teamModel.Teams = teams
	teamModel.ApplyFilter(sportFilterCombo.Text())
	teamTable.SetModel(teamModel)
}

func reloadMatches() {
	matches, err := database.LoadMatches()
	if err != nil {
		walk.MsgBox(nil, "Fehler", "Konnte Spiele nicht laden: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	matchModel.Matches = matches
	matchModel.ApplyFilter(sportFilterCombo.Text())
	matchTable.SetModel(matchModel)
}

func askPassword(prompt string) (string, bool) {
	var input string
	var in *walk.LineEdit
	var dlg *walk.Dialog

	err := Dialog{
		AssignTo: &dlg,
		Title:    "Passwort eingeben",
		MinSize:  Size{Width: 300, Height: 120},
		Layout:   VBox{},
		Children: []Widget{
			Label{
				Text: prompt,
			},
			LineEdit{
				AssignTo:     &in,
				PasswordMode: true,
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "OK",
						OnClicked: func() {
							input = strings.TrimSpace(in.Text()) // Text sofort sichern
							dlg.Accept()
						},
					},
					PushButton{
						Text: "Abbrechen",
						OnClicked: func() {
							dlg.Cancel()
						},
					},
				},
			},
		},
	}.Create(nil)

	if err != nil {
		return "", false
	}

	if dlg.Run() == walk.DlgCmdOK {
		return input, true
	}
	return "", false
}

func chooseLogo() {
	dlg := new(walk.FileDialog)
	dlg.Filter = "PNG Bilder (*.png)|*.png"

	if ok, _ := dlg.ShowOpen(nil); ok {
		// Datei öffnen
		data, err := os.ReadFile(dlg.FilePath)
		if err != nil {
			walk.MsgBox(nil, "Fehler", "Bild konnte nicht geladen werden.", walk.MsgBoxIconError)
			return
		}

		// PNG dekodieren
		src, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			walk.MsgBox(nil, "Fehler", "Bild ist kein gültiges PNG.", walk.MsgBoxIconError)
			return
		}

		// Neues 70x70 NRGBA-Bild
		dst := image.NewNRGBA(image.Rect(0, 0, 70, 70))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

		// Bitmap für die Vorschau
		bmp, err := walk.NewBitmapFromImage(dst)
		if err != nil {
			walk.MsgBox(nil, "Fehler", "Vorschau konnte nicht erstellt werden.", walk.MsgBoxIconError)
			return
		}

		logoPreview.SetImage(bmp)

		// LogoDaten zwischenspeichern
		currentLogoData = data // <-- neue Variable, siehe unten
	}
}

func saveTeam() {
	logoData := currentLogoData
	if logoData == nil {
		logoData = []byte{}
	}

	var id int64
	if index := teamTable.CurrentIndex(); index >= 0 && index < len(teamModel.Filtered) {
		id = int64(teamModel.Filtered[index].ID)
	}

	team := &models.Team{
		ID:       int(id), // falls 0 → wird Insert, sonst Update
		Name:     teamNameEdit.Text(),
		Sportart: sportCombo.Text(),
		LogoData: logoData,
	}

	if err := database.SaveTeam(team); err != nil {
		walk.MsgBox(nil, "Fehler", "Team konnte nicht gespeichert werden:\n"+err.Error(), walk.MsgBoxIconError)
		log.Printf("Fehler beim Speichern: %+v", err)
		return
	}

	walk.MsgBox(nil, "Erfolg", "Team erfolgreich gespeichert.", walk.MsgBoxIconInformation)

	// Formular zurücksetzen
	resetForm()
	teamTable.SetCurrentIndex(-1)
	reloadTeams()
}

func saveMatch() {
	logoData := currentLogoData
	if logoData == nil {
		logoData = []byte{}
	}

	var id int64
	if index := matchTable.CurrentIndex(); index >= 0 && index < len(matchModel.Filtered) {
		id = int64(matchModel.Filtered[index].ID)
	}

	match := &models.Match{
		ID:       int(id), // falls 0 → wird Insert, sonst Update
		Sportart: sportCombo.Text(),
	}

	if err := database.SaveMatches(match); err != nil {
		walk.MsgBox(nil, "Fehler", "Match konnte nicht gespeichert werden:\n"+err.Error(), walk.MsgBoxIconError)
		log.Printf("Fehler beim Speichern: %+v", err)
		return
	}

	walk.MsgBox(nil, "Erfolg", "Match erfolgreich gespeichert.", walk.MsgBoxIconInformation)

	// Formular zurücksetzen
	resetMatchForm()
	matchTable.SetCurrentIndex(-1)
	reloadMatches()
}

func resetForm() {
	teamNameEdit.SetText("")
	sportCombo.SetCurrentIndex(0)
	logoPreview.SetImage(nil)
	currentLogoData = nil
}

func resetMatchForm() {
	sportCombo.SetCurrentIndex(0)
}

func setLogoFromData(data []byte) {
	if data == nil || len(data) == 0 {
		logoPreview.SetImage(nil)
		return
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("Fehler beim Dekodieren des Logos: %v", err)
		logoPreview.SetImage(nil)
		return
	}

	dst := image.NewNRGBA(image.Rect(0, 0, 70, 70))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)

	bmp, err := walk.NewBitmapFromImage(dst)
	if err != nil {
		log.Printf("Fehler beim Erstellen der Vorschau: %v", err)
		logoPreview.SetImage(nil)
		return
	}

	logoPreview.SetImage(bmp)
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

func loadSports() ([]string, error) {
	sports, err := database.LoadSports()
	if err != nil {
		return nil, err
	}

	var names []string
	for _, s := range sports {
		names = append(names, s.Sportart)
	}
	return names, nil
}

func loadTeams() ([]string, error) {

	teams, err := database.LoadTeams()
	if err != nil {
		return nil, err
	}

	var names []string
	for _, t := range teams {
		names = append(names, t.Name)
	}
	return names, nil
}

func showTemplateForm(t *models.TemplateSettings) {
	templateForm.SetVisible(true)
	templateActionButtons.SetVisible(false)
	templateSaveButtons.SetVisible(true)

	if t != nil {
		templateNameEdit.SetText(t.Name)
		sportSelect.SetText(t.Sportart)
		widthEdit.SetValue(float64(t.Width))
		heightEdit.SetValue(float64(t.Height))
		xEdit.SetValue(float64(t.X))
		yEdit.SetValue(float64(t.Y))
		sportSelect.SetText(t.Sportart)
		periodLabelEdit.SetText(t.PeriodLabel)
		periodsCountEdit.SetValue(float64(t.PeriodsCount))
		periodDurationEdit.SetValue(float64(t.PeriodDuration))
		gameclockModeCombo.SetText(t.GameclockMode)
		showPeriodCB.SetChecked(t.ShowPeriod)
		showGameclockCB.SetChecked(t.ShowGameclock)
		showClockCB.SetChecked(t.ShowClock)

		if t.ClockFontColor != "" {
			color, _ := parseHexColor(t.ClockFontColor)
			brush, _ := walk.NewSolidColorBrush(color)
			clockColorPreview.SetBackground(brush)
			clockFontColor = t.ClockFontColor // Update aktuelle Variable
		}
		if t.ScoreFontColor != "" {
			color, _ := parseHexColor(t.ScoreFontColor)
			brush, _ := walk.NewSolidColorBrush(color)
			scoreColorPreview.SetBackground(brush)
			scoreFontColor = color
		}
		if t.PeriodFontColor != "" {
			color, _ := parseHexColor(t.PeriodFontColor)
			brush, _ := walk.NewSolidColorBrush(color)
			periodColorPreview.SetBackground(brush)
			periodFontColor = color
		}
		if t.BackgroundFontColor != "" {
			color, _ := parseHexColor(t.BackgroundFontColor)
			brush, _ := walk.NewSolidColorBrush(color)
			backgroundColorPreview.SetBackground(brush)
			backgroundFontColor = color
		}
		if t.ExtraTimeFontColor != "" {
			color, _ := parseHexColor(t.ExtraTimeFontColor)
			brush, _ := walk.NewSolidColorBrush(color)
			extraTimeFontColorPreview.SetBackground(brush)
			extraTimeFontColor = color
		}

		currentTemplateID = t.ID
	} else {
		resetTemplateForm()
	}
}

func hideTemplateForm() {
	templateForm.SetVisible(false)
	templateActionButtons.SetVisible(true)
	templateSaveButtons.SetVisible(false)
}

func editSelectedTemplate() {
	index := templateTable.CurrentIndex()
	if index < 0 || index >= len(templateModel.Templates) {
		return
	}

	t := templateModel.Templates[index]

	expectedPassword := expectedTemplatePassword(t.Sportart)
	if expectedPassword == "" {
		walk.MsgBox(nil, "Fehler", "Unbekannte Sportart, kann nicht bearbeitet werden.", walk.MsgBoxIconError)
		return
	}

	input, ok := askPassword("Bitte Passwort für Bearbeiten eingeben:")
	if !ok {
		return
	}

	if trim(input) != expectedPassword {
		walk.MsgBox(nil, "Fehler", "Falsches Passwort für Sportart "+t.Sportart+".", walk.MsgBoxIconError)
		return
	}

	showTemplateForm(t)
}

func saveTemplate() {
	t := &models.TemplateSettings{
		ID:             currentTemplateID,
		Name:           templateNameEdit.Text(),
		Width:          int(widthEdit.Value()),
		Height:         int(heightEdit.Value()),
		X:              int(xEdit.Value()),
		Y:              int(yEdit.Value()),
		Sportart:       sportSelect.Text(),
		PeriodLabel:    periodLabelEdit.Text(),
		PeriodsCount:   int(periodsCountEdit.Value()),
		PeriodDuration: int(periodDurationEdit.Value()),
		GameclockMode:  gameclockModeCombo.Text(),
		ShowPeriod:     showPeriodCB.Checked(),
		ShowGameclock:  showGameclockCB.Checked(),
		ShowClock:      showClockCB.Checked(),

		ClockFontColor:      clockFontColor,
		ScoreFontColor:      colorToHex(scoreFontColor),
		PeriodFontColor:     colorToHex(periodFontColor),
		BackgroundFontColor: colorToHex(backgroundFontColor),
		ExtraTimeFontColor:  colorToHex(extraTimeFontColor),
	}

	if err := database.SaveTemplate(t); err != nil {
		walk.MsgBox(nil, "Fehler", "Template konnte nicht gespeichert werden:\n"+err.Error(), walk.MsgBoxIconError)
		return
	}

	reloadTemplates()
	resetTemplateForm()
	hideTemplateForm()
}

func resetTemplateForm() {
	currentTemplateID = 0
	templateNameEdit.SetText("")
	passwordEdit.SetText("")
	widthEdit.SetValue(0)
	heightEdit.SetValue(0)
	xEdit.SetValue(0)
	yEdit.SetValue(0)
	sportSelect.SetCurrentIndex(0)
	periodLabelEdit.SetText("")
	periodsCountEdit.SetValue(0)
	periodDurationEdit.SetValue(0)
	gameclockModeCombo.SetCurrentIndex(0)
	showPeriodCB.SetChecked(false)
	showGameclockCB.SetChecked(false)
	showClockCB.SetChecked(false)
}
func reloadTemplates() {
	templates, err := database.LoadTemplates()
	if err != nil {
		walk.MsgBox(nil, "Fehler", "Konnte Templates nicht laden: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	templateModel.Templates = templates
	templateModel.PublishRowsReset()
	templateTable.SetModel(templateModel)
}

func deleteSelectedTemplate() {
	index := templateTable.CurrentIndex()
	if index < 0 || index >= len(templateModel.Templates) {
		walk.MsgBox(nil, "Hinweis", "Bitte ein Template auswählen.", walk.MsgBoxIconInformation)
		return
	}

	t := templateModel.Templates[index]

	expectedPassword := expectedTemplatePassword(t.Sportart)
	if expectedPassword == "" {
		walk.MsgBox(nil, "Fehler", "Unbekannte Sportart, kann nicht gelöscht werden.", walk.MsgBoxIconError)
		return
	}

	input, ok := askPassword("Bitte Passwort für Löschen eingeben:")
	if !ok {
		return
	}

	if trim(input) != expectedPassword {
		walk.MsgBox(nil, "Fehler", "Falsches Passwort für Sportart "+t.Sportart+".", walk.MsgBoxIconError)
		return
	}

	if err := database.DeleteTemplate(t.ID); err != nil {
		walk.MsgBox(nil, "Fehler", "Template konnte nicht gelöscht werden: "+err.Error(), walk.MsgBoxIconError)
		return
	}

	reloadTemplates()
	walk.MsgBox(nil, "Erfolg", "Template erfolgreich gelöscht.", walk.MsgBoxIconInformation)
}

func expectedTemplatePassword(sportart string) string {
	switch sportart {
	case "American Football":
		return "FinishStrong"
	case "Fußball":
		return "Alzstadion"
	default:
		return ""
	}
}
func openPreviewWindow() {
	if previewWindow != nil {
		return
	}

	index := templateTable.CurrentIndex()
	if index < 0 || index >= len(templateModel.Templates) {
		return
	}
	t := templateModel.Templates[index]

	// Dummy-Logo laden
	executablePath, _ := os.Executable()
	dummyPath := filepath.Join(filepath.Dir(executablePath), "icons", "logo.png")

	dummyImage, err := walk.NewBitmapFromFile(dummyPath)
	if err != nil {
		log.Printf("Fehler beim Laden des Dummy-Logos: %v", err)
		dummyImage = nil
	}

	err = MainWindow{
		AssignTo: &previewWindow,
		Title:    "Scoreboard Vorschau",
		Bounds: Rectangle{
			X:      t.X,
			Y:      t.Y,
			Width:  t.Width,
			Height: t.Height,
		},
		Layout: VBox{},
		Children: []Widget{
			Composite{
				Layout: VBox{},
				Children: []Widget{
					Label{
						Text:      "00:00",
						Font:      Font{Family: defaultFontFamily, PointSize: defaultClockSize},
						Alignment: AlignHCenterVCenter,
						TextColor: mustParseColor(t.ClockFontColor),
					},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							ImageView{
								Image:   dummyImage,
								MinSize: Size{Width: 70, Height: 70},
							},
							Label{
								Text:      "7 : 3",
								Font:      Font{Family: defaultFontFamily, PointSize: defaultScoreSize},
								Alignment: AlignHCenterVCenter,
								TextColor: mustParseColor(t.ScoreFontColor),
							},
							ImageView{
								Image:   dummyImage,
								MinSize: Size{Width: 70, Height: 70},
							},
						},
					},
					Label{
						Text:      t.PeriodLabel,
						Font:      Font{Family: defaultFontFamily, PointSize: defaultPeriodSize},
						Alignment: AlignHCenterVCenter,
						TextColor: mustParseColor(t.PeriodFontColor),
					},
				},
			},
		},
	}.Create()
	if err != nil {
		log.Printf("Fehler bei openPreviewWindow: %v", err)
		return
	}

	// Frameless:
	win.SetWindowLong(previewWindow.Handle(), win.GWL_STYLE, 0)

	// Always on top:
	win.SetWindowPos(previewWindow.Handle(), win.HWND_TOPMOST, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE)

	// Hintergrundfarbe setzen
	if t.BackgroundFontColor != "" {
		color, err := parseHexColor(t.BackgroundFontColor)
		if err == nil {
			brush, _ := walk.NewSolidColorBrush(color)
			previewWindow.SetBackground(brush)
		}
	}

	previewWindow.Show()
	previewWindow.SetVisible(true)

	previewOpen = true
	updatePreviewButtonState()
}
func mustParseColor(hex string) walk.Color {
	if hex == "" {
		return walk.RGB(255, 255, 255) // Default Weiß
	}
	color, err := parseHexColor(hex)
	if err != nil {
		return walk.RGB(255, 255, 255)
	}
	return color
}
func closePreviewWindow() {
	if previewWindow != nil {
		previewWindow.Dispose()
		previewWindow = nil
	}
	previewOpen = false
	updatePreviewButtonState()
}

// Hilfsfunktion, um Farbcodes wie "#FFFFFF" zu parsen
func parseHexColor(s string) (walk.Color, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return 0, fmt.Errorf("ungültiger Farbcode: %s", s)
	}
	var rgb uint64
	rgb, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, err
	}
	return walk.RGB(byte(rgb>>16), byte((rgb>>8)&0xFF), byte(rgb&0xFF)), nil
}
func refreshPreviewWindow() {
	if previewWindow == nil || previewContent == nil {
		return
	}

	index := templateTable.CurrentIndex()
	if index < 0 || index >= len(templateModel.Templates) {
		return
	}
	t := templateModel.Templates[index]

	log.Printf("refreshPreviewWindow: Template: %+v", t)

	// Clock
	if t.ShowGameclock || t.ShowClock {
		lblClock.SetVisible(true)
		lblClock.SetText("00:00")
		fontClock, _ := walk.NewFont(defaultFontFamily, defaultClockSize, 0)
		lblClock.SetFont(fontClock)
		if color, err := parseHexColor(t.ClockFontColor); err == nil {
			lblClock.SetTextColor(color)
		}
	} else {
		lblClock.SetVisible(false)
	}

	// Score
	lblHome.SetText("🏈")
	lblScore.SetText("7 : 3")
	lblGuest.SetText("⚽")

	fontScore, _ := walk.NewFont(defaultFontFamily, defaultScoreSize, 0)
	lblScore.SetFont(fontScore)
	if color, err := parseHexColor(t.ScoreFontColor); err == nil {
		lblScore.SetTextColor(color)
	}

	// Period
	if t.ShowPeriod {
		lblPeriod.SetVisible(true)
		lblPeriod.SetText(t.PeriodLabel)
		fontPeriod, _ := walk.NewFont(defaultFontFamily, defaultPeriodSize, 0)
		lblPeriod.SetFont(fontPeriod)
		if color, err := parseHexColor(t.PeriodFontColor); err == nil {
			lblPeriod.SetTextColor(color)
		}
	} else {
		lblPeriod.SetVisible(false)
	}

	// Hintergrundfarbe
	if color, err := parseHexColor(t.BackgroundFontColor); err == nil {
		brush, _ := walk.NewSolidColorBrush(color)
		previewContent.SetBackground(brush)
	}

	previewContent.Invalidate()
	previewWindow.Invalidate()
}

func pickColor(owner walk.Form) (walk.Color, bool) {
	var customColors [16]win.COLORREF

	hwnd := win.HWND(0)
	if owner != nil {
		hwnd = win.HWND(owner.Handle())
	}

	chooseColor := win.CHOOSECOLOR{
		LStructSize:  uint32(unsafe.Sizeof(win.CHOOSECOLOR{})),
		HwndOwner:    hwnd,
		Flags:        win.CC_FULLOPEN | win.CC_RGBINIT,
		RgbResult:    0xFFFFFF, // Startfarbe Weiß
		LpCustColors: &customColors,
	}

	if win.ChooseColor(&chooseColor) {
		return walk.Color(chooseColor.RgbResult), true
	}
	return 0, false
}
func colorToHex(c walk.Color) string {
	r := byte(c)
	g := byte(c >> 8)
	b := byte(c >> 16)

	return strings.ToUpper("#" + toHex(r) + toHex(g) + toHex(b))
}

func toHex(b byte) string {
	const hexdigits = "0123456789ABCDEF"
	return string([]byte{hexdigits[b>>4], hexdigits[b&0xF]})
}
