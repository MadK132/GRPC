package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	fmt.Println("Starting e-commerce microservices platform...")

	var wg sync.WaitGroup

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	wg.Add(1)
	go func() {
		defer wg.Done()
		cmd := exec.Command("go", "run", "cmd/gateway/main.go")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			fmt.Printf("Failed to start gateway: %v\n", err)
			return
		}
		fmt.Println("API Gateway started on port 8080")

		select {
		case <-sigs:
			if err := cmd.Process.Kill(); err != nil {
				fmt.Printf("Failed to kill gateway process: %v\n", err)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		cmd := exec.Command("go", "run", "cmd/inventory/main.go")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			fmt.Printf("Failed to start inventory service: %v\n", err)
			return
		}
		fmt.Println("Inventory Service started on port 8081")

		select {
		case <-sigs:
			if err := cmd.Process.Kill(); err != nil {
				fmt.Printf("Failed to kill inventory service process: %v\n", err)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		cmd := exec.Command("go", "run", "cmd/orders/main.go")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			fmt.Printf("Failed to start order service: %v\n", err)
			return
		}
		fmt.Println("Order Service started on port 8082")

		select {
		case <-sigs:
			if err := cmd.Process.Kill(); err != nil {
				fmt.Printf("Failed to kill order service process: %v\n", err)
			}
		}
	}()

	fmt.Println("All services started. Press Ctrl+C to shutdown gracefully.")

	<-sigs
	fmt.Println("Shutting down all services...")

	wg.Wait()
	fmt.Println("All services terminated. Goodbye!")
}
