package specgen

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"sitepod.io/sitepod/pkg/api/v1"
	"text/template"
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
		Filename:  "ssh_host_rsa_key",
		Directory: "/etc/sitepod/ssh",
		FileMode:  "0600",
		Uid:       0,
		Gid:       0,
	}

	publicKeySSHFile := v1.AppComponentConfigFile{
		Name:      "sshpublic",
		Content:   publicKey,
		Filename:  "ssh_host_rsa_key.pub",
		Directory: "/etc/sitepod/ssh",
		FileMode:  "0600",
		Uid:       0,
		Gid:       0,
	}

	sshdConfigContent := generateSSHDConfig()
	sshdConfigFile := v1.AppComponentConfigFile{
		Name:      "sshdconfig",
		Content:   sshdConfigContent,
		Filename:  "sshd_config",
		Directory: "/etc/sitepod/ssh",
		FileMode:  "0600",
		Uid:       0,
		Gid:       0,
	}

	ac.Spec.ConfigFiles = append(ac.Spec.ConfigFiles,
		privateKeyPemFile,
		publicKeySSHFile,
		sshdConfigFile)
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

func generateSSHDConfig() string {
	//TODO figure out where to store templates
	template, err := template.ParseFiles("../../templates/sshd_config")
	if err != nil {
		panic(err)
	}
	buffer := bytes.NewBuffer([]byte{})
	err = template.Execute(buffer, struct{}{})
	if err != nil {
		panic(err)
	}
	return buffer.String()
}
