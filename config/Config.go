/*
 * Copyright (c) 2014, webvariants GmbH, http://www.webvariants.de
 *
 * This file is released under the terms of the MIT license. You can find the
 * complete text in the attached LICENSE file or online at:
 *
 * http://www.opensource.org/licenses/mit-license.php
 *
 * @author: Tino Rusch (tino.rusch@webvariants.de)
 */

package config

import (
	"encoding/json"
	"flag"
	"github.com/trusch/susi/state"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var configPath = flag.String("configPath", ".", "path to your configfiles")

type ConfigManager struct {
	modifiedTimes map[string]time.Time
}

func (ptr *ConfigManager) LoadFileToState(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	filename = filename[len(*configPath):]
	basekey := strings.Replace(filename, "/", ".", -1)
	lastDot := strings.LastIndex(basekey, ".")
	firstDot := strings.Index(basekey, ".")
	basekey = basekey[firstDot+1 : lastDot]
	decoder := json.NewDecoder(f)
	data := make(map[string]interface{})
	err = decoder.Decode(&data)
	if err != nil {
		log.Print("malformed config file: ", filename, " (", err, ")")
		return err
	}
	for key, val := range data {
		//log.Print("load config: ", basekey+"."+key, " : ", val)
		state.Set(basekey+"."+key, val)
	}
	return nil
}

func (ptr *ConfigManager) LoadFiles() {
	filepath.Walk(*configPath, func(path string, info os.FileInfo, err error) error {
		name := info.Name()
		//log.Print("scan file ", name)
		if !info.IsDir() && (strings.HasSuffix(name, "conf") || strings.HasSuffix(name, "cfg")) {
			oldTime := ptr.modifiedTimes[name]
			newTime := info.ModTime()
			if !newTime.Equal(oldTime) {
				ptr.LoadFileToState(path)
				ptr.modifiedTimes[name] = newTime
			}
		}
		return nil
	})
}

func (ptr *ConfigManager) LoadDefaultFlags() {
	flag.VisitAll(func(flag *flag.Flag) {
		state.Set(flag.Name, flag.Value.String())
	})
}

func (ptr *ConfigManager) LoadFlags() {
	flag.VisitAll(func(flag *flag.Flag) {
		if flag.DefValue != flag.Value.String() {
			state.Set(flag.Name, flag.Value.String())
		}
	})
}

func Go() {
	flag.Parse()
	ptr := new(ConfigManager)
	ptr.modifiedTimes = make(map[string]time.Time)
	ch := make(chan bool)

	go func() {
		flag.Parse()
		ptr.LoadDefaultFlags()
		ptr.LoadFiles()
		ptr.LoadFlags()
		ch <- true
		time.Sleep(5 * time.Second)
		for {
			ptr.LoadFiles()
			time.Sleep(5 * time.Second)
		}
	}()
	<-ch
	log.Print("Successfully loaded config files from ", *configPath)
}
