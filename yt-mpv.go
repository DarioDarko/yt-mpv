package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"time"
)

type PlayRequest struct {
	URL string `json:"url"`
}

type SeekRequest struct {
	Seconds int `json:"seconds"`
}

type mpvRequest struct {
	Command []string `json:"command"`
}

type mpvResponse struct {
	Data  float64 `json:"data"`
	Error string  `json:"error"`
}

var (
	Player string
	Display string
	Socket = "/tmp/yt-mpv.socket"
)

func playHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		log.Fatal("Only POST allowed", http.StatusMethodNotAllowed)
	}

	var req PlayRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil || req.URL == "" {
		log.Fatal("Invalid JSON or missing URL", http.StatusBadRequest)
	}

	checkCmd := exec.Command("pgrep", "-x", Player)
	err = checkCmd.Run()

	if err == nil {
		log.Println(Player, "is already running")
		log.Println("Changing video to", req.URL)

		conn, err := net.Dial("unix", Socket)

		if err != nil {
			panic(err)
		}

		defer conn.Close()

		loadCmd := fmt.Sprintf("{ \"command\": [\"loadfile\", \"%s\"] }\" }\n", req.URL)
		_, err = conn.Write([]byte(loadCmd))
		
		if err != nil {
			log.Fatal("Failed to change video:", err)
		}

		playCmd := "{ \"command\": [\"set_property\", \"pause\", false] }\n"
		_, err = conn.Write([]byte(playCmd))

		if err != nil {
			log.Fatal("Failed to continue video:", err)
		}
	} else {
		log.Println(Player, "is not running")
		log.Println("Starting", Player, "with video", req.URL)

		var args []string

		if Player == "mpvpaper" {
			args = append(args, "-o")
			args = append(args, "--input-ipc-server=/tmp/yt-mpv.socket")
			args = append(args, Display)
		} else {
			args = append(args, "--keep-open")
			args = append(args, "--input-ipc-server=/tmp/yt-mpv.socket")
		}

		args = append(args, req.URL)

		cmd := exec.Command(Player, args...)
		cmd.Run()
	}

	w.WriteHeader(http.StatusOK)
}

func playPauseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conn, err := net.Dial("unix", Socket)

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	_, err = conn.Write([]byte("cycle pause\n"))

	if err != nil {
		http.Error(w, "Failed to send command", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func seekHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SeekRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	conn, err := net.Dial("unix", Socket)

	if err != nil {
		return
	}

	defer conn.Close()

	seekCmd := fmt.Sprintf("seek %d absolute\n", int(req.Seconds))
	_, err = conn.Write([]byte(seekCmd))

	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func timeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	conn, err := net.Dial("unix", Socket)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect mpv socket: %v", err), http.StatusInternalServerError)
		return
	}

	defer conn.Close()

	req := mpvRequest{Command: []string{"get_property", "time-pos"}}
	reqBytes, err := json.Marshal(req)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal request: %v", err), http.StatusInternalServerError)
		return
	}

	_, err = conn.Write(append(reqBytes, '\n'))

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write to socket: %v", err), http.StatusInternalServerError)
		return
	}

	conn.SetReadDeadline(time.Now().Add(1 * time.Second))

	respBytes := make([]byte, 1024)
	n, err := conn.Read(respBytes)

	if err != nil && err != io.EOF {
		http.Error(w, fmt.Sprintf("Failed to read from socket: %v", err), http.StatusInternalServerError)
		return
	}

	var resp mpvResponse
	err = json.Unmarshal(respBytes[:n], &resp)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse mpv response: %v", err), http.StatusInternalServerError)
		return
	}

	if resp.Error != "success" {
		http.Error(w, "mpv response error: "+resp.Error, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]float64{"time": resp.Data})
}

func main() {
	var port int

	flag.IntVar(&port, "port", 54345, "Port to listen on")
	flag.StringVar(&Player, "player", "mpv", "Player to launch")
	flag.StringVar(&Display, "display", "", "Display for mpvpaper")
	flag.Parse()

	if Player == "mpvpaper" && Display == "" {
		log.Fatal("Display must be set for mpvpaper")
	}

	addr := fmt.Sprintf("localhost:%d", port)

	http.HandleFunc("/play", playHandler)
	http.HandleFunc("/play-pause", playPauseHandler)
	http.HandleFunc("/seek", seekHandler)
	http.HandleFunc("/time", timeHandler)

	log.Printf("Starting server on http://%s\n", addr)

	err := http.ListenAndServe(addr, nil)

	if err != nil {
		log.Fatal(err)
	}
}
