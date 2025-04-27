// internal/database/database.go

package database

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/KernTom/scoreboard-manager/internal/models"
	_ "modernc.org/sqlite"
)

type TemplateSettings struct {
	Width          int
	Height         int
	X              int
	Y              int
	Sport          string
	PeriodLabel    string
	PeriodCount    int
	PeriodDuration int
	GameclockMode  string
	ShowPeriod     bool
	ShowGameclock  bool
	ShowClock      bool
}

type Sport struct {
	ID             int
	Name           string
	PeriodLabel    string
	PeriodCount    int
	PeriodDuration int
	GameclockMode  string
}

var db *sql.DB

func InitDatabase() error {
	var err error
	dbPath := "settings.db"

	// Prüfen, ob Datei existiert
	if _, err = os.Stat(dbPath); os.IsNotExist(err) {
		file, err := os.Create(dbPath)
		if err != nil {
			return err
		}
		file.Close()
	}

	// DB öffnen
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	// Prüfen ob Tabellen existieren
	if err := checkAndMigrateDatabase(); err != nil {
		return err
	}

	err = migrateDatabase()
	if err != nil {
		return err
	}

	return nil
}

func checkAndMigrateDatabase() error {
	// Check ob Tabelle "template_settings" existiert

	// Tabelle(n) fehlt → komplette Struktur neu anlegen
	if createErr := createTables(); createErr != nil {
		return createErr
	}
	if insertErr := insertDefaultSports(); insertErr != nil {
		return insertErr
	}

	return nil
}

func createTables() error {
	sqlStmts := []string{
		`CREATE TABLE IF NOT EXISTS template_settings (
			id INTEGER PRIMARY KEY,
			width INTEGER,
			height INTEGER,
			x INTEGER,
			y INTEGER,
			sport TEXT,
			period_label TEXT,
			period_count INTEGER,
			period_duration INTEGER,
			gameclock_mode TEXT,
			show_period BOOLEAN,
			show_gameclock BOOLEAN,
			show_clock BOOLEAN
		);`,
		`CREATE TABLE IF NOT EXISTS sports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sportart TEXT NOT NULL UNIQUE,
			period_label TEXT,
			periods_count INTEGER,
			period_duration INTEGER,
			clock_format TEXT,
			clock_direction TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			sportart TEXT NOT NULL,
			logo_data BLOB
		);`,
		`CREATE TABLE IF NOT EXISTS matches (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sportart TEXT NOT NULL,
			team_home int not null,
			team_away int not null,
			template_id int not null,
			start_time datetime
		);`,
	}

	for _, stmt := range sqlStmts {
		_, err := db.Exec(stmt)
		if err != nil {
			return err
		}
	}

	return nil
}

func migrateDatabase() error {
	// Tabellen und erwartete Spalten samt Typen
	tableColumns := map[string]map[string]string{
		"template_settings": {
			"clock_font_family":     "TEXT DEFAULT 'Segoe UI'",
			"clock_font_size":       "INTEGER DEFAULT 32",
			"clock_font_color":      "TEXT DEFAULT '#FFFFFF'",
			"period_font_family":    "TEXT DEFAULT 'Segoe UI'",
			"period_font_size":      "INTEGER DEFAULT 20",
			"period_font_color":     "TEXT DEFAULT '#FFFFFF'",
			"score_font_family":     "TEXT DEFAULT 'Segoe UI'",
			"score_font_size":       "INTEGER DEFAULT 32",
			"score_font_color":      "TEXT DEFAULT '#FFFFFF'",
			"separator_font_family": "TEXT DEFAULT 'Segoe UI'",
			"separator_font_size":   "INTEGER DEFAULT 28",
			"separator_font_color":  "TEXT DEFAULT '#FFFFFF'",
			"extra_time_font_color": "TEXT DEFAULT '#FF0000'",
			"name":                  "TEXT DEFAULT 'Standard'",
			"background_font_color": "TEXT DEFAULT '#000000'",
		},
		"teams": {
			"logo_data": "BLOB",
		},
	}

	for tableName, columns := range tableColumns {
		existing, err := getExistingColumns(tableName)
		if err != nil {
			return err
		}

		for col, definition := range columns {
			if _, ok := existing[col]; !ok {
				// Spalte fehlt → hinzufügen
				query := "ALTER TABLE " + tableName + " ADD COLUMN " + col + " " + definition + ";"
				_, err := db.Exec(query)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func insertDefaultSports() error {
	defaultSports := []models.SportartDefinition{
		{Sportart: "American Football", PeriodLabel: "Halbzeit", PeriodsCount: 2, PeriodDuration: 15, ClockFormat: "MM:SS", ClockDirection: "Up"},
		{Sportart: "Fußball", PeriodLabel: "Halbzeit", PeriodsCount: 2, PeriodDuration: 45, ClockFormat: "Minuten", ClockDirection: "Up"},
	}

	for _, sport := range defaultSports {
		_, err := db.Exec(`INSERT OR IGNORE INTO sports (sportart, period_label, periods_count, period_duration, clock_format, clock_direction)
			VALUES (?, ?, ?, ?, ?, ?)`,
			sport.Sportart, sport.PeriodLabel, sport.PeriodsCount, sport.PeriodDuration, sport.ClockFormat, sport.ClockDirection)
		if err != nil {
			return err
		}
	}

	return nil
}

// LoadTemplateSettings lädt die Anzeigeeinstellungen (TemplateSettings) aus der Datenbank
func LoadTemplates() ([]*models.TemplateSettings, error) {
	rows, err := db.Query(`SELECT id,name, width, height, x, y, sport, period_label, period_count, period_duration,
			gameclock_mode, show_period, show_gameclock, show_clock,
			clock_font_family, clock_font_size, clock_font_color,
			period_font_family, period_font_size, period_font_color,
			score_font_family, score_font_size, score_font_color,
			separator_font_family, separator_font_size, separator_font_color,
			extra_time_font_color, background_font_color
		FROM template_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*models.TemplateSettings
	for rows.Next() {
		var ts models.TemplateSettings
		err := rows.Scan(
			&ts.ID,
			&ts.Name,
			&ts.Width,
			&ts.Height,
			&ts.X,
			&ts.Y,
			&ts.Sportart,
			&ts.PeriodLabel,
			&ts.PeriodsCount,
			&ts.PeriodDuration,
			&ts.GameclockMode,
			&ts.ShowPeriod,
			&ts.ShowGameclock,
			&ts.ShowClock,
			&ts.ClockFontFamily,
			&ts.ClockFontSize,
			&ts.ClockFontColor,
			&ts.PeriodFontFamily,
			&ts.PeriodFontSize,
			&ts.PeriodFontColor,
			&ts.ScoreFontFamily,
			&ts.ScoreFontSize,
			&ts.ScoreFontColor,
			&ts.SeparatorFontFamily,
			&ts.SeparatorFontSize,
			&ts.SeparatorFontColor,
			&ts.ExtraTimeFontColor,
			&ts.BackgroundFontColor,
		)
		if err != nil {
			return nil, err
		}
		templates = append(templates, &ts)
	}
	return templates, nil
}

// SaveTemplateSettings speichert die Anzeigeeinstellungen (TemplateSettings) in die Datenbank
func SaveTemplate(template *models.TemplateSettings) error {
	log.Printf("speichere template: %v", template)
	if template.ID == 0 {
		// Neu
		res, err := db.Exec(`
			INSERT INTO template_settings (
				width, height, x, y, sport, period_label, period_count, period_duration,
				gameclock_mode, show_period, show_gameclock, show_clock,
				clock_font_family, clock_font_size, clock_font_color,
				period_font_family, period_font_size, period_font_color,
				score_font_family, score_font_size, score_font_color,
				separator_font_family, separator_font_size, separator_font_color,
				extra_time_font_color, name, background_font_color
			) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,?,?)
		`,
			template.Width,
			template.Height,
			template.X,
			template.Y,
			template.Sportart,
			template.PeriodLabel,
			template.PeriodsCount,
			template.PeriodDuration,
			template.GameclockMode,
			template.ShowPeriod,
			template.ShowGameclock,
			template.ShowClock,
			template.ClockFontFamily,
			template.ClockFontSize,
			template.ClockFontColor,
			template.PeriodFontFamily,
			template.PeriodFontSize,
			template.PeriodFontColor,
			template.ScoreFontFamily,
			template.ScoreFontSize,
			template.ScoreFontColor,
			template.SeparatorFontFamily,
			template.SeparatorFontSize,
			template.SeparatorFontColor,
			template.ExtraTimeFontColor,
			template.Name,
			template.BackgroundFontColor,
		)
		if err != nil {
			return err
		}
		lastID, _ := res.LastInsertId()
		template.ID = int(lastID)
	} else {
		// Update
		_, err := db.Exec(`
			UPDATE template_settings SET
				width = ?, height = ?, x = ?, y = ?, sport = ?, period_label = ?, period_count = ?, period_duration = ?,
				gameclock_mode = ?, show_period = ?, show_gameclock = ?, show_clock = ?,
				clock_font_family = ?, clock_font_size = ?, clock_font_color = ?,
				period_font_family = ?, period_font_size = ?, period_font_color = ?,
				score_font_family = ?, score_font_size = ?, score_font_color = ?,
				separator_font_family = ?, separator_font_size = ?, separator_font_color = ?,
				extra_time_font_color = ?, name = ?, background_font_color = ?
			WHERE id = ?
		`,
			template.Width,
			template.Height,
			template.X,
			template.Y,
			template.Sportart,
			template.PeriodLabel,
			template.PeriodsCount,
			template.PeriodDuration,
			template.GameclockMode,
			template.ShowPeriod,
			template.ShowGameclock,
			template.ShowClock,
			template.ClockFontFamily,
			template.ClockFontSize,
			template.ClockFontColor,
			template.PeriodFontFamily,
			template.PeriodFontSize,
			template.PeriodFontColor,
			template.ScoreFontFamily,
			template.ScoreFontSize,
			template.ScoreFontColor,
			template.SeparatorFontFamily,
			template.SeparatorFontSize,
			template.SeparatorFontColor,
			template.ExtraTimeFontColor,
			template.Name,
			template.BackgroundFontColor,
			template.ID,
		)
		if err != nil {
			log.Printf("Fehler beim speichern des templates: %v", err)
			return err
		}
	}

	return nil
}

// SaveTemplateSettings speichert die Anzeigeeinstellungen (TemplateSettings) in die Datenbank
func SaveMatches(match *models.Match) error {
	if match.ID == 0 {
		// Neu
		res, err := db.Exec(`
			INSERT INTO matches (
				sportart, team_home, team_away, template_id, start_time
			) VALUES ( ?, ?, ?, ?, ?)
		`,
			match.Sportart,
			match.Team1.ID,
			match.Team2.ID,
			match.TemplateSettings.ID,
			match.GameTime.Format(time.RFC3339),
		)
		if err != nil {
			return err
		}
		lastID, _ := res.LastInsertId()
		match.ID = int(lastID)
	} else {
		// Update
		_, err := db.Exec(`
			UPDATE matches SET
				sportart = ?, team_home = ?, team_away = ?, template_id = ?, start_time = ?
			WHERE id = ?
		`,
			match.Sportart,
			match.Team1.ID,
			match.Team2.ID,
			match.TemplateSettings.ID,
			match.GameTime.Format(time.RFC3339),
			match.ID,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// LoadMatches lädt die Matches aus der Datenbank
func LoadMatches() ([]*models.Match, error) {
	rows, err := db.Query(`
		SELECT 
			id, sportart, team_home, team_away, template_id, start_time
			t1.name,
			t2.name,
			ts.name
		FROM matches m
		JOIN teams t1 ON m.team_home = t1.id
		JOIN teams t2 ON m.team_away = t2.id
		JOIN template_settings ts ON m.template_id = ts.id
		ORDER BY m.start_time DESC
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*models.Match
	for rows.Next() {
		var ms models.Match
		err := rows.Scan(
			&ms.ID,
			&ms.Sportart,
			&ms.Team1.ID,
			&ms.Team2.ID,
			&ms.TemplateSettings.ID,
			&ms.GameTime,
			&ms.Team1.Name,
			&ms.Team2.Name,
			&ms.TemplateSettings.Name,
		)
		if err != nil {
			return nil, err
		}
		matches = append(matches, &ms)
	}
	return matches, nil
}

// LoadMatches lädt die Matches aus der Datenbank
func LoadSingleMatch(id int) (m models.Match, err error) {
	err = db.QueryRow(`
		SELECT 
			id, sportart, team_home, team_away, template_id, start_time
			t1.name, t1.logo_data,
			t2.name, t2.logo_data,
			ts.period_label, ts.periods_count, ts.period_duration,
			ts.gameclock_mode, ts.show_period, ts.show_gameclock, ts.show_clock,
			ts.clock_font_family, ts.clock_font_size, ts.clock_font_color,
			ts.period_font_family, ts.period_font_size, ts.period_font_color,
			ts.score_font_family, ts.score_font_size, ts.score_font_color,
			ts.separator_font_family, ts.separator_font_size, ts.separator_font_color,
			ts.extra_time_font_color, ts.name
		FROM matches m
		JOIN teams t1 ON m.team_home = t1.id
		JOIN teams t2 ON m.team_away = t2.id
		JOIN template_settings ts ON m.template_id = ts.id
		WHERE id = ?
		`, id).Scan(
		&m.ID,
		&m.Sportart,
		&m.Team1.ID,
		&m.Team2.ID,
		&m.TemplateSettings.ID,
		&m.GameTime,
		&m.Team1.Name,
		&m.Team1.LogoData,
		&m.Team2.Name,
		&m.Team2.LogoData,
		&m.TemplateSettings.PeriodLabel,
		&m.TemplateSettings.PeriodsCount,
		&m.TemplateSettings.PeriodDuration,
		&m.TemplateSettings.GameclockMode,
		&m.TemplateSettings.ShowPeriod,
		&m.TemplateSettings.ShowGameclock,
		&m.TemplateSettings.ShowClock,
		&m.TemplateSettings.ClockFontFamily,
		&m.TemplateSettings.ClockFontSize,
		&m.TemplateSettings.ClockFontColor,
		&m.TemplateSettings.PeriodFontFamily,
		&m.TemplateSettings.PeriodFontSize,
		&m.TemplateSettings.PeriodFontColor,
		&m.TemplateSettings.ScoreFontFamily,
		&m.TemplateSettings.ScoreFontSize,
		&m.TemplateSettings.ScoreFontColor,
		&m.TemplateSettings.SeparatorFontFamily,
		&m.TemplateSettings.SeparatorFontSize,
		&m.TemplateSettings.SeparatorFontColor,
		&m.TemplateSettings.ExtraTimeFontColor,
		&m.TemplateSettings.Name,
	)
	if err != nil {
		return
	}

	return
}

func DeleteMatch(id int) error {
	_, err := db.Exec(`DELETE FROM matches WHERE id = ?`, id)
	return err
}

func DeleteTemplate(id int) error {
	_, err := db.Exec(`DELETE FROM template_settings WHERE id = ?`, id)
	return err
}

func Close() {
	if db != nil {
		db.Close()
	}
}

// LoadTeams lädt alle Teams aus der Datenbank
func LoadTeams() ([]*models.Team, error) {
	rows, err := db.Query(`SELECT id, name, sportart, logo_data FROM teams`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []*models.Team
	for rows.Next() {
		var team models.Team
		if err := rows.Scan(&team.ID, &team.Name, &team.Sportart, &team.LogoData); err != nil {
			return nil, err
		}
		teams = append(teams, &team)
	}
	return teams, nil
}

func SaveTeam(team *models.Team) error {
	if team.ID == 0 {
		// Neues Team einfügen
		res, err := db.Exec(`INSERT INTO teams (name, sportart, logo_data) VALUES (?, ?, ?)`,
			team.Name, team.Sportart, team.LogoData)
		if err != nil {
			return err
		}
		lastID, err := res.LastInsertId()
		if err == nil {
			team.ID = int(lastID)
		}
	} else {
		// Bestehendes Team updaten
		_, err := db.Exec(`UPDATE teams SET name = ?, sportart = ?, logo_data = ? WHERE id = ?`,
			team.Name, team.Sportart, team.LogoData, team.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteTeam löscht ein Team anhand seiner ID
func DeleteTeam(teamID int) error {
	_, err := db.Exec(`DELETE FROM teams WHERE id = ?`, teamID)
	return err
}

// Sportarten laden
func LoadSports() ([]*models.SportartDefinition, error) {
	rows, err := db.Query(`SELECT id, sportart, period_label, periods_count, period_duration, clock_format, clock_direction FROM sports`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sports []*models.SportartDefinition
	for rows.Next() {
		var sport models.SportartDefinition
		if err := rows.Scan(&sport.ID, &sport.Sportart, &sport.PeriodLabel, &sport.PeriodsCount, &sport.PeriodDuration, &sport.ClockFormat, &sport.ClockDirection); err != nil {
			return nil, err
		}
		sports = append(sports, &sport)
	}
	return sports, nil
}

// Sportart speichern
func SaveSport(sport *models.SportartDefinition) error {
	_, err := db.Exec(`
		INSERT INTO sports (sportart, period_label, periods_count, period_duration, clock_format, clock_direction)
		VALUES (?, ?, ?, ?, ?, ?)
	`, sport.Sportart, sport.PeriodLabel, sport.PeriodsCount, sport.PeriodDuration, sport.ClockFormat, sport.ClockDirection)
	return err
}

// Holt die vorhandenen Spaltennamen einer Tabelle
func getExistingColumns(tableName string) (map[string]struct{}, error) {
	columns := make(map[string]struct{})

	rows, err := db.Query("PRAGMA table_info(" + tableName + ");")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString

		err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		if err != nil {
			return nil, err
		}

		columns[name] = struct{}{}
	}

	return columns, nil
}
