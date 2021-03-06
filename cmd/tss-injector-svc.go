package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/DelineaXPM/tss-k8s/v2/pkg/injector"

	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const rolesFile = "roles.json"

func main() {
	var hostPort, certFile, keyFile, rolesFile string

	flag.StringVar(&hostPort, "hostport", ":18543", "the host:port e.g. localhost:8080")
	flag.StringVar(&certFile, "cert", "injector.pem", "the path of the certificate file in PEM format")
	flag.StringVar(&keyFile, "key", "injector.key", "the path of the certificate key file in PEM format")
	flag.StringVar(&rolesFile, "roles", "roles.json", "the path of JSON formatted roles file")
	flag.Parse()

	rcj, err := ioutil.ReadFile(rolesFile)

	if err != nil {
		log.Fatalf("unable to open configuration file '%s': %s", rolesFile, err)
	}

	roles := new(injector.Roles)

	if err := json.Unmarshal(rcj, roles); err != nil {
		log.Fatalf("unable to parse configuration file '%s': %s", rolesFile, err)
	}

	roleNames := make([]string, 0, len(*roles)) // for logging

	for name := range *roles {
		roleNames = append(roleNames, name)
	}
	log.Printf("[INFO] configured role(s): [%s]", strings.Join(roleNames, ", "))

	http.HandleFunc("/inject", func(w http.ResponseWriter, r *http.Request) {
		if request, err := ioutil.ReadAll(r.Body); err == nil {
			defer r.Body.Close()
			log.Printf("[DEBUG] the request body is %d bytes", len(request))

			ar := new(v1.AdmissionReview)

			if err := json.Unmarshal(request, ar); err == nil {
				var response []byte

				if err := injector.Inject(ar, *roles); err == nil {
					if response, err = json.Marshal(ar); err == nil {
						w.WriteHeader(http.StatusOK)
						w.Write(response)
					} else {
						log.Printf("[DEBUG] unable to re-marshal AdmissionReview: %s", err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
				} else {
					log.Printf("[DEBUG] injector.Inject error: %s", err)
					ar.Response = &v1.AdmissionResponse{
						Result: &metav1.Status{Message: err.Error()},
					}
					response, _ := json.Marshal(ar)

					w.WriteHeader(http.StatusOK)
					w.Write(response)
				}
				log.Printf("[DEBUG] sent a %d byte response", len(response))
			} else {
				log.Printf("[DEBUG] unable to unmarshal AdmissionReview: %s", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	})
	log.Printf("[INFO] listening for HTTPS requests on host:port '%s'", hostPort)
	log.Fatal(http.ListenAndServeTLS(hostPort, certFile, keyFile, nil))
}
