package specgen

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"sitepod.io/sitepod/pkg/api/v1"
)

func SpecGenSSHServer(obj interface{}) error {
	ac := obj.(*v1.Appcomponent)

	//TODO really three times?
	ac.Spec.Type = "ssh"
	ac.Spec.DisplayName = "SSH Server"
	ac.Spec.Description = "SSH Server"

	ac.Spec.Image = "sitepod/sshdftp"
	//TODO never use latest antipattern
	ac.Spec.ImageVersion = "latest"
	ac.Spec.MountHome = true
	ac.Spec.MountEtcs = true

	privateKey, publicKey := genNewKeys()
	privateKeyPemFile := v1.AppComponentConfigFile{
		Name:      "sshprivate",
		Content:   privateKey,
		Directory: "/etc/sitepod/ssh",
		FileMode:  "0600",
		Uid:       0,
		Gid:       0,
	}

	publicKeySSHFile := v1.AppComponentConfigFile{
		Name:      "sshpublic",
		Content:   publicKey,
		Directory: "/etc/sitepod/ssh",
		FileMode:  "0600",
		Uid:       0,
		Gid:       0,
	}
	ac.Spec.ConfigFiles = append(ac.Spec.ConfigFiles, privateKeyPemFile, publicKeySSHFile)
	return nil
}

func genNewKeys() (privateKeyPem string, sshPublicKey string) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2014)
	if err != nil {
		panic(err)
	}

	privateKeyDer := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privateKeyDer,
	}

	privateKeyPem = string(pem.EncodeToMemory(&privateKeyBlock))

	publicKey := privateKey.PublicKey

	pub, err := ssh.NewPublicKey(&publicKey)

	if err != nil {
		panic(err)
	}

	pkBytes := pub.Marshal()

	sshPublicKey = fmt.Sprintf("ssh-rsa %s %s", base64.StdEncoding.EncodeToString(pkBytes), "placeholder@sitepod.io")

	return
}

//type registry struct {
//m map[string]func(interface{}) error
//}

//func Initialize(key string, obj interface{}) error {

//if initializer, ok := m[key]; ok {
//return intializer(obj)
//} else {
//return errors.New("No specgen for " + key)
//}
//}

//func init() {
//registry = make(map[string]func(interface{}) error)
//}
