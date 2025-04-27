package models

import (
	"time"
)

// TemplateSettings speichert die globalen Anzeigeoptionen
type TemplateSettings struct {
	ID                  int
	Width               int
	Height              int
	X                   int
	Y                   int
	Sportart            string
	PeriodLabel         string
	PeriodsCount        int
	PeriodDuration      int
	GameclockMode       string
	ShowPeriod          bool
	ShowGameclock       bool
	ShowClock           bool
	ClockFontFamily     string
	ClockFontSize       int
	ClockFontColor      string
	PeriodFontFamily    string
	PeriodFontSize      int
	PeriodFontColor     string
	ScoreFontFamily     string
	ScoreFontSize       int
	ScoreFontColor      string
	SeparatorFontFamily string
	SeparatorFontSize   int
	SeparatorFontColor  string
	ExtraTimeFontColor  string
	Name                string
	BackgroundFontColor string
}

// Team speichert Infos zu einem Team
type Team struct {
	ID       int
	Name     string
	Sportart string
	LogoData []byte // Pfad zum Logo
}

// SportartDefinition speichert Perioden- und Zeitregeln je Sportart
type SportartDefinition struct {
	ID             int
	Sportart       string
	PeriodLabel    string
	PeriodsCount   int
	PeriodDuration int
	ClockFormat    string // "MM:SS" oder "Minuten"
	ClockDirection string // "Up" oder "Down"
}

type Match struct {
	ID               int
	Team1            *Team
	Team2            *Team
	TemplateSettings *TemplateSettings
	GameTime         time.Time
	Sportart         string
}
