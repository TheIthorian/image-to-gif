// main.go
package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"

	"mime/multipart"

	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
)

var templates = template.Must(template.ParseFiles("upload.html", "result.html"))

var palette = []color.Color{
	color.RGBA{0x00, 0x00, 0x00, 0xff}, color.RGBA{0x00, 0x00, 0xff, 0xff},
	color.RGBA{0x00, 0xff, 0x00, 0xff}, color.RGBA{0x00, 0xff, 0xff, 0xff},
	color.RGBA{0xff, 0x00, 0x00, 0xff}, color.RGBA{0xff, 0x00, 0xff, 0xff},
	color.RGBA{0xff, 0xff, 0x00, 0xff}, color.RGBA{0xff, 0xff, 0xff, 0xff},
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	log.Println("rendering template: " + tmpl)

	err := templates.ExecuteTemplate(w, tmpl+".html", data)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("handling upload")

	if r.Method != http.MethodPost {
		renderTemplate(w, "upload", nil)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	files := r.MultipartForm.File["files"]
	fps, _ := strconv.ParseUint(r.MultipartForm.Value["fps"][0], 10, 64)
	size, _ := strconv.ParseUint(r.MultipartForm.Value["size"][0], 10, 64)

	gifPath, err := createGIF(files, int(fps), int(size))
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/result?gif="+gifPath, http.StatusSeeOther)
	return

}

func createGIF(files []*multipart.FileHeader, fps int, size int) (string, error) {
	log.Println("creating gif from " + fmt.Sprint(len(files)) + " files")

	var images = make([]*image.Paletted, len(files))

	tempDir, err := os.MkdirTemp("", "uploaded-images")
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer os.RemoveAll(tempDir)

	ch := make(chan *image.Paletted, len(files))
	for i, file := range files {
		fmt.Println("Processing file: " + fmt.Sprint(i))
		go processImageAndAddToArray(file, fps, size, tempDir, ch)
	}

	fmt.Println("Waiting for channels")
	for i := range files {
		fmt.Println("Reading from channel: " + fmt.Sprint(i))
		image := <-ch
		images[i] = image
	}

	gifPath := "output.gif"
	gifFile, err := os.Create(gifPath)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer gifFile.Close()

	var delays = make([]int, len(files))
	for i := range delays {
		delays[i] = fps
	}

	err = gif.EncodeAll(gifFile, &gif.GIF{
		Image: images,
		Delay: delays,
	})
	if err != nil {
		log.Println(err)
		return "", err
	}

	log.Println("\tencoded gif")
	return gifPath, nil
}

func processImageAndAddToArray(
	file *multipart.FileHeader,
	fps int,
	size int,
	tempDir string,
	ch chan *image.Paletted,
) {
	src, err := file.Open()
	if err != nil {
		log.Println(err)
		return
	}
	defer src.Close()

	// Save the file to the temporary directory
	tempFile, err := os.CreateTemp(tempDir, "uploaded-image-*.jpg")
	if err != nil {
		log.Println(err)
		return
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, src)
	if err != nil {
		log.Println(err)
		return
	}

	// Rewind the temporary file before processing
	tempFile.Seek(0, io.SeekStart)

	var img image.Image
	_, format, err := image.DecodeConfig(tempFile)
	if err != nil {
		return
	}

	// Rewind the temporary file before decoding
	tempFile.Seek(0, io.SeekStart)

	switch format {
	case "jpeg":
		img, err = jpeg.Decode(tempFile)
		if err != nil {
			return
		}
	case "png":
		img, err = png.Decode(tempFile)
		if err != nil {
			return
		}
	default:
		return
	}

	// Resize, crop, and process the image as before
	resizedImg := resize.Resize(uint(size), 0, img, resize.Bicubic)
	croppedImg := resize.Resize(uint(size), uint(size), resizedImg, resize.Bicubic)

	palettedImg := image.NewPaletted(croppedImg.Bounds(), palette)
	draw.FloydSteinberg.Draw(palettedImg, croppedImg.Bounds(), croppedImg, image.Point{})

	// images = append(images, palettedImg)
	ch <- palettedImg
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	gifPath := r.URL.Query().Get("gif")
	renderTemplate(w, "result", gifPath)
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		return
	}

	vars := mux.Vars(r)

	// Handle the request for the GIF file
	imagePath := vars["image"]
	log.Println("Trying to serve image " + imagePath)
	http.ServeFile(w, r, imagePath)
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/upload", uploadHandler)
	r.HandleFunc("/result", resultHandler)
	r.HandleFunc("/images/{image}", imageHandler).Methods("GET")
	http.Handle("/", r)

	fmt.Println("Listening on port 8080: http://localhost:8080/")

	log.Fatal(http.ListenAndServe(":8080", nil))
	fmt.Println("Done :)")
}
