package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/nissy/bon"
	"github.com/nissy/bon/_examples/api/controller"
	"github.com/nissy/bon/middleware"
)

type (
	service struct {
		Server *Server
		API    *controller.API
	}

	Server struct {
		Listen  string
		PIDFile string
	}
)

func newService() *service {
	return &service{
		Server: &Server{
			Listen: ":0",
		},
	}
}

func (sv *service) applyConfig(filename string) error {
	if _, err := toml.DecodeFile(filename, &sv); err != nil {
		return err
	}

	if err := sv.API.Build(); err != nil {
		return err
	}

	return nil
}

func (sv *service) route() http.Handler {
	r := bon.NewRouter()

	r.Use(middleware.Timeout(20 * time.Second))
	r.Get("/users/:name", sv.API.GetUser)

	return r
}

func (sv *service) serve() error {
	if len(sv.Server.PIDFile) > 0 {
		if err := ioutil.WriteFile(sv.Server.PIDFile, []byte(strconv.Itoa(os.Getpid())), os.ModePerm); err != nil {
			return err
		}
	}

	return gracehttp.Serve(&http.Server{
		Addr:    sv.Server.Listen,
		Handler: sv.route(),
	})
}
