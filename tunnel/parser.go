package tunnel

import (
	"github.com/sdbbs/idok/tunnel/go.crypto/ssh"
	"io/ioutil"
	"log"
)

// Parse local ssh private key to get signer
func parseSSHKeys(keyfile string) (ssh.Signer, error) {
	content, err := ioutil.ReadFile(keyfile)
	private, err := ssh.ParsePrivateKey(content)
	if err != nil {
		log.Println("Unable to parse private key")
	}
	return private, err
}
