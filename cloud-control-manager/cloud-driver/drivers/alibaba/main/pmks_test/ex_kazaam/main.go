package main

import (
	"encoding/json"
	"fmt"

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/qntfy/kazaam/v4"
)

func ex1() {
	k, _ := kazaam.NewKazaam(`[{"operation": "shift", "spec": {"output": "input"}}]`)
	kazaamOut, _ := k.TransformJSONStringToString(`{"input":"input value"}`)

	fmt.Println(kazaamOut)
}

func ex2() {
	spec := `[
		{
			"operation": "shift",
			"spec": {
			"object.id": "doc.uid",
			"gid2": "doc.guid[1]",
			"allGuids": "doc.guidObjects[*].id"
			}
		}
	]`
	k, e := kazaam.NewKazaam(spec)
	if e != nil {
		fmt.Println(e)
	}

	text := `{
		"doc": {
			"uid": 12345,
			"guid": ["guid0", "guid2", "guid4"],
			"guidObjects": [{"id": "guid0"}, {"id": "guid2"}, {"id": "guid4"}]
		},
		"top-level-key": null
	}`
	kazaamOut, _ := k.TransformJSONStringToString(text)

	fmt.Println(kazaamOut)

}

func ex3() {

	spec := `[
		{
			"operation": "shift",
			"spec": {
				"IId.NameId": "doc.uid",	
				"Objects": "doc.guidObjects[*]",
				"state": "x"
			}
		},
		{
			"operation": "default",
			"spec": {
			  "type": "message"
			}			
		}
	]`

	k, e := kazaam.NewKazaam(spec)
	if e != nil {
		fmt.Println(e)
	}

	text := `{
		"doc": {
			"uid": 12345,
			"guid": ["guid0", "guid2", "guid4"],
			"guidObjects": [{"id": "guid0"}, {"id": "guid2"}, {"id": "guid4"}]
		},
		"top-level-key": null
	}`

	out, _ := k.TransformJSONStringToString(text)
	println(out)

	data := &irs.NodeGroupInfo{}
	_ = json.Unmarshal([]byte(out), &data)
	print(data.IId.NameId)
}

func main() {
	// ex1()
	//ex2()
	ex3()
}
