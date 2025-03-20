package main

import (
	"fmt"
	"net/http"
	"encoding/json"
	"github.com/alexflint/go-arg"
)

// Globals:
const CRTSH_BASE_URL := "https://crt.sh/?q=%s&output=json"

// Functions:
func main() {
	resp, err := http.Get("https://google.com")

	fmt.Printf("Response: %s, %s\n", resp, err)
}
