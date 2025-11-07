package main

import (
	"fmt"
	"log"
	"omniapi/adapters"
)

func main() {
	fmt.Println("🧪 Testing OmniAPI Connectors Registration...")

	// Registrar adaptadores
	if err := adapters.RegisterAllAdapters(); err != nil {
		log.Fatalf("❌ Failed to register adapters: %v", err)
	}

	fmt.Println("✅ All adapters registered successfully!")
	fmt.Println("🏁 Registration test completed!")
}
