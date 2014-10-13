package main

import (
	"flag"
	"fmt"
	"libfastget"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

)

var (
	files     = make(map[string]string)
	downloads = make(map[string]string)
	portPtr   = flag.String("port", "10443", "a valid port number")
)

func deleteFile(w http.ResponseWriter, r *http.Request) {

	fpath, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	fpath += "/public/"
	r.ParseForm()
	fname := strings.Join(r.Form["fid"], "")

	fpath += fname
	fmt.Println("Deleteting :", fpath)

	for k := range downloads {
		if k == files[fname] {
			fmt.Println("Val found")
			delete(downloads, k)

		}

	}
	delete(files, fname)
	err := os.Remove(fpath)

	if err != nil {
		fmt.Fprintf(w, "Error occured")
	} else {
		fmt.Fprintf(w, "file successfully deleted")
	}

}

func uploadFunc(w http.ResponseWriter, r *http.Request) {
	//fmt.Println(r.Method," request from ",r.RemoteAddr) //get request method
	if r.Method == "GET" {
		htmlString := "<html><head><title>FServer</title></head><body><form action='/upload' method='post'><label>File URL: </label><input type='text' name='URL'/><br/><input type='submit' value='Upload'/></form><a href='/status'>status</a></body></html>"
		fmt.Fprintf(w, htmlString)

	} else {
		r.ParseForm()
		// logic part of log in
		urlArray := []string{strings.Join(r.Form["URL"], "")}
		url := strings.Join(urlArray, "")
		url = strings.Replace(url, " ", "%20", -1)
		fmt.Println("URL:", url)
		if _, ok := downloads[url]; ok {

		} else {

			date0 := time.Now().Format("20060102-150405")

			files[date0] = url
			downloads[url] = "pending"

			connCount := 8
			filename := "public/" + date0
			_, err := libfastget.FastGet(url, connCount, filename)
			if err != nil {
				fmt.Println(err)
				downloads[url] = "error"
				return
			} else {
				fmt.Printf("Command Executed")
				downloads[url] = "completed"

				return
			}

			//		go execWget(command, url)
		}
		http.Redirect(w, r, "/status", http.StatusMovedPermanently)

	}

}

func statusDownloads(w http.ResponseWriter, r *http.Request) {
	out := []string{"<html>"}
	for k, v := range files {
		out = append(out, k)
		out = append(out, "|")
		out = append(out, v)
		out = append(out, "|")
		out = append(out, downloads[v])
		out = append(out, "|")
		out = append(out, "<a href=/files/"+k+">Get</a>")
		out = append(out, "|")
		out = append(out, "<a href=/delete?fid="+k+">Delete</a>")
		out = append(out, "<br/>")
	}
	out = append(out, "<a href='/upload'>upload</a>&nbsp;</html>")
	output := strings.Join(out, " ")
	fmt.Fprintf(w, output)
	return
}

func main() {
   gen_cert()
   //easy_cert()
	err := os.Mkdir("public", 0777)

	if err != nil {
		fmt.Println("Could not create directory")

	}
	fpath, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	fpath += "/public"

	http.HandleFunc("/", AuthHandler)
	http.HandleFunc("/status", statusDownloads)
	http.HandleFunc("/delete", deleteFile)
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(fpath))))
	port := ":"

	flag.Parse()
	port += *portPtr

	fmt.Printf("Serving on PORT %s and saving to: %s\n", port, fpath)
	if err := http.ListenAndServeTLS(port, "cert.pem", "key.pem", nil); err != nil {
		http.ListenAndServeTLS(":10443", "cert.pem", "key.pem", nil)
	}

}
