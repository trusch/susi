package firebirdconnector

import (
	"../../events"
	"../../state"
	"errors"
	"flag"
	"database/sql"
    _ "github.com/nakagami/firebirdsql"
    "log"
)

var username = flag.String("firebird.username","sysdba","The firebird db username")
var password = flag.String("firebird.password","masterkey","The firebird db password")
var host = flag.String("firebird.host","localhost","The firebird db host")
var path = flag.String("firebird.path","/usr/share/doc/firebird2.5-common-doc/examples/empbuild/employee.fdb","The firebird db path")

type FirebirdConnection struct {
	dbHandle *sql.DB
}

func (ptr *FirebirdConnection) Open(db string) error {
	conn,err := sql.Open("firebirdsql",db)
	if err!=nil {
		return err
	}
	ptr.dbHandle = conn
	return nil
}

func (ptr *FirebirdConnection) Query(query string,args ...interface{}) ([]map[string]interface{},error) {
	result := make([]map[string]interface{},0)
	rows,err := ptr.dbHandle.Query(query,args...)
	if err!=nil {
		return nil,err
	}
	for rows.Next() {
		row := make(map[string]interface{})
		cols, _ := rows.Columns()
		c := len(cols)
		vals := make([]interface{}, c)
		valPtrs := make([]interface{}, c)

		for i := range cols {
			valPtrs[i] = &vals[i]
		}

		rows.Scan(valPtrs...)

		for i := range cols {
			if val, ok := vals[i].([]byte); ok {
				row[cols[i]] = string(val)
			} else {
				row[cols[i]] = vals[i]
			}
		}
		result = append(result,row)
	}
	return result,nil
}


func awnserOk(evt *events.Event,result []map[string]interface{}){
	if evt.ReturnAddr == "" {
		return
	}
	payload := map[string]interface{}{
		"error" : false,
		"result": result,
	}
	res := events.NewEvent(evt.ReturnAddr,payload)
	res.AuthLevel = 0
	events.Publish(res)
}

func awnserError(evt *events.Event,err error){
	if evt.ReturnAddr == "" {
		return
	}
	payload := map[string]interface{}{
		"error" : true,
		"message": err.Error(),
	}
	res := events.NewEvent(evt.ReturnAddr,payload)
	res.AuthLevel = 0
	events.Publish(res)
}

func Go(){
	conn := FirebirdConnection{}

	user := state.Get("firebird.username").(string)
	pw   := state.Get("firebird.password").(string)
	host   := state.Get("firebird.host").(string)
	path   := state.Get("firebird.path").(string)


	connectLine := user + ":"
	connectLine = connectLine + pw + "@"
	connectLine = connectLine + host
	connectLine = connectLine + path
	err := conn.Open(connectLine)
	if err!=nil {
		log.Print(err)
		return
	}

	evtChan,_ := events.Subscribe("firebird::query",0)

	go func(){
		for event := range evtChan {
			if event.AuthLevel > 0 {
				awnserError(event,errors.New("permission denied: need authlevel zero."))
				continue
			}
			if payload,ok := event.Payload.(map[string]interface{}); ok {
				if query,ok := payload["query"].(string); ok {
					if args,ok := payload["args"].([]interface{}); ok {
						res,err := conn.Query(query,args...)
						if err!=nil {
							awnserError(event,err)
							continue
						}else{
							awnserOk(event,res)
							continue
						}
					}else{
						res,err := conn.Query(query)
						if err!=nil {
							awnserError(event,err)
							continue
						}else{
							awnserOk(event,res)
							continue
						}
					}
				}
			}
			awnserError(event,errors.New("query failed: malformed event payload"))
		}
	}()
}