package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

const TICK_CHECK = 1
var verbose = false

// response of get players
type itemresp struct {
	Id      int
	Jsonrpc string
	Result  []map[string]interface{}
}

func SetVerbose(inbool bool) {
	verbose = inbool
	if verbose {
		log.Println(" utils verbose: ", verbose)
	}
}

// Send the play command to Kodi/XBMC.
func Send(scheme, host, file string, port int) <-chan int {

	u := url.URL{Path: file}
	file = u.String()
	var addr string
	if file == "/" {
		addr = fmt.Sprintf("%s://%s:%d/", scheme, host, port)
	} else if file == "-" {
		addr = fmt.Sprintf("%s://%s:%d", scheme, host, port)
	} else {
		addr = fmt.Sprintf("%s://%s:%d/%s", scheme, host, port, file)
	}

	request := []interface{} {GlobalConfig.JsonRPC, "application/json", bytes.NewBufferString(fmt.Sprintf(BODY, addr))}
	if verbose {
		log.Println(" Send request: ", request)
	}
	r, err := http.Post(GlobalConfig.JsonRPC, "application/json", bytes.NewBufferString(fmt.Sprintf(BODY, addr)))
	if err != nil {
		log.Fatal(err)
	}
	response, _ := ioutil.ReadAll(r.Body)
	log.Println(string(response))

	// and wait media end
	return checkPlaying()
}

// send basic stream...
func SendBasicStream(uri string, local bool) <-chan int {
	_body := fmt.Sprintf(BODY, uri)

	r, err := http.Post(GlobalConfig.JsonRPC, "application/json", bytes.NewBufferString(_body))
	if err != nil {
		log.Fatal(err)
	}
	response, _ := ioutil.ReadAll(r.Body)
	log.Println(string(response))

	// handle CTRL+C to stop
	go OnQuit()

	// and wait the end of media
	return checkPlaying()
}

// Ask to play youtube video.
func PlayYoutube(vidid string) <-chan int {

	request := []interface{} {GlobalConfig.JsonRPC, "application/json", bytes.NewBufferString(fmt.Sprintf(YOUTUBEAPI, vidid))}
	if verbose {
		log.Println(" PlayYoutube request: ", request)
	}
	r, err := http.Post(GlobalConfig.JsonRPC, "application/json", bytes.NewBufferString(fmt.Sprintf(YOUTUBEAPI, vidid)))
	if err != nil {
		log.Fatal(err)
	}
	response, _ := ioutil.ReadAll(r.Body)
	log.Println(string(response))

	// handle CTRL+C to stop
	go OnQuit()

	return checkPlaying()
}

func PlayViaSendToKodi(vidid string) <-chan int {

	request := []interface{} {GlobalConfig.JsonRPC, "application/json", bytes.NewBufferString(fmt.Sprintf(SENDTOKODIAPI, vidid))}
	if verbose {
		log.Println(" PlayViaSendToKodi request: ", request)
	}
	r, err := http.Post(GlobalConfig.JsonRPC, "application/json", bytes.NewBufferString(fmt.Sprintf(SENDTOKODIAPI, vidid)))
	if err != nil {
		log.Fatal(err)
	}
	response, _ := ioutil.ReadAll(r.Body)
	log.Println(string(response))

	// handle CTRL+C to stop
	go OnQuit()

	return checkPlaying()
}

// test if media is playing, write 1 in returned chan when media has finished.
func checkPlaying() <-chan int {
	tick := time.Tick(TICK_CHECK * time.Second)
	c := make(chan int, 0)
	go func() {
		for _ = range tick {
			resp := getActivePlayer()
			if len(resp.Result) == 0 {
				c <- 1
			}
		}
	}()
	return c
}

// return active player from XBMC.
func getActivePlayer() *itemresp {
	r, _ := http.Post(GlobalConfig.JsonRPC, "application/json", bytes.NewBufferString(GETPLAYERBODY))
	response, _ := ioutil.ReadAll(r.Body)
	resp := &itemresp{}
	resp.Result = make([]map[string]interface{}, 0)
	json.Unmarshal(response, resp)
	return resp
}
