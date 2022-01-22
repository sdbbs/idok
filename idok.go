package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"net/http"
	"bytes"
	"io/ioutil" //  Go 1.16 -- Q1 2021 -- ioutil deprecation: ioutil.ReadAll() => io.ReadAll()):

	"github.com/sdbbs/idok/asserver"
	"github.com/sdbbs/idok/tunnel"
	"github.com/sdbbs/idok/utils"
)

// Current VERSION - should be var and not const to be
// set at compile time (see Makefile OPTS)
var (
	VERSION = "notversionned"
)

func main() {

	// flags
	var (
		xbmcaddr        = flag.String("target", "", "xbmc/kodi ip (raspbmc address, ip or hostname)")
		username        = flag.String("login", "", "jsonrpc login (configured in xbmc settings)")
		password        = flag.String("password", "", "jsonrpc password (configured in xbmc settings)")
		viassh          = flag.Bool("ssh", false, "use SSH Tunnelling (need ssh user and password)")
		nossh           = flag.Bool("nossh", false, "force to not use SSH tunnel - usefull to override configuration file")
		port            = flag.Int("port", 8080, "local port (ignored if you use ssh option)")
		sshuser         = flag.String("sshuser", "pi", "ssh login")
		sshpassword     = flag.String("sshpass", "", "ssh password")
		sshport         = flag.Int("sshport", 22, "target ssh port")
		version         = flag.Bool("version", false, fmt.Sprintf("Print the current version (%s)", VERSION))
		xbmcport        = flag.Int("targetport", 80, "XBMC/Kodi jsonrpc port")
		stdin           = flag.Bool("stdin", false, "read file from stdin to stream")
		stdin_outnm     = flag.String("stdin_outnm", "out.mp4", "fake name of the stdin stream in the output url")
		stdin_nokodicmd = flag.Bool("stdin_nokodicmd", false, "If set/true, then no Kodi command is sent (tests stdin server)")
		confexample     = flag.Bool("conf-example", false, "print a configuration file example to STDOUT")
		disablecheck    = flag.Bool("disable-check-release", false, "disable release check")
		checknew        = flag.Bool("check-release", false, "check for new release")
		verbose         = flag.Bool("verbose", false, "bit more verbose log output")
		sendtokodi      = flag.Bool("sendtokodi", false, "Send the argument (URL) to SendToKodi addon in Kodi")
	)

	flag.Usage = utils.Usage

	flag.Parse()

	utils.SetVerbose(*verbose)
	asserver.SetVerbose(*verbose)
	asserver.SetNoKodiCmd(*stdin_nokodicmd)

	// print the current version
	if *version {
		fmt.Println(VERSION)
		fmt.Println("Compiled for", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	// If user asks to prints configuration file example, print it and exit
	if *confexample {
		utils.PrintExampleConfig()
		os.Exit(0)
	}

	// Set new configuration from options
	conf := &utils.Config{
		Target:       *xbmcaddr,
		Targetport:   *xbmcport,
		Localport:    *port,
		User:         *username,
		Password:     *password,
		Sshuser:      *sshuser,
		Sshpassword:  *sshpassword,
		Sshport:      *sshport,
		Ssh:          *viassh,
		ReleaseCheck: *disablecheck,
	}

	// check if conf file exists and override options
	if filename, found := utils.CheckLocalConfigFiles(); found {
		utils.LoadLocalConfig(filename, conf)
	}

	if *verbose {
		log.Println("Configuration settings are:")
		log.Println("Target       : ", *xbmcaddr)
		log.Println("Targetport   : ", *xbmcport)
		log.Println("Localport    : ", *port)
		log.Println("User         : ", *username)
		log.Println("Password     : ", *password)
		log.Println("Sshuser      : ", *sshuser)
		log.Println("Sshpassword  : ", *sshpassword)
		log.Println("Sshport      : ", *sshport)
		log.Println("Ssh          : ", *viassh)
		log.Println("ReleaseCheck : ", *disablecheck)
	}

	// Release check
	if *checknew || conf.ReleaseCheck {
		p := fmt.Sprintf("%s%c%s", os.TempDir(), os.PathSeparator, "idok_release_checked")
		stat, err := os.Stat(p)
		isold := false

		// if file exists and is old, we must recheck
		if err == nil && time.Since(stat.ModTime()) > time.Duration(24*3600*time.Second) {
			isold = true
		}

		// if doesn't exists, or is old, or we have -check-release flag, do check
		if os.IsNotExist(err) || isold || *checknew {
			release, err := utils.CheckRelease()
			if err != nil {
				log.Println(err)
			} else if release.TagName != VERSION {
				log.Println("A new release is available on github: ", release.TagName)
				log.Println("You can download it from ", release.Url)
			}
		}
		// create the file
		os.Create(p)

		// quit if -check-release flag
		if *checknew {
			os.Exit(0)
		}
	}

	if conf.Target == "" {
		fmt.Println("\033[33mYou must provide the xbmc server address\033[0m")
		flag.Usage()
		os.Exit(1)
	}

	utils.SetTarget(conf)

	// note: method is mostly Player.Open, via utils/const.go: BODY and YOUTUBEAPI
	// in xbmc/xbmc/interfaces/json-rpc/JSONServiceDescription.cpp:  { "Player.Open", CPlayerOperations::Open },
	// xbmc-master/xbmc/interfaces/json-rpc/PlayerOperations.cpp:JSONRPC_STATUS CPlayerOperations::Open(const std::string &method, ITransportLayer *transport, IClient *client, const CVariant &parameterObject, CVariant &result)
    //    "id":1,"jsonrpc":"2.0",
    //    "method":"Player.Open",
    //    "params": { "item": { "file": "tcp://192.168.0.5:8080/" } }

	if *verbose{
		log.Println("Checking if XMBC/Kodi is online, by asking it for it jsonrpc version")
	}
	resp, err := http.Post(utils.GlobalConfig.JsonRPC, "application/json", bytes.NewBufferString(
		`{"id":1, "jsonrpc":"2.0","method":"JSONRPC.Version"}` ))
	if *verbose{
		log.Println("jsonrpc err: ", err, ", Response:")
		log.Println("  Status           :", resp.Status           )
		log.Println("  StatusCode       :", resp.StatusCode       )
		log.Println("  Proto            :", resp.Proto            )
		log.Println("  ProtoMajor       :", resp.ProtoMajor       )
		log.Println("  ProtoMinor       :", resp.ProtoMinor       )
		log.Println("  Header           :", resp.Header           )
		//log.Println("  Body             :", resp.Body             )
		log.Println("  ContentLength    :", resp.ContentLength    )
		log.Println("  TransferEncoding :", resp.TransferEncoding )
		log.Println("  Close            :", resp.Close            )
		log.Println("  Uncompressed     :", resp.Uncompressed     )
		log.Println("  Trailer          :", resp.Trailer          )
		log.Println("  Request          :", resp.Request          )
		log.Println("  TLS              :", resp.TLS              )
	}

	if ( (err != nil) || (resp.StatusCode != http.StatusOK) ) { // http.StatusOK = 200
		fmt.Println("\nSorrie me lad, old Koddie cannot be reached.")
		fmt.Println("Probably best to exit now, ei?\n")
		os.Exit(2)
	} else {
		if *verbose{

			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			bodyString := string(bodyBytes)
			log.Println("Jsonrpc version: ", bodyString)
		}
	}

	var dir, file string

	// we don't use stdin, so we should check if scheme is file, youtube or other...
	if !*stdin {
		if len(flag.Args()) < 1 {
			fmt.Println("\033[33mYou must provide a file to serve\033[0m")
			flag.Usage()
			os.Exit(2)
		}

		if *sendtokodi {
			isvalid, validurl := utils.IsValidURL(flag.Arg(0))
			if (isvalid) {
				log.Println("Sending to SendToKodi: %s", validurl)
				utils.PlayViaSendToKodi(validurl)
				os.Exit(0)
			} else {
				log.Println("URL not good, exiting")
				os.Exit(2)
			}
		}

		if youtube, vid := utils.IsYoutubeURL(flag.Arg(0)); youtube {
			log.Println("Youtube video, using youtube addon from XBMC/Kodi")
			utils.PlayYoutube(vid)
			os.Exit(0)
		}

		if ok, local := utils.IsOtherScheme(flag.Arg(0)); ok {
			log.Println("\033[33mWarning, other scheme could be not supported by you Kodi/XBMC installation. If doesn't work, check addons and stream\033[0m")
			utils.SendBasicStream(flag.Arg(0), local)
			os.Exit(0)
		}

		// find the good path
		toserve := flag.Arg(0)
		dir = "."
		toserve, _ = filepath.Abs(toserve)
		file = filepath.Base(toserve)
		dir = filepath.Dir(toserve)

	}

	if conf.Ssh && !*nossh {
		config := tunnel.NewConfig(*sshuser, *sshpassword)
		// serve ssh tunnel !
		if !*stdin {
			if *verbose{
				log.Println("Running SshHTTPForward", config, file, dir)
			}
			tunnel.SshHTTPForward(config, file, dir)
		} else {
			if *verbose{
				log.Println("Running SshForwardStdin", config)
			}
			tunnel.SshForwardStdin(config)
		}
	} else {
		// serve local port !
		if !*stdin {
			if *verbose{
				log.Println("Running HttpServe", file, dir, *port)
			}
			asserver.HttpServe(file, dir, *port)
		} else {
			if *verbose{
				log.Println("Running TCPServeStdin", *port)
			}
			//asserver.TCPServeStdin(*port)
			asserver.HTTPServeStdin(*port, *stdin_outnm)
		}
	}
}
