package authentification

import (
	"github.com/trusch/susi/config"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/state"
	"os"
	"testing"
)

func init() {
	state.Go()
	events.Go()
	config.Go()
	state.Set("authentification.usersFile", "/tmp/users.json")
	Go()
}

func assert(t *testing.T, assertion bool, message string, a ...interface{}) {
	if !assertion {
		t.Errorf(message, a...)
	}
}

func TestAddUser(t *testing.T) {
	awnserChan, closeChan := events.Subscribe("awnserChan", 0)
	userManagerRef.Load()
	defer func() {
		closeChan <- true
		os.Remove("/tmp/users.json")
	}()
	/**
	 * Testing normal adding
	 */
	request := events.NewEvent("authentification::adduser", map[string]interface{}{
		"username":  "test1",
		"password":  "test1",
		"authlevel": 2,
	})
	request.AuthLevel = 0
	request.ReturnAddr = "awnserChan"
	events.Publish(request)
	awnser := <-awnserChan
	payload, ok := awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == true, "creating testuser failed: %v", payload)

	/**
	 * Testing of allready taken username
	 */
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == false, "creating testuser should have been failed: %v", payload)

	/**
	 * Testing if authlevel 0 is needed
	 */
	request.AuthLevel = 1
	request.Payload.(map[string]interface{})["username"] = "foo"
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == false, "creating testuser should have been failed: %v", payload)

	/**
	 * Test adding a second user
	 */
	request.Payload = map[string]interface{}{
		"username":  "test2",
		"password":  "test2",
		"authlevel": 2,
	}
	request.AuthLevel = 0
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == true, "adding of a second testuser should have been succeded: %v", payload)
	assert(t, userManagerRef.users[0].ID == 1 && userManagerRef.users[1].ID == 2,
		"IDs are wrong: %v %v", userManagerRef.users[0].ID, userManagerRef.users[1].ID)

	request.AuthLevel = 0

	/**
	 * Testing malformed request 1
	 */
	request.Payload = map[string]interface{}{
		"username": "test2",
		"password": "test2",
	}
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == false, "creating testuser should have been failed: %v", payload)

	/**
	 * Testing malformed request 2
	 */
	request.Payload = map[string]interface{}{
		"username":  "test2",
		"password":  123,
		"authlevel": 2,
	}
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == false, "creating testuser should have been failed: %v", payload)

	/**
	 * Testing malformed request 3
	 */
	request.Payload = ""
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == false, "creating testuser should have been failed: %v", payload)

}

func TestDelUser(t *testing.T) {
	awnserChan, closeChan := events.Subscribe("awnserChan", 0)
	userManagerRef.Load()
	defer func() {
		closeChan <- true
		os.Remove("/tmp/users.json")
	}()
	/**
	 * add two users
	 */
	request := events.NewEvent("authentification::adduser", map[string]interface{}{
		"username":  "test1",
		"password":  "test1",
		"authlevel": 2,
	})
	request.AuthLevel = 0
	request.ReturnAddr = "awnserChan"
	events.Publish(request)
	awnser := <-awnserChan
	request.Payload = map[string]interface{}{
		"username":  "test2",
		"password":  "test2",
		"authlevel": 2,
	}
	events.Publish(request)
	awnser = <-awnserChan

	/**
	 * Delete first user
	 */
	request = events.NewEvent("authentification::deluser", map[string]interface{}{
		"username": "test1",
	})
	request.AuthLevel = 0
	request.ReturnAddr = "awnserChan"
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok := awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == true, "deleting of a first testuser should have been succeded: %v", payload)
	assert(t, len(userManagerRef.users) == 1, "There should be only one user left")
	assert(t, userManagerRef.users[0].ID == 2, "The ID should be 2")

	/**
	 * Delete non existent user
	 */
	request = events.NewEvent("authentification::deluser", map[string]interface{}{
		"username": "testFAIL",
	})
	request.AuthLevel = 0
	request.ReturnAddr = "awnserChan"
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == false, "deleting of nonexistent testuser should have been failed: %v", payload)

	/**
	 * Testing if authlevel 0 is needed
	 */
	request.AuthLevel = 1
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == false, "deleting testuser should have been failed: %v", payload)

	request.AuthLevel = 0

	/**
	 * Testing malformed request 1
	 */
	request.Payload = map[string]interface{}{}
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == false, "deleting testuser should have been failed: %v", payload)

	/**
	 * Testing malformed request 2
	 */
	request.Payload = map[string]interface{}{
		"usernameFAIL": "test2",
	}
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == false, "deleting testuser should have been failed: %v", payload)

	/**
	 * Testing malformed request 3
	 */
	request.Payload = ""
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == false, "creating testuser should have been failed: %v", payload)
}

func TestCheckUser(t *testing.T) {
	awnserChan, closeChan := events.Subscribe("awnserChan", 0)
	userManagerRef.Load()
	defer func() {
		closeChan <- true
		os.Remove("/tmp/users.json")
	}()
	/**
	 * add user
	 */
	request := events.NewEvent("authentification::adduser", map[string]interface{}{
		"username":  "test1",
		"password":  "test1",
		"authlevel": 2,
	})
	request.AuthLevel = 0
	request.ReturnAddr = "awnserChan"
	events.Publish(request)
	awnser := <-awnserChan

	/**
	 * Check first user for success
	 */
	request = events.NewEvent("authentification::checkuser", map[string]interface{}{
		"username": "test1",
		"password": "test1",
	})
	request.AuthLevel = 0
	request.ReturnAddr = "awnserChan"
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok := awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	user, ok := payload.Message.(*User)
	assert(t, ok, "checkuser doesnt return a user: %v", payload)
	assert(t, user.Username == "test1", "username should be test1: %v", user.Username)
	assert(t, user.Password == "", "password should be empty in reply: %v", user.Username)

	/**
	 * Check first user for error
	 */
	request = events.NewEvent("authentification::checkuser", map[string]interface{}{
		"username": "test1",
		"password": "test1FAIL",
	})
	request.AuthLevel = 0
	request.ReturnAddr = "awnserChan"
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Message == nil, "user should be nil: %v", payload)

	/**
	 * Testing if authlevel 0 is needed
	 */
	request.AuthLevel = 1
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == false, "creating testuser should have been failed: %v", payload)

	request.AuthLevel = 0

	/**
	 * Testing malformed request 1
	 */
	request.Payload = ""
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == false, "checking testuser should have been failed: %v", payload)

	/**
	 * Testing malformed request 2
	 */
	request.Payload = map[string]interface{}{
		"username": "test2",
		"password": 123,
	}
	events.Publish(request)
	awnser = <-awnserChan
	payload, ok = awnser.Payload.(*AwnserData)
	assert(t, ok, "payload is not a AwnserData struct: %T", awnser.Payload)
	assert(t, payload.Success == false, "checking testuser should have been failed: %v", payload)

}

func TestLoadSave(t *testing.T) {
	/**
	 * Test loading of non existent file
	 */
	os.Remove("/tmp/users.json")
	userManagerRef.Load()
	assert(t, len(userManagerRef.users) == 0, "user list should be empty: %v", userManagerRef.users)
	userManagerRef.AddUser("test", "test", 0)
	/**
	 * Test loading after writing
	 */
	userManagerRef.Load()
	assert(t, len(userManagerRef.users) == 1, "user list should contain one entry: %v", userManagerRef.users)
	os.Remove("/tmp/users.json")
	/**
	 * Test failing to write
	 */
	userManagerRef.usersFile = "/i/dont/exist"
	userManagerRef.Save()
	userManagerRef.Load()
	assert(t, len(userManagerRef.users) == 0, "user list should zero entries: %v", userManagerRef.users)

	/**
	 * Test loading a broke file
	 */
	f, _ := os.Create("/tmp/test")
	defer func() {
		os.Remove("/tmp/test")
	}()
	f.Write([]byte(`{"Username":"foo","Password":123}`))
	f.Close()
	userManagerRef.usersFile = "/tmp/test"
	userManagerRef.Load()
	assert(t, len(userManagerRef.users) == 0, "user list should zero entries: %v", userManagerRef.users)
}
