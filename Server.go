package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)


var (
	host      = flag.String("localhost", "10.179.3.49", "127.0.0.1")
	validFrom = flag.String("Jan 1 15:04:05 2014", "", "Creation date formatted as Jan 1 15:04:05 2011")
	validFor  = flag.Duration("43800", 365*24*time.Hour, "Duration that certificate is valid for")
	isCA      = flag.Bool("ca", true, "whether this cert should be its own Certificate Authority")
	rsaBits   = flag.Int("rsa-bits", 2048, "Size of RSA key to generate")
   files = make(map[string]string)
   downloads = make(map[string]string)
)

func genCert() {
	flag.Parse()

	if len(*host) == 0 {
		log.Fatalf("Missing required --host parameter")
	}

	priv, err := rsa.GenerateKey(rand.Reader, *rsaBits)
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	var notBefore time.Time
	if len(*validFrom) == 0 {
		notBefore = time.Now()
	} else {
		notBefore, err = time.Parse("Jan 2 15:04:05 2006", *validFrom)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse creation date: %s\n", err)
			os.Exit(1)
		}
	}

	notAfter := notBefore.Add(*validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(*host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if *isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	certOut, err := os.Create("cert.pem")
	if err != nil {
		log.Fatalf("failed to open cert.pem for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()
	log.Print("written cert.pem\n")

	keyOut, err := os.OpenFile("key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Print("failed to open key.pem for writing:", err)
		return
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()
	log.Print("written key.pem\n")
}

func uploadFunc(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method) //get request method
	if r.Method == "GET" {
		htmlString := "<html><head><title>FServer</title></head><body><form action='/upload' method='post'><label>File URL</label><input type='text' name='URL'/><br/><input type='submit' value='Upload'/></form><br/><a href='/files'>files</a></body></html>"
		fmt.Fprintf(w, htmlString)

	} else {
		r.ParseForm()
		// logic part of log in
		url := strings.Join(r.Form["URL"], "")
		url=strings.Replace(url, " ", "%20", -1)
		fmt.Println("URL:", url)

		datecom := "date '+%Y%m%d-%H%M%S'"
		date1, _ := exec.Command("sh", "-c", datecom).Output()
		date0 := string(date1[:])
		date0 = strings.Replace(date0, "\n", "", -1)

		files[date0] = url
		downloads[url] = "pending"
		s := []string{"/usr/bin/wget -O ", "public/", date0, " ", url}
		command := strings.Join(s, "")

		_, err := exec.Command("sh", "-c", command).Output()
		if err != nil {

			log.Fatal(err)
		}
		fmt.Printf("Command Executed")
		downloads[url] = "completed"
		link:= "Submitted <a href='/status'>status</a><br/> <a href='/files'>files</a>"
		fmt.Fprintf(w,link)
	}
	return
}
func statusDownloads(w http.ResponseWriter, r *http.Request) {
	out := []string{}
	for k, v := range files {
		out = append(out, k)
		out = append(out, "|")
		out = append(out, v)
		out = append(out, "|")
		out = append(out, downloads[v])
		out = append(out, "\n")
	}
	output := strings.Join(out, " ")
	fmt.Fprintf(w, output)
	return
}


func main() {
	genCert()
	fpath, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	fpath += "/public"



	http.HandleFunc("/", uploadFunc)
	http.HandleFunc("/status", statusDownloads)
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(fpath))))
	fmt.Println("Serving on PORT 10443 and saving to:",fpath)
	panic(http.ListenAndServeTLS(":10443", "cert.pem", "key.pem", nil))

}

func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	if status == http.StatusNotFound {
		fmt.Fprint(w, r.URL.Path)
	}
}
