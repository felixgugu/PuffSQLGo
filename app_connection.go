package main

import "puffsql/internal/model"

func (a *App) GetConnections() ([]model.ConnectionProfile, error) {
	return a.manager.Profiles()
}

func (a *App) SaveConnection(req model.SaveConnectionRequest) (model.ConnectionProfile, error) {
	return a.manager.SaveProfile(req)
}

func (a *App) DeleteConnection(id string) error {
	return a.manager.DeleteProfile(id)
}

func (a *App) TestConnection(req model.SaveConnectionRequest) error {
	return a.manager.TestConnection(req)
}

func (a *App) Connect(id string) error {
	return a.manager.Connect(a.context(), id)
}

func (a *App) Disconnect(id string) error {
	return a.manager.Disconnect(id)
}
