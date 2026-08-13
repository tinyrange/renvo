package tca8418

import (
	"errors"
	"testing"
)

type transaction struct {
	write []byte
	read  []byte
	err   error
}

type fakeBus struct {
	transactions []transaction
	index        int
}

func (b *fakeBus) Tx(gotAddress uint16, write, read []byte) error {
	if gotAddress != address {
		return errors.New("wrong address")
	}
	if b.index >= len(b.transactions) {
		return errors.New("unexpected transaction")
	}
	want := b.transactions[b.index]
	b.index++
	if len(write) != len(want.write) {
		return errors.New("wrong write length")
	}
	for index := range write {
		if write[index] != want.write[index] {
			return errors.New("wrong write")
		}
	}
	copy(read, want.read)
	return want.err
}

func TestInitializeConfiguresCardputerMatrix(t *testing.T) {
	transactions := []transaction{}
	for register := byte(0x23); register <= 0x25; register++ {
		transactions = append(transactions, transaction{write: []byte{register, 0}})
	}
	for register := byte(0x20); register <= 0x22; register++ {
		transactions = append(transactions, transaction{write: []byte{register, 0xff}})
	}
	for register := byte(0x26); register <= 0x28; register++ {
		transactions = append(transactions, transaction{write: []byte{register, 0}})
	}
	for register := byte(0x11); register <= 0x13; register++ {
		transactions = append(transactions, transaction{write: []byte{register, 0xff}})
	}
	transactions = append(transactions,
		transaction{write: []byte{0x1d, 0x7f}},
		transaction{write: []byte{0x1e, 0xff}},
		transaction{write: []byte{0x04}, read: []byte{0x81}},
		transaction{write: []byte{0x04}, read: []byte{0}},
		transaction{write: []byte{0x02, 0x03}},
		transaction{write: []byte{0x01}, read: []byte{0x20}},
		transaction{write: []byte{0x01, 0x21}},
	)
	bus := &fakeBus{transactions: transactions}
	if err := New(bus).Initialize(); err != nil {
		t.Fatal(err)
	}
	if bus.index != len(transactions) {
		t.Fatalf("performed %d transactions, want %d", bus.index, len(transactions))
	}
}

func TestNextEvent(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x03}, read: []byte{1}},
		{write: []byte{0x04}, read: []byte{0x80 | 35}},
		{write: []byte{0x02, 0x01}},
		{write: []byte{0x03}, read: []byte{0}},
	}}
	device := New(bus)
	event, ok, err := device.NextEvent()
	if err != nil || !ok {
		t.Fatalf("NextEvent() = %#v, %t, %v", event, ok, err)
	}
	if event.Row != 3 || event.Column != 4 || !event.Pressed {
		t.Fatalf("event = %#v", event)
	}
	_, ok, err = device.NextEvent()
	if err != nil || ok {
		t.Fatalf("empty NextEvent() ok = %t, err = %v", ok, err)
	}
}

func TestNextEventRejectsUnwiredColumn(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x03}, read: []byte{1}},
		{write: []byte{0x04}, read: []byte{9}},
	}}
	_, ok, err := New(bus).NextEvent()
	if ok || err != ErrInvalidEvent {
		t.Fatalf("NextEvent() ok = %t, err = %v", ok, err)
	}
}
