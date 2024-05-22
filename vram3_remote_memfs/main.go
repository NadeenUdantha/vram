package main

func main() {
	select {}
}

func assert(x bool) {
	if !x {
		panic("?")
	}
}

func throw(err error) {
	if err != nil {
		panic(err)
	}
}
