package main

import "puffsql/internal/model"

func (a *App) GetConnections() ([]model.ConnectionProfile, error) {
	profiles, err := a.manager.Profiles()
	a.logError("connection.list", err)
	return profiles, err
}

func (a *App) SaveConnection(req model.SaveConnectionRequest) (model.ConnectionProfile, error) {
	profile, err := a.manager.SaveProfile(req)
	a.logError("connection.save", err)
	return profile, err
}

func (a *App) DeleteConnection(id string) error {
	err := a.manager.DeleteProfile(id)
	a.logError("connection.delete", err)
	return err
}

func (a *App) TestConnection(req model.SaveConnectionRequest) error {
	defer a.recoverPanic("connection.test")
	err := a.manager.TestConnection(req)
	a.logError("connection.test", err)
	return err
}

func (a *App) Connect(id string) error {
	defer a.recoverPanic("connection.connect")
	err := a.manager.Connect(a.context(), id)
	a.logError("connection.connect", err)
	return err
}

func (a *App) Disconnect(id string) error {
	err := a.manager.Disconnect(id)
	a.logError("connection.disconnect", err)
	return err
}
