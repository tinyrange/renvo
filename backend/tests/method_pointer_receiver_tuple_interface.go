package main

type tupleWriter interface {
	Write([]byte) (int, error)
}

type tupleSink struct {
	count int
}

func (sink *tupleSink) Write(data []byte) (int, error) {
	sink.count += len(data)
	return len(data), nil
}

func writeTuple(writer tupleWriter) (int, error) {
	count, err := writer.Write([]byte("PASS\n"))
	return count, err
}

func appMain() int {
	var sink tupleSink
	count, err := writeTuple(&sink)
	if err != nil || count != 5 || sink.count != 5 {
		return 1
	}
	print("PASS\n")
	return 0
}
