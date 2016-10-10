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

func NewWebApi(cc *client.Client) *WebApi {

	inst := &WebApi{}
	inst.container = restful.NewContainer()

	fileStore := sessions.NewFilesystemStore("", []byte("session1"))
	//	fileStore.Options.Secure = true
	fileStore.MaxAge(86400) //one day

	inst.sessionStore = fileStore

	inst.client = cc

	ws := new(restful.WebService)
	ws.Path("/api").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	ws.Route(ws.POST("/login").To(inst.Login))
	ws.Route(ws.POST("/logout").To(inst.Logout))
	ws.Route(ws.GET("/context").To(inst.WhoAmI))

	staticWs := new(restful.WebService)
	staticWs.Route(ws.GET("/ui/{subpath:*}").To(staticFromPathParam))

	gob.Register(&SitepodSession{})
	inst.container.Add(ws)
	inst.container.Add(staticWs)
	return inst
}

func staticFromPathParam(req *restful.Request, resp *restful.Response) {

	subPath := req.Request.URL.Path[4:]

	baseDir := "/home/matt/ws/sitepodfe/resources/public"
	fmt.Println(subPath)
	if strings.Index(subPath, "css/") == 0 || strings.Index(subPath, "js/") == 0 || strings.Index(subPath, "img/") == 0 {
		actual := path.Join(baseDir, subPath)
		if strings.Index(subPath, "js/") == 0 {
			resp.Header().Set("Content-Type", "text/javascript; charset=UTF-8")
		}

		http.ServeFile(resp.ResponseWriter, req.Request, actual)
		return
	}

	http.ServeFile(resp.ResponseWriter, req.Request, path.Join(baseDir, "index.html"))
}

//func (i *WebApi) SessionFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {

//session, err := i.sessionStore.Get(req.Request, "sitepodfe")
//if err != nil {
//resp.WriteHeaderAndEntity(500, NewAPIError("No session"))
//return
//}

//session.Values["lastSeen"] = &time.Now()

//chain.ProcessFilter(req, resp)
//}

func (i *WebApi) Logout(req *restful.Request, resp *restful.Response) {
	session, _ := i.sessionStore.Get(req.Request, "sitepodfe")
	if !session.IsNew {
		delete(session.Values, "sess")
		session.Options.MaxAge = -1
	}
	session.Save(req.Request, resp.ResponseWriter)
	resp.WriteHeaderAndEntity(200, struct{}{})
}

func (i *WebApi) Login(req *restful.Request, resp *restful.Response) {

	entity := &LoginRequest{}
	err := req.ReadEntity(entity)

	if err != nil {
		resp.WriteHeaderAndEntity(400, NewAPIError("Invalid Payload"))
		return
	}

	entity.Data.Email = strings.ToLower(strings.TrimSpace(entity.Data.Email))

	if len(entity.Data.Email) == 0 || len(entity.Data.Password) == 0 {
		resp.WriteHeaderAndEntity(400, NewAPIError("Username and password required"))
		return
	}

	key := "sitepod-user-" + GetMD5Hash(entity.Data.Email)

	user, exists := i.client.SitepodUsers().MaybeGetByKey(key)

	if !exists {
		resp.WriteHeaderAndEntity(404, NewAPIError("user not found: "+entity.Data.Email))
		return
	}

	saltedPassword, err := Hash(entity.Data.Password, user.Spec.Salt)

	if err != nil {
		resp.WriteHeaderAndEntity(500, NewAPIError("user not found"))
		return
	}

	if saltedPassword != user.Spec.SaltedPassword {
		resp.WriteHeaderAndEntity(403, NewAPIError("password incorrect"))
		return
	}

	session, err := i.sessionStore.New(req.Request, "sitepodfe")
	lastSeen := time.Now().UTC()
	lastSeen = lastSeen.Round(time.Second)
	sitepodSession := SitepodSession{&lastSeen, session.Options.MaxAge, true}
	session.Values["sess"] = sitepodSession
	session.Save(req.Request, resp.ResponseWriter)
	resp.WriteHeaderAndEntity(200, sitepodSession)
	return
}

func (i *WebApi) WhoAmI(req *restful.Request, resp *restful.Response) {

	session, _ := i.sessionStore.Get(req.Request, "sitepodfe")

	if session.IsNew {
		resp.WriteHeaderAndEntity(200, &SitepodSession{})
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
