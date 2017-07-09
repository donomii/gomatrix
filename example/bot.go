package main

import (
	"time"
	"bytes"
	"os/exec"
	"github.com/necrophonic/go-eliza"
	"strconv"
	"log"
	//"math/rand"
	"strings"
	"fmt"
	"io/ioutil"
	"os"
)

type Server struct {
	Address   string
	Port      uint16
	PublicKey []byte
}

const MAX_AVATAR_SIZE = 65536 // see github.com/Tox/Tox-STS/blob/master/STS.md#avatars

type FileTransfer struct {
	fileHandle *os.File
	fileSize   uint64
	fileName   string
}

//Run exec.Cmd, capture and return STDOUT
func QuickCommandStdout (cmd *exec.Cmd) string {
    fmt.Println()
    in := strings.NewReader("")
    cmd.Stdin = in 
    var out bytes.Buffer
    cmd.Stdout = &out
    var err bytes.Buffer
    cmd.Stderr = &err
    //res := cmd.Run()
    cmd.Run()
   b := out.Bytes()
    fmt.Printf("Command result: %v\n", string(b))
    ret := fmt.Sprintf("%s", b)
    fmt.Println(ret)
    return ret
}


type BBSbot interface {
}
 
type BBSdata struct {
	BaseDir string
	FilesDir string
	Incoming chan *BBSmessage
	Outgoing  chan *BBSmessage
}

type BBSmessage struct {
	Message string
	PayloadString string
	PayloadBytes []byte
	UserData interface{}
}

func NewBBS(dir string) * BBSdata{
	b := BBSdata{ BaseDir: dir + "/botfiles", FilesDir: "files", Incoming: make(chan * BBSmessage, 2), Outgoing:  make(chan * BBSmessage, 2) }
	return &b
}

func (b BBSdata) Start() {
	os.Mkdir(b.BaseDir, os.FileMode(0777))
	os.Mkdir(b.BaseDir + "/" + b.FilesDir, os.FileMode(0777))
	go func() {
		for m := range b.Incoming {
			b.HandleMessage(m)	
		}
	}()
}
 
func (b BBSdata) RespondToMessageText(m *BBSmessage, t string ) {
		bbs.Outgoing<-&BBSmessage{Message: "text", PayloadString: t, UserData: m.UserData }
}

func (b BBSdata) RespondToMessageFile(m *BBSmessage, t string ) {
		bbs.Outgoing<-&BBSmessage{Message: "file", PayloadString: t, UserData: m.UserData }
}


func AppendStringToFile(path, text string) error {
      f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
      if err != nil {
              return err
      }
      defer f.Close()

      _, err = f.WriteString(text)
      if err != nil {
              return err
      }
      return nil
}

func (b BBSdata) HandleMessage(m *BBSmessage) {
	message := m.PayloadString
	lmessage := strings.ToLower(message)
	if strings.HasPrefix(lmessage, "help") ||  strings.HasPrefix(lmessage, "hi") ||  strings.HasPrefix(lmessage, "hello"){
		b.RespondToMessageText(m, `
Type 'list' to see a list of files.
Type 'file 1' to receive file 1.
Send any file to add it to the list.

Or type a sentence to chat to Eliza.
`)
	} else if strings.HasPrefix(lmessage, "b ") {
		str := strings.Trim(strings.Replace(message, "b ", "",  1), " ")
		str = strings.Trim(strings.Replace(str, "B ", "",  1), " ")
		AppendStringToFile("botfiles/blog.txt", fmt.Sprint(time.Now(), " | ",  str, "\n") )
	} else if strings.HasPrefix(lmessage, "ip") {
		res := QuickCommandStdout(exec.Command(`ifconfig`, `-a`))
		b.RespondToMessageText(m, res)
	} else if strings.HasPrefix(lmessage, "list") {
		//List the received files dir
		files, err := ioutil.ReadDir(b.BaseDir + "/" + b.FilesDir)
		if err != nil {
			log.Fatal(err)
		}

		message := ""
		for i, v := range files {
			message = fmt.Sprint(message, i, ".    ", v.Name(), "\n")
		}

		 b.RespondToMessageText(m, message)
	} else if strings.HasPrefix(lmessage, "delete ") {
	
		numStr := strings.Trim(strings.Replace(lmessage, "delete ", "",  1), " ")
		n, err := strconv.ParseInt(numStr, 10, 64)
		log.Println("Sending file number ", n)
		//List the received files dir
		files, err := ioutil.ReadDir(b.BaseDir + "/" + b.FilesDir)
		if err != nil {
			log.Fatal(err)
		}

		
		//filename :=  files[rand.Intn(len(files))].Name()
		filename :=  files[n].Name()
		log.Println("Deleting file ", filename)
		os.Remove(fmt.Sprint(b.BaseDir + "/" + b.FilesDir + "/", filename))
	} else if strings.HasPrefix(lmessage, "file ") {
		
		numStr := strings.Trim(strings.Replace(lmessage, "file ", "",  1), " ")
		n, err := strconv.ParseInt(numStr, 10, 64)
		log.Println("Sending file number ", n)
		//List the received files dir
		files, err := ioutil.ReadDir(b.BaseDir + "/" + b.FilesDir)
		if err != nil {
			log.Fatal(err)
		}

		
		//filename :=  files[rand.Intn(len(files))].Name()
		filename :=  files[n].Name()
		log.Println("Sendig file ", filename)
		b.RespondToMessageFile(m, filename)
		
	} else {
		//t.FriendSendMessage(friendNumber, gotox.TOX_MESSAGE_TYPE_NORMAL, "Type 'help' for available commands.")
		response, err := eliza.AnalyseString(lmessage)
		if err != nil {
			panic(err)
		}
		 b.RespondToMessageText(m, response)

	}

}
 


