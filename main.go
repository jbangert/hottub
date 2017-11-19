package main

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"github.com/gorilla/csrf"
	"github.com/gorilla/context"
	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"github.com/jbangert/hottub/controller"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"golang.org/x/crypto/acme/autocert"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"runtime"
)


type IndexData struct {
	TargetTemp    float64
	InletTemp     float64
	OutletTemp    float64
	Status        string
	Username      string
	Authenticated bool
	CSRFTag       template.HTML
}

func randKey() []byte {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	return key
}

var	store = sessions.NewCookieStore(randKey(), randKey())
func init() {
	runtime.GOMAXPROCS(2) 
	// set the maxLength of the cookies stored on the disk to a larger number to prevent issues with:
	// securecookie: the value is too long
	// when using OpenID Connect , since this can contain a large amount of extra information in the id_token

	gothic.Store = store
}

func main() {
	goth.UseProviders(
		github.New(os.Getenv("GITHUB_KEY"), os.Getenv("GITHUB_SECRET"), "https://hottub.ninja/auth/github/callback", "user:email"))

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
			res.WriteHeader(http.StatusForbidden)
			fmt.Fprintln(res, "Error authenticating")
			log.Printf("Error  authenticating user %v", err)
			return
		}
		session, _ := store.Get(req, "hottub")
		session.Values["Email"] = user.Email
		session.Save(req, res)
		log.Printf("Logged in %v", user)
		http.Redirect(res, req, "/", 302)
	})

	p.Get("/auth/{provider}", func(res http.ResponseWriter, req *http.Request) {
		// try to get the user without re-authenticating
		if _, err := gothic.CompleteUserAuth(res, req); err == nil {
			http.Redirect(res, req, "/", 302)
		} else {
			gothic.BeginAuthHandler(res, req)
		}
	})

	p.Get("/", func(res http.ResponseWriter, req *http.Request) {
		indexData := IndexData{
			InletTemp:  hottub.GetInletTemp(),
			OutletTemp: hottub.GetOutletTemp(),
			TargetTemp: hottub.GetTargetTemp(),
			Status:     hottub.GetStatus(),
			CSRFTag:    csrf.TemplateField(req),
		}
		session, _ := store.Get(req, "hottub")
		email := session.Values["Email"]
		if email != nil { 
			indexData.Username = email.(string)
			indexData.Authenticated = authenticated(indexData.Username) 
		}
		indexTemplate.Execute(res, indexData)
	})

	p.Post("/", func(res http.ResponseWriter, req *http.Request) {
		session, _ := store.Get(req, "hottub")
		user := session.Values["Email"]
		if user == nil && !authenticated(user.(string)) {
			res.WriteHeader(http.StatusForbidden)
			fmt.Fprintln(res, "Error authenticating")
			return
		}
		err := req.ParseForm()
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(res, "Cannot parse request")
			log.Printf("Cannot parse the request %v", err)
			return
		}
		target, err := strconv.ParseFloat(req.FormValue("temperature"), 64)
		if err == nil && target < 41 {
			hottub.SetTargetTemp(target)
		} else {
			res.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(res, "Error %v with handling temperature %v", err, target)
			return
		}
		http.Redirect(res, req, "/", 303)
	})

	server := &http.Server{
		Addr: ":443",
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
		Handler: csrf.Protect(randKey())(context.ClearHandler(p)),
	}

	server.ListenAndServeTLS("", "") //key and cert are comming from Let's Encrypt
}

func authenticated(user string) bool {
	if user == "karshan.sharma@gmail.com" || user == "julian."+"o.b." + "bangert" + "@dartmouth.edu" {
		return true
	}
	return false
}

var indexTemplate = template.Must(template.New("Index").Parse(`
<html>
<head><title>Hottub</title></head>
<body>
<p> Hello {{.Username}}</p>
<p> Inlet {{.InletTemp}}C</p>
<p> Target {{.TargetTemp}}C</p>
<p> Outlet {{.OutletTemp}}C</p>
<p> Status {{.Status}}</p>

{{if .Authenticated}}
<form method="POST" action="/">
    <input type="submit" name="temperature" value="40">
    <input type="submit" name="temperature" value="39">
    <input type="submit" name="temperature" value="38">
    <input type="submit" name="temperature" value="37">
    <input type="submit" name="temperature" value="-256">
    {{.CSRFTag}}
</form>
{{else}}
<p><a href="/auth/github/">Login</a></p>
{{end}}
</body>
</html>`))
