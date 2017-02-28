package main

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"github.com/jbangert/hottub/controller"
	"golang.org/x/crypto/acme/autocert"
	"net/http"

	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
)

var CSRF [32]byte

func init() {
	store := sessions.NewFilesystemStore(os.TempDir(), []byte("goth-example"))

	// set the maxLength of the cookies stored on the disk to a larger number to prevent issues with:
	// securecookie: the value is too long
	// when using OpenID Connect , since this can contain a large amount of extra information in the id_token

	// Note, when using the FilesystemStore only the session.ID is written to a browser cookie, so this is explicit for the storage on disk
	store.MaxLength(math.MaxInt64)

	gothic.Store = store

	_, err := rand.Read(CSRF)
	if err != nil {
		panic(err)
	}
}


func main() {
	goth.UseProviders(
		github.New(os.Getenv("GITHUB_KEY"), os.Getenv("GITHUB_SECRET"), "https://hottub.ninja/auth/github/callback"))

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("hottub.ninja"), //your domain here
		Cache:      autocert.DirCache("/home/pi/ssl"),      //folder for storing certificates
	}
	hottub := controller.Hottub{}
	hottub.Start()

	p := pat.New()
	p.Get("/auth/{provider}/callback", func(res http.ResponseWriter, req *http.Request) {

		user, err := gothic.CompleteUserAuth(res, req)
		if err != nil {
			fmt.Fprintln(res, err)
			return
		}
		t, _ := template.New("foo").Parse(userTemplate)
		t.Execute(res, user)
	})

	p.Get("/auth/{provider}", func(res http.ResponseWriter, req *http.Request) {
		// try to get the user without re-authenticating
		if gothUser, err := gothic.CompleteUserAuth(res, req); err == nil {
			t, _ := template.New("foo").Parse(userTemplate)
			t.Execute(res, gothUser)
		} else {
			gothic.BeginAuthHandler(res, req)
		}
	})

	p.Get("/", func(res http.ResponseWriter, req *http.Request) {
		indexTemplate.Execute(res, map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(req),
		})
	})

	p.Post("/", func (res http.ResponseWriter, req *http.Request) {
		user, err := gothic.CompleteUserAuth(res, req)
		if err != nil || !authenticated(user) {
			res.WriteHeader(http.StatusForbidden)
			res.Write("Error authenticating")			
			log.Printf("Error  authenticating user %v", err)
			return
		}
		err = req.ParseForm()
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			res.Write("Cannot parse request")	
			log.Printf("Cannot parse the request %v", err)
			return
		}
		if  req.FormValue["status"]  == "On" {
			hottub.SetTargetTemp(40)
		} else {
			hottub.SetTargetTemp(-20)
		}
		http.Redirect(res, req, "/", 303)
	})

	server := &http.Server{
		Addr: ":443",
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	server.ListenAndServeTLS("", "", csrf.Protect(CSRF)(p)) //key and cert are comming from Let's Encrypt
}

type IndexData struct {
	InletTemp float64
	OutletTemp float64
	Status string
	User *goth.User
}

func authenticated (user *goth.User) bool {
	if user.Email == "nathan@nixpulvis.com" || user.Email == "jbangert@acm.org" {
		return true
	}
	return false
}

var templateFuncs = template.FuncMap{"authenticated", authenticated}
var indexTemplate = template.Must(template.New("Index").FuncMap(templateFuncs).Parse(`
<html>
<head><title>Hottub</title></head>
<body>
<p> Inlet {{.InletTemp}}</p>
<p> Outlet {{.OutletTemp}}</p>
<p> Status {{.Status}}</p>
{{if authenticated User}}
<form><input method="POST" action="/">
    <input type="submit" name="status" value="On">
    <input type="submit" name="status" value="Off">
</form>
{{end}}
</body>
</html>`)
