package main

func main() {
	app, err := InitServer()
	if err != nil {
		panic(err)
	}
	if err := app.Start(":8080"); err != nil {
		panic(err)
	}
}
