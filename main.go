package main

func main() {

	defer enterRaw().leaveRaw()

	reactor := NewReactor()
	app := &app{}

	collectInput(reactor, app)

	reactor.Run()
}
