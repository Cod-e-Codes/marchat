package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"marchat/server"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var adminKey = flag.String("admin-key", "", "Admin key for privileged commands like /clear")
var adminUsername = flag.String("admin-username", "Cody", "The only user allowed to connect as 'admin'")

func printBanner(addr, adminUsername string) {
	fmt.Println(`
⢀⠀⠀⠀⠀⠀⠀⠀⢀⣠⣤⣶⣶⣶⣶⣶⣶⣶⣶⣶⣦⡀⠀⠀⠀⠀⠀⠀⣀⣀⣀⣀⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀  
⣿⣷⠀⠀⣀⣤⣴⣾⣿⡿⣿⣧⣿⣶⣿⣿⣿⣽⣿⣽⣿⣷⣤⣤⣴⣶⣾⣿⣿⡿⠿⠛⠛⠿⣷⡀⢀⣀⣀⣀⣀⡀⠀⠀⠀⠀  
⠈⣿⣶⣿⣿⣛⣿⣶⣿⣿⣿⣿⣛⣿⣭⣿⣽⣿⣹⣿⣻⣿⡿⠿⠛⠛⠋⠉⠀⢀⣀⣀⣀⣀⣈⣿⠿⠿⠟⠻⠿⢿⡇⠀⠀⠀  
⠀⢹⣿⣿⣿⣿⡟⣿⣯⣿⣿⣿⣿⢿⣿⢻⣟⣻⡟⢿⠿⣿⣇⣄⣠⣤⣴⣶⣾⣿⡿⠿⠿⠻⠿⠇⠀⣀⣀⣀⣀⣸⡇⠀⠀⠀  
⠀⠀⢻⣿⣿⣿⣿⡿⣿⡛⣿⣥⣿⣿⣿⣿⢿⡿⣿⣿⣷⣿⣿⢿⣿⠿⠟⠋⠉⠀⢀⣀⣀⣀⣀⡘⣿⣿⠿⠿⠿⢿⡇⠀⠀⠀  
⠀⠀⠈⣿⡏⢿⣿⣷⣾⣿⡟⢿⣋⣿⣴⣿⣾⣷⣿⣷⣾⣾⣿⡆⠀⣀⣤⣤⣶⣾⣿⠿⠟⠛⠻⠿⣇⣀⣠⣤⣤⣼⡇⠀⠀⠀  
⠀⠀⠀⠸⣿⡀⢻⣿⣯⣸⣷⣾⡿⠟⠋⠉⠀⠀⠀⠀⠀⠀⠀⠘⣿⣿⠿⠟⠛⠉⢀⣠⣤⣤⣤⣥⣽⡿⠿⠿⠿⠿⡇⠀⠀⠀  
⠀⠀⠀⠀⢻⣷⠀⢻⣿⠟⠋⠁⠀⢀⣠⣤⣴⣶⣶⣶⣶⣾⣶⣾⡁⣀⣀⣤⣴⣾⣿⠿⠛⠋⠉⠉⢳⣤⣤⣤⣤⣤⣷⠀⠀⠀  
⠀⠀⠀⠀⠈⣿⣇⠀⣿⣀⣠⣴⣾⣿⡿⠟⠋⠉⠉⠀⠀⠀⠈⠉⣿⠿⠟⠛⠋⠁⢀⣤⣶⣶⣶⣶⣾⠟⠛⠋⠉⠉⢿⠀⠀⠀  
⠀⠀⠀⠀⠀⠘⣿⡆⠸⣿⡿⠟⠋⠁⢀⣀⣤⣴⣶⣶⣶⣶⣶⣾⣇⣀⣤⣤⣶⣾⡿⠛⠋⠉⠉⠉⠉⣤⣴⣶⣶⠶⢿⣇⠀⠀  
⠀⠀⠀⠀⠀⠀⢹⣿⡀⣿⡄⣀⣤⣾⡿⠟⠋⠉⠉⠁⠀⠀⠀⠀⡿⠛⠛⠛⠉⠁⣀⣴⣾⣿⣿⣿⣿⡟⠉⠀⠀⣀⣀⣿⡀⠀  
⠀⠀⠀⠀⠀⠀⠀⢿⣧⢸⣿⠟⠋⠁⣀⣠⣤⣴⣶⣾⣿⣿⣿⣿⣧⣤⣤⣶⠾⠟⠛⠉⢁⣀⣀⣀⣀⢰⣶⣿⠿⠿⠿⠿⣧⠀  
⠀⠀⠀⠀⠀⠀⠀⠈⣿⡆⣿⣠⣴⣿⣿⣿⠿⠟⠛⠉⠉⠀⠀⢠⡟⠋⠉⣀⣠⣴⣾⠿⠿⠟⠛⠛⠛⣿⠏⠀⠀⣀⣀⣀⣿⡄  
⠀⠀⠀⠀⠀⠀⠀⠀⠸⣿⣿⡿⠟⠋⠁⠀⠀⠀⠀⠀⠀⠀⠀⠸⣧⣶⠿⠛⠋⠁⠀⠀⠀⠀⠀⠀⠘⣿⣴⣾⠿⠛⠋⠉⠉⠁  
⠀⠀⠀⠀⠀⠀⠀⠀⠀⢻⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠁⠀⠀⠀⠀⠀⠀⠀  
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⣿⣆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀  
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⣿⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀  
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀  

 
░███     ░███                                ░██                      ░██    
░████   ░████                                ░██                      ░██    
░██░██ ░██░██  ░██████   ░██░████  ░███████  ░████████   ░██████   ░████████ 
░██ ░████ ░██       ░██  ░███     ░██    ░██ ░██    ░██       ░██     ░██    
░██  ░██  ░██  ░███████  ░██      ░██        ░██    ░██  ░███████     ░██    
░██       ░██ ░██   ░██  ░██      ░██    ░██ ░██    ░██ ░██   ░██     ░██    
░██       ░██  ░█████░██ ░██       ░███████  ░██    ░██  ░█████░██     ░████ 

`)
	fmt.Printf("🌐 WebSocket: ws://%s/ws\n", addr)
	fmt.Printf("🔑 Admin HTTP: http://%s/clear\n", addr)
	fmt.Printf("👤 Only '%s' can connect as admin\n", adminUsername)
	fmt.Println("💡 Tip: Use --username admin --server ws://.../ws?real_user=YourName --admin-url http://... to connect as admin")
}

func main() {
	flag.Parse()
	db := server.InitDB("chat.db")
	server.CreateSchema(db)

	hub := server.NewHub()
	go hub.Run()

	http.HandleFunc("/ws", server.ServeWs(hub, db, *adminUsername))
	http.HandleFunc("/clear", server.ClearHandler(db, hub, *adminKey))

	addr := ":9090"
	serverAddr := "localhost:9090"
	log.Println("marchat WebSocket server running on", addr)
	printBanner(serverAddr, *adminUsername)

	// Create a custom server instance
	srv := &http.Server{Addr: addr}

	// Channel to listen for OS signals (Ctrl+C, etc.)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Run server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Block until we receive SIGINT (Ctrl+C)
	<-stop
	log.Println("Shutting down server gracefully...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Graceful shutdown failed: %v", err)
	}

	log.Println("Server shut down cleanly")
}
