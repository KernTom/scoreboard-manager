//go:build windows

package ui

import (
	"github.com/KernTom/scoreboard-manager/internal/models"

	"github.com/lxn/walk"
)

type TeamListModel struct {
	walk.ListModelBase
	Teams []*models.Team
}

func (m *TeamListModel) ItemCount() int {
	return len(m.Teams)
}

func (m *TeamListModel) Value(index int) interface{} {
	return m.Teams[index].Name
}

func (m *TeamListModel) GetTeam(index int) *models.Team {
	if index >= 0 && index < len(m.Teams) {
		return m.Teams[index]
	}
	return nil
}
