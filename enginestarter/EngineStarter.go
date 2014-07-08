package enginestarter

import (
	"bufio"
	"flag"
	"github.com/trusch/susi/state"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

var root = flag.String("enginestarter.root", "/usr/share/susi/controller", "where to search for engines")

func Go() {
	dir := state.Get("enginestarter.root").(string)
	if d, err := os.Open(dir); err == nil {
		d.Close()
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() && (info.Mode().Perm()&0100) > 0 /*is executable?*/ {
				cmd := exec.Command(path)
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					log.Print(err)
					return err
				}
				stderr, err := cmd.StderrPipe()
				if err != nil {
					log.Print(err)
					return err
				}
				err = cmd.Start()
				if err != nil {
					log.Print(err)
				} else {
					go func() {
						reader := bufio.NewReader(stdout)
						for {
							line, _, err := reader.ReadLine()
							if err != nil {
								log.Print(err)
								return
							}
							log.Print(string(line))
						}
					}()
					go func() {
						reader := bufio.NewReader(stderr)
						for {
							line, _, err := reader.ReadLine()
							if err != nil {
								log.Print(err)
								return
							}
							log.Print(string(line))
						}
					}()
					log.Print("started ", path)
				}
			}
			return nil
		})
	} else {
		log.Print(err)
	}
}
