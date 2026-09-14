package tcpip

func ExampleHTTP_Configure() {
	var server HTTP
	mac := [6]byte{2, 0, 0, 0, 0, 1}
	ip := [4]byte{192, 168, 1, 50}
	server.Configure(mac, ip, 80)
	server.Status = HTTPStatus{IP: ip, MAC: mac}
	// Feed received Ethernet frames to Handle and transmit its replies.
	// Call Poll regularly to service connection timers.
}
