package main

import (
	"log"

	"example.org/golang/glalby"
)


func main() {
	log.Printf("equal: %v", glalby.Equal(1, 1))
	log.Printf("equal: %v", glalby.Equal(0, 1))
}