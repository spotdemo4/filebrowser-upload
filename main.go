package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/viper"
)

func main() {
	// Remove date and time from log
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	// Get config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalln(err)
	}

	filebrowser := filebrowser{
		Url:       "",
		Username:  "",
		Password:  "",
		File:      "",
		Override:  false,
		Share:     false,
		Directory: "",
		configDir: filepath.Join(configDir, "filebrowser-upload"),
	}

	// Initialize filebrowser configuration directory
	if err := filebrowser.init(); err != nil {
		log.Fatalln(err)
	}

	// Read in environment variables
	if err := filebrowser.read(); err != nil {
		log.Fatalln(err)
	}

	// Set command line flags for set
	setCmd := flag.NewFlagSet("set", flag.ExitOnError)
	setCmd.StringVar(&filebrowser.Url, "url", filebrowser.Url, "URL of the Filebrowser instance")
	setCmd.StringVar(&filebrowser.Username, "username", filebrowser.Username, "Username of the Filebrowser instance")
	setCmd.StringVar(&filebrowser.Password, "password", filebrowser.Password, "Password of the Filebrowser instance")
	setCmd.BoolVar(&filebrowser.Override, "override", filebrowser.Override, "Override file if it exists")
	setCmd.BoolVar(&filebrowser.Share, "share", filebrowser.Share, "Share file after uploading")
	setCmd.StringVar(&filebrowser.Directory, "directory", filebrowser.Directory, "Directory to upload file to")

	// Set command line flags for upload
	uploadCmd := flag.NewFlagSet("upload", flag.ExitOnError)
	uploadCmd.StringVar(&filebrowser.Url, "url", filebrowser.Url, "URL of the Filebrowser instance")
	uploadCmd.StringVar(&filebrowser.Username, "username", filebrowser.Username, "Username of the Filebrowser instance")
	uploadCmd.StringVar(&filebrowser.Password, "password", filebrowser.Password, "Password of the Filebrowser instance")
	uploadCmd.BoolVar(&filebrowser.Override, "override", filebrowser.Override, "Override file if it exists")
	uploadCmd.BoolVar(&filebrowser.Share, "share", filebrowser.Share, "Share file after uploading")
	uploadCmd.StringVar(&filebrowser.Directory, "directory", filebrowser.Directory, "Directory to upload file to")
	uploadCmd.StringVar(&filebrowser.File, "file", "", "File to upload")

	// Make sure directory starts with a slash
	if filebrowser.Directory != "" && filebrowser.Directory[0] != '/' {
		filebrowser.Directory = "/" + filebrowser.Directory
	}

	// Make sure url doesn't end with a slash
	if filebrowser.Url != "" && (filebrowser.Url)[len(filebrowser.Url)-1] == '/' {
		filebrowser.Url = (filebrowser.Url)[:len(filebrowser.Url)-1]
	}

	// Check if command line argument is empty
	if len(os.Args) < 2 {
		log.Fatalln("No command provided. Usage: filebrowser-upload <set, upload> [flags]")
	}

	// Get command line command
	switch os.Args[1] {

	case "set":
		setCmd.Parse(os.Args[2:])

		if err := filebrowser.set(); err != nil {
			log.Fatalln(err)
		}

	case "upload":
		uploadCmd.Parse(os.Args[2:])

		if filebrowser.File == "" {
			log.Fatalln("No file provided. Usage: filebrowser-upload upload -file <file>")
		}

		// Get token
		token, err := filebrowser.token()
		if err != nil {
			log.Fatalln(err)
		}

		// Upload file
		if err := filebrowser.upload(token); err != nil {
			fmt.Println()
			log.Fatalln(err)
		}

		if filebrowser.Share {
			// Share file
			shareResp, err := filebrowser.share(token)
			if err != nil {
				log.Fatalln(err)
			}

			fmt.Println("Share:", filebrowser.Url+"/share/"+shareResp.Hash)
			fmt.Println("Download:", filebrowser.Url+"/api/public/dl/"+shareResp.Hash)
			fmt.Println("Inline:", filebrowser.Url+"/api/public/dl/"+shareResp.Hash+"?inline=true")
		}

	default:
		log.Fatalln("Invalid command. Usage: filebrowser-upload <set, upload> [flags]")
	}
}

type filebrowser struct {
	Url       string `mapstructure:"URL"`
	Username  string `mapstructure:"USERNAME"`
	Password  string `mapstructure:"PASSWORD"`
	File      string
	Override  bool   `mapstructure:"OVERRIDE"`
	Share     bool   `mapstructure:"SHARE"`
	Directory string `mapstructure:"DIRECTORY"`
	configDir string
}

func (e *filebrowser) init() error {
	// Check if config directory exists
	if _, err := os.Stat(e.configDir); os.IsNotExist(err) {
		// Create config directory
		err := os.Mkdir(e.configDir, 0755)
		if err != nil {
			return errors.New("could not create config directory")
		}
	}

	// Check if env file exists
	if _, err := os.Stat(filepath.Join(e.configDir, "config.env")); os.IsNotExist(err) {
		// Create env file
		file, err := os.Create(filepath.Join(e.configDir, "config.env"))
		if err != nil {
			return errors.New("could not create config.env file")
		}
		defer file.Close()
	}

	return nil
}

func (e *filebrowser) read() error {
	// Read contents of the env file into env struct
	viper.SetConfigName("config.env")
	viper.AddConfigPath(e.configDir)
	viper.SetConfigType("env")

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	if err := viper.Unmarshal(&e); err != nil {
		return err
	}

	return nil
}

func (e *filebrowser) set() error {
	// Write to env file
	viper.SetConfigName("config.env")
	viper.AddConfigPath(e.configDir)
	viper.SetConfigType("env")

	if e.Url != "" {
		viper.Set("URL", e.Url)
	}
	if e.Username != "" {
		viper.Set("USERNAME", e.Username)
	}
	if e.Password != "" {
		viper.Set("PASSWORD", e.Password)
	}
	viper.Set("OVERRIDE", e.Override)
	viper.Set("SHARE", e.Share)
	viper.Set("DIRECTORY", e.Directory)

	if err := viper.WriteConfig(); err != nil {
		return err
	}

	return nil
}

func (e *filebrowser) token() (string, error) {
	if e.Username == "" || e.Password == "" || e.Url == "" {
		return "", errors.New("username, password, or URL not provided")
	}

	// Get token
	jsonBody := []byte(`{"username": "` + e.Username + `", "password": "` + e.Password + `", "recaptcha": ""}`)
	resp, err := http.Post(e.Url+"/api/login", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check if status code is 200
	if resp.StatusCode != 200 {
		return "", errors.New("could not get token")
	}

	// Get token
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (e *filebrowser) upload(token string) error {
	if e.Url == "" || e.File == "" {
		return errors.New("URL or file not provided")
	}

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	// Get file
	file, err := os.Open(filepath.Join(wd, e.File))
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	// Create new post request
	postReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/tus%s/%s?override=%t", e.Url, e.Directory, fileInfo.Name(), e.Override), nil)
	if err != nil {
		return err
	}

	// Set post headers
	postReq.Header.Set("X-Auth", token)
	postReq.Header.Set("Content-Type", "application/json")

	// Send post request
	postResp, err := http.DefaultClient.Do(postReq)
	if err != nil {
		return err
	}
	defer postResp.Body.Close()

	// Check if status code is 201
	if postResp.StatusCode != 201 {
		return errors.New("could not create new file")
	}

	// Create progress bar
	bar := progressbar.DefaultBytes(
		fileInfo.Size(),
		"Uploading",
	)
	barReader := progressbar.NewReader(file, bar)

	// Create new patch request
	patchReq, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/api/tus%s/%s?override=%t", e.Url, e.Directory, fileInfo.Name(), e.Override), &barReader)
	if err != nil {
		return err
	}

	// Set patch headers
	patchReq.Header.Set("X-Auth", token)
	patchReq.Header.Set("Content-Type", "application/offset+octet-stream")
	patchReq.Header.Set("Content-Length", fmt.Sprint(fileInfo.Size()))
	patchReq.Header.Set("Upload-Offset", "0")

	// Send patch request
	patchResp, err := http.DefaultClient.Do(patchReq)
	if err != nil {
		return err
	}
	defer patchResp.Body.Close()

	// Check if status code is 204
	if patchResp.StatusCode != 204 {
		if patchResp.StatusCode == 409 {
			return errors.New("file already exists, use -override flag to override")
		}
		return errors.New("could not upload file")
	}

	return nil
}

type shareResponse struct {
	Hash   string `json:"hash"`
	Path   string `json:"path"`
	UserID int    `json:"userID"`
	Expire int    `json:"expire"`
}

func (e *filebrowser) share(token string) (shareResponse, error) {
	if e.Url == "" || e.File == "" {
		return shareResponse{}, errors.New("URL or file not provided")
	}

	// Create share request
	jsonBody := []byte(`{}`)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/share%s/%s+", e.Url, e.Directory, e.File), bytes.NewReader(jsonBody))
	if err != nil {
		return shareResponse{}, err
	}

	// Set headers
	req.Header.Set("X-Auth", token)
	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	req.Header.Set("Content-Length", "2")

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return shareResponse{}, err
	}
	defer resp.Body.Close()

	// Check if status code is 200
	if resp.StatusCode != 200 {
		return shareResponse{}, errors.New("could not share file")
	}

	// Get response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return shareResponse{}, err
	}

	// Unmarshal response
	var shareResp shareResponse
	if err := json.Unmarshal(body, &shareResp); err != nil {
		return shareResponse{}, err
	}

	return shareResp, nil
}
