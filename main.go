package main

import (
	"archive/zip"
	"errors"
	"flag"
	"fmt"
	"github.com/fogleman/gg"
	"github.com/golang-module/carbon/v2"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const sourcePath = "./source"
const targetPath = "./target"
const photoDateTimeFormat = "Y/m/d H:i:s"
const fontPath = "./kinetika.ttf"

type Flags struct {
	startDateTime string
	minSecAdd     int
	maxSecAdd     int
}

func main() {
	flags := loadFlags()
	rand.Seed(time.Now().UnixNano())
	carbonDateTime := carbon.Parse(flags.startDateTime)
	removeTargetPhotos()
	sourcePhotos := listDirByReadDir(sourcePath)
	if len(sourcePhotos) < 1 {
		fmt.Println("Not found source files")
		return
	}

	waitGroup := sync.WaitGroup{}
	for _, sourcePhoto := range sourcePhotos {
		waitGroup.Add(1)
		carbonDateTime = carbonDateTime.AddSeconds(rand.Intn(flags.maxSecAdd-flags.minSecAdd) + flags.minSecAdd)
		go handlePhoto(&waitGroup, sourcePhoto, carbonDateTime)
	}

	waitGroup.Wait()
	createZip(sourcePhotos)
}

func loadFlags() Flags {
	flags := Flags{}
	flag.StringVar(&flags.startDateTime, "sdt", "", "Start date time photo (Y-m-d H:i:s)")
	flag.IntVar(&flags.minSecAdd, "min-sec", 5, "Min seconds to add")
	flag.IntVar(&flags.maxSecAdd, "max-sec", 10, "Max seconds to add")
	flag.Parse()

	if flags.startDateTime == "" {
		fmt.Println("Specify start date time photo in format Y-m-d H:i:s")
		os.Exit(1)
	}
	var re = regexp.MustCompile(`(?m)\d\d\d\d-\d\d-\d\d \d\d:\d\d:\d\d`)
	if len(re.FindAllString(flags.startDateTime, -1)) < 1 {
		fmt.Println("Not correct start date time photo. Specify start date time photo in format Y-m-d H:i:s")
		os.Exit(1)
	}

	if flags.minSecAdd > flags.maxSecAdd {
		fmt.Println("minSecAdd less than maxSecAdd")
		os.Exit(1)
	}

	if flags.minSecAdd < 0 || flags.maxSecAdd < 0 {
		fmt.Println("minSecAdd and maxSecAdd must be positive numbers")
		os.Exit(1)
	}

	return flags
}

func handlePhoto(waitGroup *sync.WaitGroup, photo string, carbonDateTime carbon.Carbon) {
	im, err := gg.LoadImage(fmt.Sprintf("%v/%v", sourcePath, photo))
	if err != nil {
		panic(err)
	}
	imgWidth := im.Bounds().Dx()
	imgHeight := im.Bounds().Dy()

	fontSize := float64(imgHeight) * 0.04

	dc := gg.NewContext(imgWidth, imgHeight)

	dc.SetRGB(1, 1, 1)
	if err := dc.LoadFontFace(fontPath, fontSize); err != nil {
		panic(err)
	}
	dc.DrawImage(im, 0, 0)
	dc.DrawString(carbonDateTime.Format(photoDateTimeFormat), float64(imgWidth)*0.7, float64(imgHeight-20))
	dc.Clip()

	words := strings.Split(photo, ".")
	fileName := strings.Join(words[:len(words)-1], ".")
	err = gg.SaveJPG(fmt.Sprintf("%v/%v.jpg", targetPath, fileName), dc.Image(), 100)
	if err != nil {
		panic(err)
	}
	waitGroup.Done()
}

func removeTargetPhotos() {
	targetFiles := listDirByReadDir(targetPath)
	for _, targetFile := range targetFiles {
		err := os.Remove(fmt.Sprintf("%v/%v", targetPath, targetFile))
		if err != nil && errors.Is(err, os.ErrNotExist) == false {
			panic(err)
		}
	}
}

func listDirByReadDir(path string) []string {
	var files []string
	lst, err := ioutil.ReadDir(path)
	if err != nil {
		panic(err)
	}
	for _, val := range lst {
		if val.IsDir() {
			continue
		}
		name := val.Name()
		if name == "" {
			continue
		}
		files = append(files, name)
	}

	return files
}

func createZip(files []string) {
	archive, err := os.Create(fmt.Sprintf("%v/photos.zip", targetPath))
	if err != nil {
		panic(err)
	}
	defer func(archive *os.File) {
		err := archive.Close()
		if err != nil {
			panic(err)
		}
	}(archive)

	zipWriter := zip.NewWriter(archive)

	for _, file := range files {
		words := strings.Split(file, ".")
		fileName := strings.Join(words[:len(words)-1], ".")

		f, err := os.Open(fmt.Sprintf("%v/%v.jpg", targetPath, fileName))
		if err != nil {
			panic(err)
		}

		w, err := zipWriter.Create(fmt.Sprintf("%v.jpg", fileName))
		if err != nil {
			panic(err)
		}
		if _, err := io.Copy(w, f); err != nil {
			panic(err)
		}

		err = f.Close()
		if err != nil {
			panic(err)
		}
	}

	err = zipWriter.Close()
	if err != nil {
		panic(err)
	}

	fmt.Println(fmt.Sprintf("Success! Zip archive available in %v/photos.zip", targetPath))
}
