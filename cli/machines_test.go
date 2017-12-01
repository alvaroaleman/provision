package cli

import (
	"io/ioutil"
	"net/http"
	"testing"
)

var machineCreateInputString = `{
  "Address": "192.168.100.110",
  "name": "john",
  "Secret": "secret1",
  "Uuid": "3e7031fe-3062-45f1-835c-92541bc9cbd3",
  "bootenv": "local"
}
`
var machineCreateJohnString = `{
  "Address": "192.168.100.110",
  "Available": true,
  "BootEnv": "local",
  "CurrentTask": 0,
  "Errors": [],
  "Name": "john",
  "Profile": {
    "Available": false,
    "Errors": [],
    "Name": "",
    "ReadOnly": false,
    "Validated": false
  },
  "Profiles": [],
  "ReadOnly": false,
  "Runnable": true,
  "Secret": "secret1",
  "Stage": "none",
  "Tasks": [],
  "Uuid": "3e7031fe-3062-45f1-835c-92541bc9cbd3",
  "Validated": true
}
`

var machineDestroyJohnString = "Deleted machine 3e7031fe-3062-45f1-835c-92541bc9cbd3\n"

func TestMachineCli(t *testing.T) {
	var machineCreateBadJSONString = "{asdgasdg"
	var machineCreateBadJSON2String = "[asdgasdg]"
	var machineUpdateBadJSONString = "asdgasdg"
	var machineUpdateInputString = `{
  "Description": "lpxelinux.0"
}
`
	var machinesParamsNextString = `{
  "jj": 3
}
`
	var machinePluginCreateString = `{
  "Available": true,
  "Errors": [],
  "Name": "incr",
  "PluginErrors": [],
  "Provider": "incrementer",
  "ReadOnly": false,
  "Validated": true
}
`

	var machineRunActionMissingParameterStdinString = "{}"
	var machineRunActionGoodStdinString = `{
	"incrementer/parameter": "parm5",
	"incrementer/step": 10
}
`
	var machineStage1CreateString = `{
	"Name": "stage1",
	"BootEnv": "local",
	"Tasks": [ "jamie", "justine" ]
}
`
	var machineStage2CreateString = `{
  "Name": "stage2",
  "BootEnv": "local",
  "Templates": [
    {
      "Contents": "{{.Param \"sp-param\"}}",
      "Name": "test",
      "Path": "{{.Machine.Path}}/file"
    }
  ]
}
`
	cliTest(false, false, "profiles", "create", "jill").run(t)
	cliTest(false, false, "profiles", "create", "jean").run(t)
	cliTest(false, false, "profiles", "create", "stage-prof").run(t)
	cliTest(false, false, "tasks", "create", "jamie").run(t)
	cliTest(false, false, "tasks", "create", "justine").run(t)
	cliTest(false, false, "stages", "create", machineStage1CreateString).run(t)
	cliTest(false, false, "stages", "create", machineStage2CreateString).run(t)
	cliTest(false, false, "plugins", "create", machinePluginCreateString).run(t)
	cliTest(true, false, "machines").run(t)
	cliTest(false, false, "machines", "list").run(t)
	cliTest(true, true, "machines", "create").run(t)
	cliTest(true, true, "machines", "create", "john", "john2").run(t)
	cliTest(false, true, "machines", "create", machineCreateBadJSONString).run(t)
	cliTest(false, true, "machines", "create", machineCreateBadJSON2String).run(t)
	cliTest(false, false, "machines", "create", machineCreateInputString).run(t)
	cliTest(false, true, "machines", "create", machineCreateInputString).run(t)
	cliTest(false, false, "machines", "list").run(t)
	cliTest(false, false, "machines", "list", "Name=fred").run(t)
	cliTest(false, false, "machines", "list", "Name=john").run(t)
	cliTest(false, false, "machines", "list", "BootEnv=local").run(t)
	cliTest(false, false, "machines", "list", "BootEnv=false").run(t)
	cliTest(false, false, "machines", "list", "Address=192.168.100.110").run(t)
	cliTest(false, false, "machines", "list", "Address=1.1.1.1").run(t)
	cliTest(false, true, "machines", "list", "Address=fred").run(t)
	cliTest(false, false, "machines", "list", "Uuid=4e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, false, "machines", "list", "Uuid=3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, true, "machines", "list", "Uuid=false").run(t)
	cliTest(false, false, "machines", "list", "Runnable=true").run(t)
	cliTest(false, false, "machines", "list", "Runnable=false").run(t)
	cliTest(false, true, "machines", "list", "Runnable=fred").run(t)
	cliTest(true, true, "machines", "show").run(t)
	cliTest(true, true, "machines", "show", "john", "john2").run(t)
	cliTest(false, true, "machines", "show", "john").run(t)
	cliTest(false, false, "machines", "show", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, false, "machines", "show", "Key:3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, false, "machines", "show", "Uuid:3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, false, "machines", "show", "Name:john").run(t)
	cliTest(true, true, "machines", "exists").run(t)
	cliTest(true, true, "machines", "exists", "john", "john2").run(t)
	cliTest(false, false, "machines", "exists", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, true, "machines", "exists", "john").run(t)
	cliTest(true, true, "machines", "exists", "john", "john2").run(t)
	cliTest(true, true, "machines", "update").run(t)
	cliTest(true, true, "machines", "update", "john", "john2", "john3").run(t)
	cliTest(false, true, "machines", "update", "3e7031fe-3062-45f1-835c-92541bc9cbd3", machineUpdateBadJSONString).run(t)
	cliTest(false, false, "machines", "update", "3e7031fe-3062-45f1-835c-92541bc9cbd3", machineUpdateInputString).run(t)
	cliTest(false, true, "machines", "update", "john2", machineUpdateInputString).run(t)
	cliTest(false, false, "machines", "show", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, false, "machines", "show", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(true, true, "machines", "destroy").run(t)
	cliTest(true, true, "machines", "destroy", "john", "june").run(t)
	cliTest(false, false, "machines", "destroy", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, true, "machines", "destroy", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, false, "machines", "list").run(t)
	cliTest(false, false, "machines", "create", "-").Stdin(machineCreateInputString + "\n").run(t)
	cliTest(false, false, "machines", "list").run(t)
	cliTest(false, false, "machines", "update", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "-").Stdin(machineUpdateInputString + "\n").run(t)
	cliTest(false, false, "machines", "show", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	// bootenv tests
	cliTest(true, true, "machines", "bootenv").run(t)
	cliTest(false, true, "machines", "bootenv", "john", "john2").run(t)
	cliTest(false, false, "machines", "bootenv", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "john2").run(t)
	cliTest(false, false, "machines", "bootenv", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "local").run(t)
	cliTest(false, false, "machines", "update", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "{ \"Runnable\": true }").run(t)
	// stage tests
	cliTest(true, true, "machines", "stage").run(t)
	cliTest(false, true, "machines", "stage", "john", "john2").run(t)
	cliTest(false, true, "machines", "stage", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "john2").run(t)
	cliTest(false, false, "machines", "stage", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "stage1").run(t)
	cliTest(false, false, "machines", "update", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "{ \"Runnable\": true }").run(t)
	cliTest(false, true, "machines", "stage", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "stage2").run(t)
	cliTest(false, false, "machines", "stage", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "stage2", "--force").run(t)
	cliTest(false, false, "machines", "stage", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "", "--force").run(t)
	// Add/Remove Profile tests
	cliTest(true, true, "machines", "addprofile").run(t)
	cliTest(false, false, "machines", "addprofile", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "jill").run(t)
	cliTest(false, false, "machines", "addprofile", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "jean").run(t)
	cliTest(false, true, "machines", "addprofile", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "jill").run(t)
	cliTest(false, false, "profiles", "set", "jill", "param", "jill-param", "to", "janga").run(t)
	cliTest(false, false, "profiles", "set", "stage-prof", "param", "sp-param", "to", "val").run(t)
	cliTest(false, false, "machines", "stage", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "stage2", "--force").run(t)
	cliTest(false, false, "stages", "addprofile", "stage2", "stage-prof").run(t)

	cliTest(false, false, "machines", "params", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, false, "machines", "params", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "--aggregate").run(t)
	tr := &http.Transport{}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("GET", "http://127.0.0.1:10002/machines/3e7031fe-3062-45f1-835c-92541bc9cbd3/file", nil)
	req.SetBasicAuth("rocketskates", "r0cketsk8ts")
	rsp, apierr := client.Do(req)
	if apierr != nil {
		t.Errorf("FAIL: Failed to query machine file: %s", apierr)
	} else {
		defer rsp.Body.Close()
		body, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			t.Errorf("FAIL: Failed to read all: %s", err)
		}
		if string(body) != "val" {
			t.Errorf("FAIL: Body was: AA%sAA expected %s", string(body), "val")
		}
	}

	cliTest(false, false, "machines", "stage", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "", "--force").run(t)
	cliTest(true, true, "machines", "removeprofile").run(t)
	cliTest(false, false, "machines", "removeprofile", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "justine").run(t)
	cliTest(false, false, "machines", "removeprofile", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "jill").run(t)
	cliTest(false, false, "machines", "removeprofile", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "jean").run(t)
	cliTest(true, true, "machines", "get").run(t)
	cliTest(false, true, "machines", "get", "john", "param", "john2").run(t)
	cliTest(false, false, "machines", "get", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "john2").run(t)
	cliTest(true, true, "machines", "set").run(t)
	cliTest(false, true, "machines", "set", "john", "param", "john2", "to", "cow").run(t)
	cliTest(false, false, "machines", "set", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "john2", "to", "cow").run(t)
	cliTest(false, false, "machines", "get", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "john2").run(t)
	cliTest(false, false, "machines", "set", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "john2", "to", "3").run(t)
	cliTest(false, false, "machines", "set", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "john3", "to", "4").run(t)
	cliTest(false, false, "machines", "get", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "john2").run(t)
	cliTest(false, false, "machines", "get", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "john3").run(t)
	cliTest(false, false, "machines", "set", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "john2", "to", "null").run(t)
	cliTest(false, false, "machines", "get", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "john2").run(t)
	cliTest(false, false, "machines", "get", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "john3").run(t)
	cliTest(true, true, "machines", "actions").run(t)
	cliTest(false, true, "machines", "actions", "john").run(t)
	cliTest(false, false, "machines", "actions", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(true, true, "machines", "action").run(t)
	cliTest(true, true, "machines", "action", "john").run(t)
	cliTest(false, true, "machines", "action", "john", "command").run(t)
	cliTest(false, true, "machines", "action", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "command").run(t)
	cliTest(false, false, "machines", "action", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "increment").run(t)
	cliTest(true, true, "machines", "runaction").run(t)
	cliTest(true, true, "machines", "runaction", "fred").run(t)
	cliTest(false, true, "machines", "runaction", "fred", "command").run(t)
	cliTest(false, true, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "command").run(t)
	cliTest(false, false, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "increment").run(t)
	cliTest(false, true, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "increment", "fred").run(t)

	cliTest(false, false, "machines", "actions", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, false, "machines", "action", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "reset_count").run(t)
	cliTest(false, false, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "reset_count").run(t)
	cliTest(false, false, "machines", "actions", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, true, "machines", "action", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "reset_count").run(t)
	cliTest(false, true, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "reset_count").run(t)
	cliTest(false, false, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "increment", "incrementer/parameter", "asgdasdg").run(t)
	cliTest(false, false, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "increment", "incrementer/parameter", "parm1", "extra", "10").run(t)
	cliTest(false, false, "machines", "get", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "parm1").run(t)
	cliTest(false, true, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "increment", "incrementer/parameter", "parm2", "incrementer/step", "asgdasdg").run(t)
	cliTest(false, false, "machines", "get", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "parm2").run(t)
	cliTest(false, false, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "increment", "incrementer/parameter", "parm2", "incrementer/step", "10").run(t)
	cliTest(false, false, "machines", "get", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "parm2").run(t)
	cliTest(false, true, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "increment", "-").Stdin("fred").run(t)
	cliTest(false, false, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "reset_count", "-").Stdin(machineRunActionMissingParameterStdinString).run(t)
	cliTest(false, true, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "reset_count", "-").Stdin(machineRunActionMissingParameterStdinString).run(t)
	cliTest(false, false, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "increment", "-").Stdin(machineRunActionMissingParameterStdinString).run(t)
	cliTest(false, false, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "increment", "-").Stdin(machineRunActionGoodStdinString).run(t)
	cliTest(false, false, "machines", "runaction", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "increment", "-").Stdin(machineRunActionGoodStdinString).run(t)
	cliTest(false, false, "machines", "get", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "param", "parm5").run(t)
	cliTest(true, true, "machines", "wait").run(t)
	cliTest(true, true, "machines", "wait", "jk").run(t)
	cliTest(true, true, "machines", "wait", "jk", "jk").run(t)
	cliTest(true, true, "machines", "wait", "jk", "jk", "jk", "jk", "jk").run(t)
	cliTest(false, true, "machines", "wait", "jk", "jk", "jk", "jk").run(t)
	cliTest(false, true, "machines", "wait", "jk", "jk", "jk").run(t)
	cliTest(false, false, "machines", "wait", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "jk", "jk", "1").run(t)
	cliTest(false, false, "machines", "wait", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "BootEnv", "local", "1").run(t)
	cliTest(false, false, "machines", "wait", "3e7031fe-3062-45f1-835c-92541bc9cbd3", "Runnable", "fred", "1").run(t)
	cliTest(true, true, "machines", "params").run(t)
	cliTest(false, true, "machines", "params", "john2").run(t)
	cliTest(false, false, "machines", "params", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, true, "machines", "params", "john2", machinesParamsNextString).run(t)
	cliTest(false, false, "machines", "params", "3e7031fe-3062-45f1-835c-92541bc9cbd3", machinesParamsNextString).run(t)
	cliTest(false, false, "machines", "params", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, false, "machines", "show", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, false, "machines", "destroy", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, false, "machines", "list").run(t)
	cliTest(false, false, "prefs", "set", "defaultStage", "stage1").run(t)
	cliTest(false, false, "machines", "create", machineCreateInputString).run(t)
	cliTest(false, false, "machines", "destroy", "3e7031fe-3062-45f1-835c-92541bc9cbd3").run(t)
	cliTest(false, false, "machines", "list").run(t)
	cliTest(false, false, "prefs", "set", "defaultStage", "none").run(t)
	cliTest(false, false, "plugins", "destroy", "incr").run(t)
	cliTest(false, false, "stages", "destroy", "stage1").run(t)
	cliTest(false, false, "stages", "destroy", "stage2").run(t)
	cliTest(false, false, "profiles", "destroy", "jill").run(t)
	cliTest(false, false, "profiles", "destroy", "jean").run(t)
	cliTest(false, false, "profiles", "destroy", "stage-prof").run(t)
	cliTest(false, false, "tasks", "destroy", "jamie").run(t)
	cliTest(false, false, "tasks", "destroy", "justine").run(t)
}
