package main

import (
  "fmt"
  "os"
  "io/ioutil"
  "strings"
//  "strconv"
  "time"
  "regexp"
  "encoding/json"
  "path/filepath"
  "github.com/hpcloud/tail"
  "net/http"
)

type Headers struct {
  Header     string    `json:"header"`
  Content    string    `json:"content"`
}

type Ini struct {
  Logfile    string    `json:"logfile"`
  Pattern    string    `json:"pattern"`
  DateFmt    string    `json:"dateFormat"`
  AddTZ      string    `json:"addTZ"`
  JsonFrmt   string    `json:"jsonFrmt"`
  JsonFlfs   string    `json:"jsonFlds"`
  ClpURL     string    `json:"clpURL"` 
  ClpHeaders []Headers `json:"clpHeaders"`
}

func getIniJson() Ini {
  //var result map[string]interface{}

  var ini Ini
  
  // --- find ini file ---
  file, _ := os.Readlink("/proc/self/exe")
  
  // --- read json ini file ---
  jsonFile, err := os.Open(filepath.Join(filepath.Dir(file), "logCollect.json"))
  if err != nil {
    panic(err.Error())
  }
  defer jsonFile.Close()
  
  content, _ := ioutil.ReadAll(jsonFile)
  err = json.Unmarshal(content, &ini)
  if err != nil {
    panic(err.Error())
  }
  return ini
}

func main() {
  conf := getIniJson()

  // --- definitions & declarations ---
  const timeOut  = "2006-01-02T15:04:05.999Z07:00"
  host,_        := os.Hostname()
  myExp         := regexp.MustCompile(conf.Pattern)
  result        := make(map[string]string)
  result["host"] = host
  locTZ, _      := time.Now().Local().Zone()
  fldsArr       := strings.Split(conf.JsonFlfs, ",")
  jsonArr       := make([]interface{}, len(fldsArr))

  //--- open log file ---
  t, err := tail.TailFile(conf.Logfile, tail.Config{
    Follow: true,
    ReOpen: true})
  if err != nil {
    panic(err.Error())
  }

  // --- loop log file ---
  for line := range t.Lines {
    
    // -->
      fmt.Printf("%s\n", line.Text)
    // <--
    
    // --- parse log entry ---
    match := myExp.FindStringSubmatch(line.Text)
    for i, name := range myExp.SubexpNames() {
      if i != 0 && name != "" {
        result[name] = match[i]
      }
    }

    // --- convert time ---    
    if conf.AddTZ == "true" {
      tStamp := fmt.Sprintf("%s %s", result["timestamp"], locTZ)
      result["timestamp"] = tStamp
    }
    dt, _ := time.Parse(conf.DateFmt, result["timestamp"])
    result["timestamp"] = dt.In(time.Local).Format(timeOut)

    // --- build sprinf array ---/
    for idx, val := range fldsArr {
      //fmt.Printf("%d => %s = %s\n", idx, val, result[val])
      jsonArr[idx] = result[val]
    }
    
    // --- build JSON output ---
    jsonOut := fmt.Sprintf(conf.JsonFrmt, jsonArr...)
    
    // -->
      fmt.Printf("%s\n", jsonOut)
    // <--
    
    // --- setd request to CLP ---
    body := strings.NewReader(jsonOut)
    req, err := http.NewRequest("POST", conf.ClpURL, body)
    if err != nil {
      panic(err.Error())
    }
    
    for _, val := range conf.ClpHeaders {
      req.Header.Set(val.Header, val.Content)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
      panic(err.Error())
    }

    // -->
      fmt.Println(resp)
return
    // <--
    
    defer resp.Body.Close()    
  }
}
