package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	port := "8080"
	fmt.Printf("Serveur démarré sur http://localhost:%s\n", port)
	
	// Sert les fichiers du répertoire parent (la racine du projet)
	fs := http.FileServer(http.Dir("."))
	
	// On utilise un handler personnalisé pour s'assurer que le type MIME du WASM est correct
	// (certains navigateurs sont capricieux)
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		fs.ServeHTTP(w, r)
	}))

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
