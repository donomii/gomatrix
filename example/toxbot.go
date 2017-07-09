package main

import (
	"bytes"
	"os/exec"
	"github.com/necrophonic/go-eliza"
	"strconv"
	"log"
	//"math/rand"
	"strings"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"github.com/codedust/go-tox"
	"io/ioutil"
	"os"
	"os/signal"
	"time"
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
func QuickCommandStdout (cmd *exec.Cmd) string{
    fmt.Println()
    in := strings.NewReader("")
    cmd.Stdin = in 
    var out bytes.Buffer
    cmd.Stdout = &out
    var err bytes.Buffer
    cmd.Stderr = &err
    //res := cmd.Run()
    cmd.Run()
    //fmt.Printf("Command result: %v\n", res)
    ret := fmt.Sprintf("%s", out)
    //fmt.Println(ret)
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
	Tox *gotox.Tox
	FriendNumber uint32
}

func NewBBS(dir string) * BBSdata{
	b := BBSdata{ BaseDir: dir, FilesDir: "files", Incoming: make(chan * BBSmessage, 2), Outgoing:  make(chan * BBSmessage, 2) }
	return &b
}

func (b BBSdata) Start() {
	go func() {
		for m := range b.Incoming {
			b.HandleMessage(m)	
		}
	}()
	go func() {
		for m := range b.Outgoing{
			if m.Message == "text" {
				m.Tox.FriendSendMessage(m.FriendNumber, gotox.TOX_MESSAGE_TYPE_NORMAL, m.PayloadString)
			} else {
				file, err := os.Open(fmt.Sprint("files/", m.PayloadString))
				if err != nil {
					 b.RespondToMessageText(m, "File not found. Please 'cd' into go-tox/examples and type 'go run example.go'")
					file.Close()
					return
				}

				// get the file size
				stat, err := file.Stat()
				if err != nil {
					 b.RespondToMessageText(m, "Could not read file stats.")
					file.Close()
					return
				}

				fmt.Println("File size is ", stat.Size())

				fileNumber, err := m.Tox.FileSend(m.FriendNumber, gotox.TOX_FILE_KIND_DATA, uint64(stat.Size()), nil, m.PayloadString)
				if err != nil {
					 b.RespondToMessageText(m, "t.FileSend() failed.")
					file.Close()
					return
				}
				transfers[fileNumber] = FileTransfer{fileHandle: file, fileSize: uint64(stat.Size()), fileName: m.PayloadString}
				log.Println("File transfer started for file number ", fileNumber)

			}
	}
	}()
}
 
func (b BBSdata) RespondToMessageText(m *BBSmessage, t string ) {
		bbs.Outgoing<-&BBSmessage{Message: "text", PayloadString: t, Tox: m.Tox, FriendNumber: m.FriendNumber }
}

func (b BBSdata) RespondToMessageFile(m *BBSmessage, t string ) {
		bbs.Outgoing<-&BBSmessage{Message: "file", PayloadString: t, Tox: m.Tox, FriendNumber: m.FriendNumber }
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
	} else if strings.HasPrefix(lmessage, "ip") {
		res := QuickCommandStdout(exec.Command(`ifconfig`, `-a`))
		bbs.Outgoing<-&BBSmessage{Message: "text", PayloadString: res, Tox: m.Tox, FriendNumber: m.FriendNumber}
	} else if strings.HasPrefix(lmessage, "list") {
		//List the received files dir
		files, err := ioutil.ReadDir("files")
		if err != nil {
			log.Fatal(err)
		}

		message := ""
		for i, v := range files {
			message = fmt.Sprint(message, i, ".    ", v.Name(), "\n")
		}

		 b.RespondToMessageText(m, message)
	} else if strings.HasPrefix(lmessage, "delete") {
	
		numStr := strings.Trim(strings.Replace(lmessage, "delete", "",  1), " ")
		n, err := strconv.ParseInt(numStr, 10, 64)
		log.Println("Sending file number ", n)
		//List the received files dir
		files, err := ioutil.ReadDir("files")
		if err != nil {
			log.Fatal(err)
		}

		
		//filename :=  files[rand.Intn(len(files))].Name()
		filename :=  files[n].Name()
		log.Println("Deleting file ", filename)
		os.Remove(fmt.Sprint("files/", filename))
	} else if strings.HasPrefix(lmessage, "file") {
		
		numStr := strings.Trim(strings.Replace(lmessage, "file", "",  1), " ")
		n, err := strconv.ParseInt(numStr, 10, 64)
		log.Println("Sending file number ", n)
		//List the received files dir
		files, err := ioutil.ReadDir("files")
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
 

// Map of active file transfers
var transfers = make(map[uint32]FileTransfer)
var bbs *BBSdata

func main() {
	bbs = NewBBS("./")
	bbs.Start()
	var newToxInstance bool = false
	var filepath string
	var options *gotox.Options

	flag.StringVar(&filepath, "save", "./example_savedata", "path to save file")
	flag.Parse()

	fmt.Printf("[INFO] Using Tox version %d.%d.%d\n", gotox.VersionMajor(), gotox.VersionMinor(), gotox.VersionPatch())


	savedata, err := loadData(filepath)
	if err == nil {
		fmt.Println("[INFO] Loading Tox profile from savedata...")
		options = &gotox.Options{
			IPv6Enabled:  true,
			UDPEnabled:   true,
			ProxyType:    gotox.TOX_PROXY_TYPE_NONE,
			ProxyHost:    "127.0.0.1",
			ProxyPort:    5555,
			StartPort:    0,
			EndPort:      0,
			TcpPort:      0, // only enable TCP server if your client provides an option to disable it
			SaveDataType: gotox.TOX_SAVEDATA_TYPE_TOX_SAVE,
			SaveData:     savedata}
	} else {
		fmt.Println("[INFO] Creating new Tox profile...")
		options = nil // default options
		newToxInstance = true
	}

	tox, err := gotox.New(options)
	if err != nil {
		panic(err)
	}

	if newToxInstance {
		tox.SelfSetName("gotoxBot")
		tox.SelfSetStatusMessage("gotox is cool!")
	}

	addr, _ := tox.SelfGetAddress()
	fmt.Println("ID: ", hex.EncodeToString(addr))
        toxid, _ := tox.SelfGetAddress()
	haddr := hex.EncodeToString(toxid[0:])
	fmt.Println(strings.ToUpper(haddr))


	err = tox.SelfSetStatus(gotox.TOX_USERSTATUS_NONE)

	// Register our callbacks
	tox.CallbackFriendRequest(onFriendRequest)
	tox.CallbackFriendMessage(onFriendMessage)
	tox.CallbackFileRecvControl(onFileRecvControl)
	tox.CallbackFileChunkRequest(onFileChunkRequest)
	tox.CallbackFileRecv(onFileRecv)
	tox.CallbackFileRecvChunk(onFileRecvChunk)

	/* Connect to the network
	 * Use more than one node in a real world szenario. This example relies one
	 * the following node to be up.
	 */
	pubkey, _ := hex.DecodeString("A179B09749AC826FF01F37A9613F6B57118AE014D4196A0E1105A98F93A54702")
	server := &Server{"205.185.116.116", 33445, pubkey}


	log.Println("Bootstrapping complete")
	err = tox.Bootstrap(server.Address, server.Port, server.PublicKey)
	if err != nil {
		panic(err)
	}

	log.Println("Bootstrap complete")

	isRunning := true

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	ticker := time.NewTicker(25 * time.Millisecond)

	for isRunning {
		select {
		case <-c:
			fmt.Printf("\nSaving...\n")
			if err := saveData(tox, filepath); err != nil {
				fmt.Println("[ERROR]", err)
			}
			fmt.Println("Killing")
			isRunning = false
			tox.Kill()
		case <-ticker.C:
			tox.Iterate()
		}
	}
}

func onFriendRequest(t *gotox.Tox, publicKey []byte, message string) {
	fmt.Printf("New friend request from %s\n", hex.EncodeToString(publicKey))
	fmt.Printf("With message: %v\n", message)
	// Auto-accept friend request
	t.FriendAddNorequest(publicKey)
}

func onFriendMessage(t *gotox.Tox, friendNumber uint32, messagetype gotox.ToxMessageType, message string) {
	//bbs.HandleMessage(message, t, friendNumber)
	bbs.Incoming<-&BBSmessage{Message: "text", PayloadString: message, Tox: t, FriendNumber: friendNumber}
	if messagetype == gotox.TOX_MESSAGE_TYPE_NORMAL {
		fmt.Printf("New message from %d : %s\n", friendNumber, message)
	} else {
		fmt.Printf("New action from %d : %s\n", friendNumber, message)
	}
}

func onFileRecv(t *gotox.Tox, friendNumber uint32, fileNumber uint32, kind gotox.ToxFileKind, filesize uint64, filename string) {
	if kind == gotox.TOX_FILE_KIND_AVATAR {

		if filesize > MAX_AVATAR_SIZE {
			// reject file send request
			t.FileControl(friendNumber, fileNumber, gotox.TOX_FILE_CONTROL_CANCEL)
			return
		}

		publicKey, _ := t.FriendGetPublickey(friendNumber)
		file, err := os.Create("example_" + hex.EncodeToString(publicKey) + ".png")
		if err != nil {
			fmt.Println("[ERROR] Error creating file", "example_"+hex.EncodeToString(publicKey)+".png")
		}

		// append the file to the map of active file transfers
		transfers[fileNumber] = FileTransfer{fileHandle: file, fileSize: filesize}

		// accept the file send request
		t.FileControl(friendNumber, fileNumber, gotox.TOX_FILE_CONTROL_RESUME)

	} else {
		// accept files of any length

		os.Mkdir("incoming", os.FileMode(0777))
		file, err := os.Create("incoming/" + filename)
		if err != nil {
			fmt.Println("[ERROR] Error creating file", filename)
		}

		// append the file to the map of active file transfers
		transfers[fileNumber] = FileTransfer{fileHandle: file, fileSize: filesize, fileName: filename}

		// accept the file send request
		t.FileControl(friendNumber, fileNumber, gotox.TOX_FILE_CONTROL_RESUME)
	}
}

func onFileRecvControl(t *gotox.Tox, friendNumber uint32, fileNumber uint32, fileControl gotox.ToxFileControl) {
	transfer, ok := transfers[fileNumber]
	if !ok {
		fmt.Println("Error: File handle does not exist")
		return
	}

	if fileControl == gotox.TOX_FILE_CONTROL_CANCEL {
		// delete file handle
		transfer.fileHandle.Sync()
		transfer.fileHandle.Close()
		delete(transfers, fileNumber)
	}
}

func onFileChunkRequest(t *gotox.Tox, friendNumber uint32, fileNumber uint32, position uint64, length uint64) {
	transfer, ok := transfers[fileNumber]
	if !ok {
		fmt.Println("Error: File handle does not exist")
		return
	}

	// a zero-length chunk request confirms that the file was successfully transferred
	if length == 0 {
		transfer.fileHandle.Close()
		delete(transfers, fileNumber)
		fmt.Println("File transfer completed (sending)", fileNumber)
		return
	}

	// read the requested data to send
	data := make([]byte, length)
	_, err := transfers[fileNumber].fileHandle.ReadAt(data, int64(position))
	if err != nil {
		fmt.Println("Error reading file", err)
		return
	}

	// send the requested data
	t.FileSendChunk(friendNumber, fileNumber, position, data)
}

func onFileRecvChunk(t *gotox.Tox, friendNumber uint32, fileNumber uint32, position uint64, data []byte) {
	transfer, ok := transfers[fileNumber]
	if !ok {
		if len(data) == 0 {
			// ignore the zero-length chunk that indicates that the transfer is
			// complete (see below)
			return
		}

		fmt.Println("Error: File handle does not exist")
		return
	}

	// write the received data to the file handle
	transfer.fileHandle.WriteAt(data, (int64)(position))

	// file transfer completed
	if position+uint64(len(data)) >= transfer.fileSize {
		// Some clients will send us another zero-length chunk without data (only
		// required for streams, not necessary for files with a known size) and some
		// will not.
		// We will delete the file handle now (we aleady received the whole file)
		// and ignore the file handle error when the zero-length chunk arrives.

		transfer.fileHandle.Sync()
		transfer.fileHandle.Close()
		os.Mkdir("files", os.FileMode(0777))
		os.Rename(fmt.Sprint("incoming/", transfer.fileName), fmt.Sprint("files/", transfer.fileName))
		delete(transfers, fileNumber)
		fmt.Println("File transfer completed (receiving)", fileNumber)
		t.FriendSendMessage(friendNumber, gotox.TOX_MESSAGE_TYPE_NORMAL, "Thanks!")
	}
}

// loadData reads a file and returns its content as a byte array
func loadData(filepath string) ([]byte, error) {
	if len(filepath) == 0 {
		return nil, errors.New("Empty path")
	}

	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	return data, err
}

// saveData writes the savedata from toxcore to a file
func saveData(t *gotox.Tox, filepath string) error {
	if len(filepath) == 0 {
		return errors.New("Empty path")
	}

	data, err := t.GetSavedata()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath, data, 0644)
	return err
}
