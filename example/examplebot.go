package main

//import "github.com/kr/pretty"
import (
	"strings"
	"os/exec"
	"io/ioutil"
	"os"
	"time"
	"log"
	"github.com/donomii/gomatrix"
	"fmt"
)


var bbs *BBSdata

// Login to a local homeserver and set the user ID and access token on success.
func Login(server, username, password string) *gomatrix.Client {
	cli, _ := gomatrix.NewClient(server, "", "")
	resp, err := cli.Login(&gomatrix.ReqLogin{
		Type:     "m.login.password",
		User:     username,
		Password: password,
	})
	if err != nil {
		panic(err)
	}
	cli.SetCredentials(resp.UserID, resp.AccessToken)
	return cli
}

func ProcessResponse(resSync *gomatrix.RespSync, talkative bool) []*gomatrix.Event {
	ret := []*gomatrix.Event{}
	for _,v := range resSync.Rooms.Join {
		//fmt.Printf("%v: %# v\n\n\n", k, pretty.Formatter(v.Timeline.Events))
		for _, e := range v.Timeline.Events {
			ret = append(ret, &e)
		}

	}
	return ret
}


func ExtractRooms(resSync *gomatrix.RespSync) map[string]string{
	rooms := map[string]string{}
	for roomID,v := range resSync.Rooms.Join {
		//fmt.Printf("%v: %# v\n\n\n", k, pretty.Formatter(v.Timeline.Events))
		for _, e := range v.Timeline.Events {
			if e.Type ==  "m.room.name" {
				rooms[fmt.Sprint(e.Content["name"])] = roomID
			}
		}

	}
	return rooms
}

func main () {
	bbs = NewBBS("./")
	bbs.Start()
	log.Print("Logging in...")
	username := os.Args[2]
	inviteUser := os.Args[4]
	roomname := "TestRoomFor" + username
	roomalias := roomname
	cli := Login(os.Args[1], username, os.Args[3])
	log.Println("Done!")
	cli.SetDisplayName(os.Args[2])
	log.Print("Creating room...")
	resp, err := cli.CreateRoom(&gomatrix.ReqCreateRoom{
	RoomAliasName: roomalias,
	Name : roomname,
	Preset: "public_chat",
})
	id := ":matrix.org"
	if err != nil {
		log.Println(err)
		log.Println("Failed!")
		id = "#" + roomalias + ":matrix.org"
	} else {
		log.Println("Done!")
		fmt.Println("Room:", resp.RoomID)
		id =  resp.RoomID
		log.Print("Joining room...")
		if _, err := cli.JoinRoom(id, "", nil); err != nil {
			panic(err)
		}
		log.Println("Done!")
		fmt.Println("Room:", id)
	}

	log.Print("Syncing...")
	filter := `{"room":{"timeline":{"limit":5000}}}`
	resSync, err := cli.SyncRequest(5, "", filter, false, "")
	if err != nil {
		panic(err)
	}
	log.Println("Done!")
	log.Print("Processing...")
	ProcessResponse(resSync, false)
	log.Println("Done!")

	nextbatch := resSync.NextBatch

	//get room id
	log.Print("Fetchng rooms...")
	rooms := ExtractRooms(resSync)
	log.Println(rooms)
	id = rooms[roomname]
	log.Println("Done!")
	log.Println(rooms)


	log.Print("Sending message...")
	if _, err := cli.SendText(id, "Hello") ; err != nil {
		panic(err)
	}
	log.Println("Done!")

	log.Print("Inviting user...")
	if _, err := cli.InviteUser(id, &gomatrix.ReqInviteUser{UserID: inviteUser}) ; err != nil {
		log.Println(err)
	}
	log.Println("Done!")


	go func() {
		for m := range bbs.Outgoing {
	 if m.Message == "text" {
			log.Print("Sending message...")
			if _, err := cli.SendText(id, m.PayloadString) ; err != nil {
				panic(err)
			}
			log.Println("Done!")
                        } else {
			log.Print("Sending file...")
			file, _ := os.Open(fmt.Sprint("botfiles/files/", m.PayloadString))
			res := QuickCommandStdout(exec.Command(`file`, `-b`, `--mime-type`, "botfiles/files/"+ m.PayloadString))
			log.Println("result '", res, "'")
			s, _ := file.Stat()
			res11 :=  strings.Trim(string(res), "  \n")
			log.Println()
			log.Println()
			log.Println("Image type: '", res11, "'")
			log.Println()
			log.Println()
			res2,err2 := cli.UploadToContentRepo(file, res11, s.Size())
			if err2 != nil {
				panic(err2)
			}
			cli.SendImage(id, m.PayloadString, res2.ContentURI)
			//fmt.Println(res2)
			}
		}
	}()

	for {
		resSync, err := cli.SyncRequest(10, nextbatch, filter, false, "")
		if err != nil {
			panic(err)
		}
		//fmt.Printf("%v: %# v\n\n\n", pretty.Formatter(resSync))
		events := ProcessResponse(resSync, true)
		for _, e := range events {
			//fmt.Printf("%v: %v\n", e.Sender, e)
			if e.Sender == "@donomii:matrix.org"{
				if e.Type ==  "m.room.message" {
					if  fmt.Sprint(e.Content["msgtype"]) == "m.text" {
						fmt.Printf("%v: %v\n", e.Sender, e.Content["body"])
			bbs.Incoming<-&BBSmessage{Message: "text", PayloadString:  fmt.Sprint(e.Content["body"])}
					} else {
						name, res, err := cli.Download(fmt.Sprint(e.Content["url"]))
						name =  fmt.Sprint(e.Content["body"])
						err = ioutil.WriteFile("botfiles/files/"+name, res, 0644)
						fmt.Println(err)
					}
				}
			}
		}
		nextbatch = resSync.NextBatch
		time.Sleep(4 * time.Second)
	}
	log.Println("Done!")
}
