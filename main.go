package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
)

func readHttpFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("cannot GET from HTTP: %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read from HTTP: %s", err)
	}
	return body, nil
}

func readLocalFile(url string) ([]byte, error) {
	fh, err := os.Open(url)
	if err != nil {
		return nil, fmt.Errorf("cannot open file %s: %s", url, err)
	}
	defer fh.Close()
	body, err := ioutil.ReadAll(fh)
	if err != nil {
		return nil, fmt.Errorf("cannot read file %s: %s", url, err)
	}
	return body, nil
}

func readFile(url string) ([]byte, error) {
	if url[0:7] == "http://" {
		return readHttpFile(url)
	}
	if url[0:7] == "file://" {
		url = url[7:]
	}
	return readLocalFile(url)
}

func load(url string) (interface{}, error) {
	body, err := readFile(url)
	if err != nil {
		return nil, err
	}
	var obj interface{}
	err = json.Unmarshal(body, &obj)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal checks: %s", err)
	}
	return obj, nil
}

func paths(w putter, obj interface{}, path string) error {
	var err error
	if m, ok := obj.(map[string]interface{}); ok {
		for k := range m {
			var p string
			if path == "" {
				p = k
			} else {
				p = path + "." + k
			}
			if err = paths(w, m[k], p); err != nil {
				break
			}
		}
		return err
	}
	if a, ok := obj.([]interface{}); ok {
		for i := range a {
			var p string
			if path == "" {
				p = fmt.Sprintf("%d", i)
			} else {
				p = fmt.Sprintf("%s.%d", path, i)
			}
			if err = paths(w, a[i], p); err != nil {
				break
			}
		}
		return err
	}
	if path != "" {
		return w.put(path, fmt.Sprintf("%v", obj))
	}
	return nil
}

type putter interface {
	put(string, string) error
	flush() error
}

type csvWriter struct {
	w *csv.Writer
}

func (c *csvWriter) put(path, obj string) error {
	return c.w.Write([]string{path, obj})
}

func (c *csvWriter) flush() error {
	c.w.Flush()
	return c.w.Error()
}

type tabWriter struct {
	w *tabwriter.Writer
}

func (t *tabWriter) put(path, obj string) error {
	path = strings.Replace(path, "\t", `\t`, -1)
	obj = strings.Replace(obj, "\t", `\t`, -1)
	_, err := t.w.Write([]byte(path + "\t" + obj + "\n"))
	return err
}

func (t *tabWriter) flush() error {
	return t.w.Flush()
}

func main() {
	outcsv := flag.Bool("csv", false, "Output in CSV")
	flag.Parse()
	arg := flag.Arg(0)
	if arg == "" {
		log.Fatal("please provide a file name or URL")
	}
	obj, err := load(arg)
	if err != nil {
		log.Fatalf("cannot parse JSON: %s")
	}
	var out io.Writer
	out = os.Stdout
	var w putter
	if *outcsv {
		w = &csvWriter{csv.NewWriter(out)}
	} else {
		w = &tabWriter{tabwriter.NewWriter(out, 0, 8, 0, '\t', 0)}
	}
	if err = paths(w, obj, ""); err != nil {
		log.Fatalf("cannot print: %s", err)
	}
	if err = w.flush(); err != nil {
		log.Fatalf("cannot flush: %s", err)
	}
}
