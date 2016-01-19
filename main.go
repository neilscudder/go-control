package main

import (
//  "fmt"
  "log"
  "path"
  "strconv"
  "encoding/json"
  "io/ioutil"
  "net/http"
  "html/template"
  "github.com/fhs/gompd/mpd"
  "github.com/nu7hatch/gouuid"
)

func mpdConnect(r *http.Request) *mpd.Client {
  host := r.FormValue("MPDHOST") + ":" + r.FormValue("MPDPORT")
  pass := r.FormValue("MPDPASS")
  conn, ror := mpd.DialAuthenticated("tcp", host, pass); er(ror)
  return conn
}

func mpdNoStatus(r *http.Request) {
  cmd := r.FormValue("a")
  conn := mpdConnect(r)
  defer conn.Close()
  status, ror := conn.Status(); er(ror)
  switch cmd {
    case "fw":
      ror := conn.Next(); er(ror)
    case "up":
      current, ror := strconv.Atoi(status["volume"]); er(ror)
      if current <= 95 {
        new := current + 5
        ror = conn.SetVolume(new); er(ror)
      }
    case "dn":
      current, ror := strconv.Atoi(status["volume"]); er(ror)
      if current >= 5 {
        new := current - 5
        ror = conn.SetVolume(new); er(ror)
      }
    case "random":
      current, ror := strconv.Atoi(status["random"]); er(ror)
      if current == 1 {
        ror = conn.Random(false); er(ror)
      } else {
        ror = conn.Random(true); er(ror)
      }
   }
}

func mpdStatus(w http.ResponseWriter, r *http.Request) {
  conn := mpdConnect(r)
  defer conn.Close()
  status, ror := conn.Status(); er(ror)
  song, ror := conn.CurrentSong(); er(ror)
  t, ror := template.ParseFiles("templates/status.html"); er(ror)
  if status["state"] == "play" && song["Title"] != "" {
    p := map[string]string{
      "title": song["Title"],
      "artist": song["Artist"],
      "album": song["Album"],
    }
    t.Execute(w, p)
  } else if status["state"] == "play" {
    filename := path.Base(song["file"])
    directory := path.Dir(song["file"])
    p := map[string]string{
      "title": filename,
      "artist": song["Artist"],
      "album": directory,
    }
    t.Execute(w, p)
  } else {
    p := map[string]string{
      "title": status["state"],
      "artist": "",
      "album": "",
    }
    t.Execute(w, p)
  }
}

type Params struct {
  GUIURL,
  APIURL,
  APIALT,
  LABEL,
  EMAIL,
  MPDPORT,
  MPDHOST,
  MPDPASS,
  KPASS,
  RPASS string
}

func (p *Params) save() error {
  filename := p.KPASS + ".txt"
  byteP,ror := json.Marshal(p); er(ror)
  return ioutil.WriteFile(filename, byteP, 0600)
}

func loadParams(rpass string) *Params {
  var p *Params
  filename := rpass + ".txt"
  byteP,ror := ioutil.ReadFile(filename); er(ror)
  ror = json.Unmarshal(byteP, &p); er(ror)
  return p
}

func gui(w http.ResponseWriter, r *http.Request) {
  p := map[string]string{
    "APIURL": r.FormValue("APIURL"),
    "APIALT": r.FormValue("APIALT"),
    "MPDPORT": r.FormValue("MPDPORT"),
    "LABEL": r.FormValue("LABEL"),
    "MPDHOST": r.FormValue("MPDHOST"),
    "MPDPASS": r.FormValue("MPDPASS"),
    "KPASS": r.FormValue("KPASS"),
  }
  //var templates = template.Must(template.ParseGlob("templates/gui/*"))
  t, ror := template.ParseGlob("templates/gui/*"); er(ror)
  t.ExecuteTemplate(w, "GUI" ,p)
}

func authority(w http.ResponseWriter, r *http.Request) {
  p := map[string]string{
    "dummy": r.FormValue("dummy"),
  }
  t, ror := template.ParseFiles("templates/authority.html"); er(ror)
  t.Execute(w, p)
}

func authorize(w http.ResponseWriter, r *http.Request) {
  p := &Params{
    GUIURL: r.FormValue("GUIURL"),
    APIURL: r.FormValue("APIURL"),
    APIALT: r.FormValue("APIALT"),
    LABEL: r.FormValue("LABEL"),
    EMAIL: r.FormValue("EMAIL"),
    MPDPORT: r.FormValue("MPDPORT"),
    MPDHOST: r.FormValue("MPDHOST"),
    MPDPASS: r.FormValue("MPDPASS"),
  }
  cURL := p.GUIURL + "/?"
  if p.MPDPASS != "" && p.MPDHOST != "" {
    cURL += "MPDPASS=" + p.MPDPASS + "&MPDHOST=" + p.MPDHOST
  }
  if p.MPDPASS == "" && p.MPDHOST != "" {
    cURL += "&MPDHOST=" + p.MPDHOST
  }
  if p.MPDPORT != "" { cURL += "&MPDPORT=" + p.MPDPORT }
  if p.LABEL != "" { cURL += "&LABEL=" + p.LABEL }
  if p.EMAIL != "" { cURL += "&EMAIL=" + p.EMAIL }
  if p.APIURL != "" { cURL += "&APIURL=" + p.APIURL }
  if p.APIALT != "" { cURL += "&APIALT=" + p.APIALT }
  rURL := cURL
  cURL += "&KPASS="
  rURL += "&RPASS="
  rkey,_ := uuid.NewV4()
  ckey,_ := uuid.NewV4()
  p.KPASS = ckey.String()
  p.RPASS = rkey.String()
  ror := p.save(); er(ror)            // Save to file
  rURL += rkey.String()              // Reset URL
  cURL += ckey.String()              // Control URL
  u := map[string]string{
    "controlURL": cURL,
    "resetURL": rURL,
  }
  t, ror := template.ParseFiles("templates/authorize.html"); er(ror)
  t.Execute(w, u)
}

func get(w http.ResponseWriter, r *http.Request) {
  switch r.FormValue("a"){
    case "info":
      w.Header().Set("Content-Type", "text/html")
      mpdStatus(w,r)
  }
}

func cmd(w http.ResponseWriter, r *http.Request) {
  log.Printf("API Call: " + r.FormValue("a") + " " + r.FormValue("LABEL"))
  mpdNoStatus(r)
}

func er(ror error){
  if ror != nil { log.Fatalln(ror) }
}

func main() {
  http.HandleFunc("/", gui)
  http.HandleFunc("/get", get)
  http.HandleFunc("/cmd", cmd)
  http.HandleFunc("/authority", authority)
  http.HandleFunc("/authorize", authorize)
  http.ListenAndServe(":8080", nil)
}