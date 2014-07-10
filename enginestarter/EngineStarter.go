package enginestarter

import (
	"bufio"
	"flag"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/state"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

var root = flag.String("enginestarter.root", "/usr/share/susi/controller", "where to search for engines")

type Engine struct {
	cmd    *exec.Cmd
	name   string
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func NewEngine(path string) (*Engine, error) {
	engine := new(Engine)
	engine.name = path
	engine.cmd = exec.Command(path)
	stdout, err := engine.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	engine.stdout = stdout
	stderr, err := engine.cmd.StderrPipe()
	if err != nil {
		log.Print(err)
		return nil, err
	}
	engine.stderr = stderr
	err = engine.cmd.Start()
	if err != nil {
		return nil, err
	} else {
		go func() {
			reader := bufio.NewReader(engine.stdout)
			for {
				line, _, err := reader.ReadLine()
				if err != nil {
					//log.Print(err)
					return
				}
				log.Print(string(line))
			}
		}()
		go func() {
			reader := bufio.NewReader(engine.stderr)
			for {
				line, _, err := reader.ReadLine()
				if err != nil {
					//log.Print(err)
					return
				}
				log.Print(string(line))
			}
		}()
		ch, _ := events.Subscribe("global::shutdown", 0)
		go func() {
			event := <-ch
			if event.AuthLevel > 0 {
				log.Print("wrong authlevel for global::shutdown")
				return
			}
			log.Printf("stopping engine %s...", engine.name)
			if err := engine.cmd.Process.Kill(); err != nil {
				log.Printf("Failed killing engine %s...", engine.name)
			}
		}()
		log.Print("started ", path)
	}

	return engine, nil
}

type EngineStarter struct {
	commands []*Engine
}

func (ptr *EngineStarter) backend() {
	dir := state.Get("enginestarter.root").(string)
	if d, err := os.Open(dir); err == nil {
		d.Close()
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() && (info.Mode().Perm()&0100) > 0 /*is executable?*/ {
				if engine, err := NewEngine(path); err != nil {
					log.Print(err)
				} else {
					ptr.commands = append(ptr.commands, engine)
				}
			}
			return nil
		})
	} else {
		log.Print(err)
	}
}

func Go() {
	starter := EngineStarter{}
	go starter.backend()
}
