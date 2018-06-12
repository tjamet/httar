package main

import (
	"archive/tar"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// https://gist.github.com/maximilien/328c9ac19ab0a158a8df
func addFileToTarWriter(prefix, filePath string, tarWriter *tar.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not open file '%s', got error '%s'", filePath, err.Error()))
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return errors.New(fmt.Sprintf("Could not get stat for file '%s', got error '%s'", filePath, err.Error()))
	}

	header := &tar.Header{
		Name:    strings.TrimPrefix(filePath, prefix),
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not write header for file '%s', got error '%s'", filePath, err.Error()))
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not copy the file '%s' data to the tarball, got error '%s'", filePath, err.Error()))
	}

	return nil
}

func main() {
	var listen string
	var dir string
	var err error
	flag.StringVar(&listen, "listen", ":8080", "The address to listen on")
	flag.StringVar(&dir, "directory", "", "The directory to store files into")
	flag.Parse()
	if dir == "" {
		dir, err = ioutil.TempDir("", "files")
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			fmt.Printf("deleting %s\n", dir)
			err := os.RemoveAll(dir)
			if err != nil {
				fmt.Println(err.Error())
			}
		}()
	}
	os.MkdirAll(dir, 0766)
	fmt.Println(http.ListenAndServe(listen, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.Method == "GET" {
			tarWriter := tar.NewWriter(w)
			defer tarWriter.Close()
			err := filepath.Walk(dir+r.URL.Path, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					return addFileToTarWriter(dir, path, tarWriter)
				}
				return nil
			})
			if err != nil {
				log.Println(err)
			}
		} else if r.Method == "POST" {
			f := path.Join(dir, r.URL.Path)
			d, _ := filepath.Split(f)
			err := os.MkdirAll(d, 0766)
			if err != nil {
				fmt.Println(err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			fd, err := os.Create(f)
			if err != nil {
				fmt.Println(err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer func() {
				err = fd.Close()
				if err != nil {
					log.Println(err.Error())
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			_, err = io.Copy(fd, r.Body)
			if err != nil {
				log.Println(err.Error())
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	})))
}
