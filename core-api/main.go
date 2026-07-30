package main

func main() {
	if err := InitServer().Start(":8080"); err != nil {
		panic(err)
	}
}
