package webapi

import "encoding/gob"
import "github.com/gorilla/sessions"
import "github.com/gorilla/context"
import "crypto/md5"
import "encoding/hex"
import "fmt"
import "net/http"
import "sitepod.io/sitepod/pkg/client"
import "sitepod.io/sitepod/pkg/util"
import "github.com/emicklei/go-restful"
import "path"

import "strings"
import "time"

type WebApi struct {
	container    *restful.Container
	client       *client.Client
	sessionStore sessions.Store
}

var rootdir string = "/tmp"

func NewWebApi(cc *client.Client) *WebApi {

	inst := &WebApi{}
	inst.container = restful.NewContainer()
	inst.sessionStore = sessions.NewFilesystemStore("", []byte("session1"))
	inst.client = cc

	ws := new(restful.WebService)
	ws.Path("/auth").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	ws.Route(ws.POST("/login").To(inst.Login))
	ws.Route(ws.GET("/whoami").To(inst.WhoAmI))

	//staticWs := new(restful.WebService)
	//ws.Route("/").Produces(restful.MIMk

	gob.Register(&SitepodSession{})
	inst.container.Add(ws)
	return inst
}

func staticFromPathParam(req *restful.Request, resp *restful.Response) {
	actual := path.Join(rootdir, req.PathParameter("subpath"))
	fmt.Printf("serving %s ... (from %s)\n", actual, req.PathParameter("subpath"))
	http.ServeFile(resp.ResponseWriter, req.Request, actual)
	return
}

func staticFromQueryParam(req *restful.Request, resp *restful.Response) {
	http.ServeFile(resp.ResponseWriter, req.Request, path.Join(rootdir, req.QueryParameter("resource")))
}

func (i *WebApi) SessionFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {

	session, err := i.sessionStore.Get(req.Request, "sitepodfe")
	if err != nil {
		resp.WriteHeaderAndEntity(500, NewAPIError("No session"))
		return
	}

	session.Values["lastSeen"] = time.Now()

	chain.ProcessFilter(req, resp)
}

func (i *WebApi) Login(req *restful.Request, resp *restful.Response) {

	entity := &LoginRequest{}
	err := req.ReadEntity(entity)

	if err != nil {
		resp.WriteHeaderAndEntity(400, NewAPIError("Invalid Payload"))
		return
	}

	entity.Email = strings.ToLower(strings.TrimSpace(entity.Email))

	if len(entity.Email) == 0 || len(entity.Password) == 0 {
		resp.WriteHeaderAndEntity(400, NewAPIError("Username and password required"))
		return
	}

	key := "sitepod-user-" + GetMD5Hash(entity.Email)

	user, exists := i.client.SitepodUsers().MaybeGetByKey(key)

	if !exists {
		resp.WriteHeaderAndEntity(404, NewAPIError("user not found: "+entity.Email))
		return
	}

	saltedPassword, err := Hash(entity.Password, user.Spec.Salt)

	if err != nil {
		resp.WriteHeaderAndEntity(500, NewAPIError("user not found"))
		return
	}

	if saltedPassword != user.Spec.SaltedPassword {
		resp.WriteHeaderAndEntity(403, struct{ Ok bool }{Ok: false})
		return
	}

	session, err := i.sessionStore.New(req.Request, "sitepodfe")
	session.Values["sess"] = SitepodSession{time.Now()}
	session.Save(req.Request, resp.ResponseWriter)
	resp.WriteHeaderAndEntity(200, struct{ Ok bool }{Ok: true})
	return
}

func (i *WebApi) WhoAmI(req *restful.Request, resp *restful.Response) {

	session, _ := i.sessionStore.Get(req.Request, "sitepodfe")

	if session.IsNew {
		resp.WriteHeaderAndEntity(503, NewAPIError("not logged in"))
		return
	}

	resp.WriteHeaderAndEntity(200, session.Values["sess"])
}

func (i *WebApi) Start() {
	server := &http.Server{Addr: ":8081", Handler: context.ClearHandler(i.container)}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func Hash(clearPassword string, salt string) (string, error) {

	hash, err := util.Sha512Crypt(clearPassword, salt)

	if err != nil {
		return "", err
	}

	return hash, nil
}
