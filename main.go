package main

import (
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

var (
	wg         sync.WaitGroup
	configfile string
	config     []*RSS
)

type RSS struct {
	RSSURL        string
	SleepTime     time.Duration
	LastCheckTime time.Time
	Links         map[string]struct{}
	Cmd           string
	Paths         map[string]string
	NeedSave      bool `json:"-"`
}
type Channel struct {
	Items []Item `xml:"channel>item"`
}
type Item struct {
	Title     string    `xml:"title"`
	Link      string    `xml:"link"`
	Category  string    `xml:"category"`
	Enclosure Enclosure `xml:"enclosure"`
}
type Enclosure struct {
	URL string `xml:"url,attr"`
}

func (r *RSS) Download(url, path string) {
	var cmd *exec.Cmd
	if path == "" {
		cmd = exec.Command(r.Cmd, "-a", url)
	} else {
		cmd = exec.Command(r.Cmd, "-a", url, "-w", path)
	}
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}
func (r *RSS) Update() {
	ht := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	resp, err := ht.Get(r.RSSURL)
	if err == nil {
		buf, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			t := decxml(buf)
			for i := range t.Items {
				if r.Links == nil {
					r.Links = make(map[string]struct{})
				}
				if _, ok := r.Links[t.Items[i].Link]; !ok {
					var path string
					if r.Paths != nil {
						if v, ok := r.Paths[t.Items[i].Category]; ok {
							path = v
						} else {
							path = r.Paths["default"]
						}
					}
					r.Download(t.Items[i].Enclosure.URL, path)
					r.Links[t.Items[i].Link] = struct{}{}

					r.NeedSave = true
				}
			}
		} else {
			fmt.Println(err)

		}
		resp.Body.Close()
	} else {
		fmt.Println(err)
	}
}
func (r *RSS) AutoCheck() {
	defer wg.Done()
	for {
		fmt.Println("开始检查", r.RSSURL)
		r.Update()

		r.LastCheckTime = time.Now()
		time.Sleep(r.SleepTime * time.Minute)
		if r.NeedSave {
			r.NeedSave = false
			Save()
		}

	}
}
func Save() {
	b, _ := json.Marshal(config)
	ioutil.WriteFile(configfile, b, 0666)
}
func main() {
	if len(os.Args) < 2 {
		fmt.Println(os.Args[0], "config.json")
		return
	}
	configfile = os.Args[1]
	b, err := ioutil.ReadFile(configfile)
	if err == nil {
		json.Unmarshal(b, &config)
		for i := range config {
			wg.Add(1)
			go config[i].AutoCheck()
		}
		wg.Wait()
	}
	fmt.Println(err)

}
func decxml(in []byte) Channel {
	var out Channel
	xml.Unmarshal(in, &out)
	return out

}
