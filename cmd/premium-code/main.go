package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/premium"
)

func main() {
	var (
		days  = flag.Int("days", 30, "masa berlaku kode (hari)")
		count = flag.Int("count", 1, "jumlah kode yang dibuat")
	)
	flag.Parse()

	if *days <= 0 {
		log.Fatal("days harus lebih dari 0")
	}
	if *count <= 0 || *count > 1000 {
		log.Fatal("count harus antara 1 sampai 1000")
	}

	expiry := time.Now().UTC().Add(time.Duration(*days) * 24 * time.Hour)

	for i := 0; i < *count; i++ {
		code, err := premium.BuildSecretCode(expiry)
		if err != nil {
			log.Fatalf("gagal membuat code: %v", err)
		}
		fmt.Println(code)
	}
}
