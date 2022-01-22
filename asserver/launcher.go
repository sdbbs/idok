// This package gives method to send streams as an http server.
package asserver

import (
	"fmt"
	"github.com/sdbbs/idok/utils"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
//	"bufio"
	"time"
)

var verbose bool
func SetVerbose(inbool bool) {
	verbose = inbool
	if verbose {
		log.Println(" asserver verbose: ", verbose)
	}
}

var stdin_nokodicmd bool
func SetNoKodiCmd(inbool bool) {
	stdin_nokodicmd = inbool
	if verbose {
		log.Println(" asserver stdin_nokodicmd: ", stdin_nokodicmd)
	}
}

// Open a port locally and tell to kodi to stream
// from this port
func HttpServe(file, dir string, port int) {

	localip, err := utils.GetLocalInterfaceIP()
	log.Println(localip)
	if err != nil {
		log.Fatal(err)
	}

	// handle file http response
	fullpath := filepath.Join(dir, file)
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, fullpath)
	}))

	// send xbmc the file query
	go utils.Send("http", localip, file, port)

	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil));
}

// NOTE: TCPServeStdin sets Kodi to load `"file": "tcp://192.168.0.5:8080/"`
// however, in Kodi 19 log viewer it fails with: WARNING <general>: Create - unsupported protocol(tcp) in tcp://192.168.0.5:8080/
// (`"file" : "plugin://plugin.video.youtube/?action=play_video&videoid=o5snlP8Y5GY"` works)
// Serve STDIN stream from a local port
func TCPServeStdin(port int) {

	localip, err := utils.GetLocalInterfaceIP()
	log.Println(localip)
	if err != nil {
		log.Fatal(err)
	}

	if verbose {
		log.Println("Running TCPServeStdin: Send", "tcp", localip, "", port)
	}
	con, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Fatal(err)
	}
	// send xbmc the file query
	//utils.Send("tcp", localip, "", port) // was go 
	utils.Send("http", localip, "", port) // was go 
	c, err := con.Accept()
	if verbose {
		log.Println("Running TCPServeStdin: after con.Accept c, err", c, err)
	}
	if err != nil {
		log.Fatal(err)
	}
	go io.Copy(c, os.Stdin)
}

// test with: while [ 1 ]; do echo -ne "GET / HTTP/1.0\n\n\n" | nc 127.0.0.1 9000 | hexdump -C; sleep 0.25; echo -n .; done
func HTTPServeStdin(port int, req_stream_name string) {

	if verbose {
		log.Println("Entered HTTPServeStdin")
	}

	localip, err := utils.GetLocalInterfaceIP()
	log.Println(localip)
	if err != nil {
		log.Fatal(err)
	}

	// Create a mux for routing incoming requests
	m := http.NewServeMux()
	//stdin_reader := bufio.NewReader(os.Stdin) // SO:20895552

	// All URLs will be handled by this function
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//w.Write([]byte(os.Stdin)) // invalid type conversion
		//w.Write(os.Stdin)
		//io.Copy(w, os.Stdin)
		log.Println("Entered handlefunc")
		firsttime := true
		data := make([]byte, 2048) // SO:14469511
		for {
			//data = data[:cap(data)]
			//n, err := stdin_reader.Read(data)
			//if err != nil {
			//	if err == io.EOF {
			//		break
			//	}
			//	fmt.Println(err)
			//	return
			//}
			//fmt.Printf("%d: %v\n", n, data) // SO:24489384
			//w.Write(data)
			_, err := io.ReadFull(os.Stdin, data) // SO:29060922; was n, err
			if firsttime {
				// this is the very first time we see any data, before that ReadFull blocks
				firsttime = false
			}
			for err == nil {
				//fmt.Printf("%d: %v\n", n, data) // SO:24489384
				w.Write(data)
				_, err = io.ReadFull(os.Stdin, data)
			}
			if err != io.EOF {
				panic(err)
			}
		}
	})
	// Create a server listening on port
	srvaddr := fmt.Sprintf(":%d", port)
	s := &http.Server{
		Addr:    srvaddr,
		Handler: m,
	}

	// this section should delay sending command to xbmc/Kodi, until first bytes arrive for the stream
	// actually, nothing is visible much but change of mtime (size is still reported zero)
	// so, wait for mtime change instead
	//var size int64 = 0
	//for size == 0 { // that is: while size==0
	//	stdinfi, err := os.Stdin.Stat() // SO:22563616
	//	if err != nil {
	//		fmt.Println("os.Stdin.Stat() error", err)
	//		os.Exit(1)
	//	}
	//	size = stdinfi.Size()
	//	//log.Println("os.Stdin.Stat() ", stdinfi)
	//	fmt.Printf("stdin name %s size %d mode %s mtime %s isdir %t\n  sys %s\n", stdinfi.Name(), size, stdinfi.Mode(), stdinfi.ModTime(), stdinfi.IsDir(), stdinfi.Sys()) // stdinfi.Sys()
	//	time.Sleep(250*time.Millisecond)
	//}
	var first_mtime, now_mtime time.Time
	stdinfi, err := os.Stdin.Stat() // SO:22563616
	if err != nil {
		fmt.Println("os.Stdin.Stat() error", err)
		os.Exit(1)
	}
	// there is no a = b = 1 in go, so assign single value to multiple vars like this:
	first_mtime, now_mtime = stdinfi.ModTime(), stdinfi.ModTime()
	for first_mtime == now_mtime {
		//if verbose {
		//	fmt.Printf("stdin name %s size %d mode %s mtime %s isdir %t\n  sys %s\n", stdinfi.Name(), stdinfi.Size(), stdinfi.Mode(), stdinfi.ModTime(), stdinfi.IsDir(), stdinfi.Sys()) // stdinfi.Sys()
		//}
		time.Sleep(250*time.Millisecond)
		stdinfi, err := os.Stdin.Stat() // SO:22563616
		if err != nil {
			fmt.Println("os.Stdin.Stat() error", err)
			os.Exit(1)
		}
		now_mtime = stdinfi.ModTime()
	}

	if verbose {
		log.Println("Running HTTPServeStdin: srvaddr", srvaddr, "http", localip, port)
	}

	if ! stdin_nokodicmd {
		time.Sleep(1000*time.Millisecond)
		// send xbmc the file query
		file := req_stream_name
		utils.Send("http", localip, file, port)// was go
		if verbose {
			log.Println("Sent Kodi command for http", localip, file, port)
		}
	}

	// Continue to process new requests until an error occurs
	log.Fatal(s.ListenAndServe())
}
