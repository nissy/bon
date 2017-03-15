package controller

import (
	"errors"
	"net/http"

	"github.com/nissy/bon/multiple"
)

type API struct {
	Databases multiple.Databases
}

func (api *API) Build() error {
	if len(api.Databases) == 0 {
		return errors.New("not database")
	}

	if err := api.Databases.Set(); err != nil {
		return err
	}

	return nil
}

func (api *API) GetUser(w http.ResponseWriter, r *http.Request) {
	//db := api.Databases.Get("read")

}
