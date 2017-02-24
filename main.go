package main

import (
	"crypto/tls"
	"github.com/jbangert/hottub/controller"
    "golang.org/x/crypto/acme/autocert"
    "net/http"
)


func main() {
    certManager := autocert.Manager{
        Prompt:     autocert.AcceptTOS,
        HostPolicy: autocert.HostWhitelist("hottub.ninja"), //your domain here
        Cache:      autocert.DirCache("/home/pi/ssl"), //folder for storing certificates
    }
    hottub := controller.Hottub{}
    go hottub.Run()	
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	    w.Write([]byte(fmt.Formatf("Hottub: %v Celsius, Heater: %v Celsius", hottub.GetInletTemp(), hottub.GetOutletTemp())))
    })

    server := &http.Server{
        Addr: ":443",
        TLSConfig: &tls.Config{
            GetCertificate: certManager.GetCertificate,
        },
    }

    server.ListenAndServeTLS("", "") //key and cert are comming from Let's Encrypt
}
