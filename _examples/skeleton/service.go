package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/nissy/bon"
	"github.com/nissy/bon/_examples/skeleton/controller"
	"github.com/nissy/bon/middleware"
	"gopkg.in/BurntSushi/toml.v0"
)

type service struct {
	Server *Server         `toml:"server"`
	Name   controller.Name `toml:"name"`
}

type Server struct {
	Listen  string `toml:"listen"`
	PIDFile string `toml:"pidfile"`
}

func newService() *service {
	return &service{
		Server: &Server{
			Listen: ":0",
		},
	}
}

func (s *service) readConfig(filename string) error {
	if _, err := toml.DecodeFile(filename, &s); err != nil {
		return err
	}

	if err := s.validate(); err != nil {
		return err
	}

	return nil
}

func (s *service) validate() error {
	return nil
}

func (s *service) run() error {
	r := bon.NewRouter()

	if err := s.Name.Init(); err != nil {
		return err
	}

	r.Use(middleware.Timeout(20 * time.Second))
	r.Get("/", s.Name.Hallo)

	if len(s.Server.PIDFile) > 0 {
		if err := ioutil.WriteFile(s.Server.PIDFile, []byte(strconv.Itoa(os.Getpid())), os.ModePerm); err != nil {
			return err
		}
	}

	return gracehttp.Serve(&http.Server{
		Addr:    s.Server.Listen,
		Handler: r,
	})
}
