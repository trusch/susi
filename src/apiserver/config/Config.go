package config

import (
	"../state"
	"os"
	"log"
	"encoding/json"
	"flag"
	"path/filepath"
	"strings"
	"time"
)

var configPath = flag.String("configPath",".","path to your configfiles")

type ConfigManager struct {
	modifiedTimes map[string]time.Time
}

func (ptr *ConfigManager) LoadFileToState(filename string) error {
	f,err := os.Open(filename)
	if err!=nil {
		return err
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	data := make(map[string]interface{})
	err = decoder.Decode(&data)
	if err!=nil {
		log.Print("malformed config file: ",filename," (",err,")")
		return err
	}
	for key,val := range data {
		/*log.Print("load config: ",key," : ",val)*/
		state.Set(key,val)
	}
	return nil
}

func (ptr *ConfigManager) LoadFiles(){
	filepath.Walk(*configPath,func(path string, info os.FileInfo, err error) error {
		name := info.Name()
		if !info.IsDir() && (strings.HasSuffix(name,"conf") || strings.HasSuffix(name,"cfg")){
			oldTime := ptr.modifiedTimes[name]
			newTime := info.ModTime()
			if(!newTime.Equal(oldTime)){
				ptr.LoadFileToState(path)
				ptr.modifiedTimes[name] = newTime
			}
		}
		return nil
	})
}

func (ptr *ConfigManager) LoadDefaultFlags(){
	flag.VisitAll(func(flag *flag.Flag){
		state.Set(flag.Name,flag.Value.String())	
	})
}

func (ptr *ConfigManager) LoadFlags(){
	flag.VisitAll(func(flag *flag.Flag){
		if flag.DefValue != flag.Value.String() {
			state.Set(flag.Name,flag.Value.String())
		}
	})
}

func NewManager() *ConfigManager{
	ptr := new(ConfigManager)
	ptr.modifiedTimes = make(map[string]time.Time)
	ch := make(chan bool)
	go func(){
		flag.Parse()
		ptr.LoadDefaultFlags()
		ptr.LoadFiles()
		ptr.LoadFlags()
		ch <- true
		time.Sleep(5*time.Second)
		for{
			ptr.LoadFiles()
			ptr.LoadFlags()

			time.Sleep(5*time.Second)
		}
	}()
	<-ch
	return ptr
}
