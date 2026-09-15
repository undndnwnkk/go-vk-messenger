package main

func main() {
	handler := handler.NewHandler()
	server := http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Println("error starting server: " + err.Error())
	}
}
